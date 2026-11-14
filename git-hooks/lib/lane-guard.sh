#!/bin/sh
#
# lane-guard.sh — Scoped pre-commit guard for persistent named lanes (HARNESS-186).
#
# POSIX sh on purpose: this runs under bash, zsh, dash and BusyBox alike, on
# Linux, macOS and Git-for-Windows. No bash-only expansion is permitted here.
#
# Opt-in is the activation record `dotf converge --only lanes` writes into the
# git common dir (dotf-lanes-active), and nothing else arms this guard (#3437). A
# marker in a working tree, committed or not, arms nothing: otherwise every
# checkout of the default branch, the deploy clone included, would arm itself.
# The record carries the committed marker's content, which declares no paths and
# no host names, so it is identical on every OS, agent host and seat, and no
# other feature's config file can switch this guard on.
#
# It exempts:
#   - Obsidian vaults (.obsidian/ present)
#   - Active git rebase or cherry-pick operations (rebase-merge, rebase-apply,
#     or CHERRY_PICK_HEAD present in git-dir, fixing B6)
#
# In a repository holding the activation record, it blocks:
#   1. Direct commits on main/master, at the main checkout or in a linked worktree
#      that holds it (each message names which; both direct to a branch or lane)
#   2. Commits inside a PARKED persistent lane (lease state == "PARKED")
#   3. Commits inside a registered lane that carries no lease
#      ($git_dir/dotf-worktree.json absent, and no legacy in-tree copy). A lane is recognised by the `dotf-lane` record `dotf lane init`
#      leaves in the worktree's own git admin dir ($git_dir), never by its
#      directory name: git moves and prunes that record with the worktree, so a
#      renamed lane stays a lane and a stranger named after one is not.
#   4. Commits inside a declared third-party session root (third_party_roots)
#
# It warns, and never blocks, when the worktree's lease has expired (#3438).

set -u

top="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
[ -n "$top" ] || exit 0

# git answers --git-dir absolute but --git-common-dir relative to the cwd when
# run from a subdirectory, and the two are made absolute against $top below.
# Standing at $top makes every relative answer mean the same directory.
cd "$top" || exit 0

git_dir="$(git rev-parse --git-dir 2>/dev/null)" || exit 0
case "$git_dir" in
  /*|[A-Za-z]:/*|[A-Za-z]:\\*) ;;
  *) git_dir="$top/$git_dir" ;;
esac
common_dir="$(git rev-parse --git-common-dir 2>/dev/null)" || common_dir="$git_dir"
case "$common_dir" in
  /*|[A-Za-z]:/*|[A-Za-z]:\\*) ;;
  *) common_dir="$top/$common_dir" ;;
esac
common_slash="$(printf '%s' "$common_dir" | sed 's|\\|/|g')"
common_slash="${common_slash%/}"
if [ "${common_slash##*/}" = ".git" ]; then
  main_root="$(dirname "$common_dir")"
else
  main_root="$top"
fi

# 1. Exempt active git rebase, cherry-pick, revert, or sequencer (B6, m-CODE-4).
if [ -d "$git_dir/rebase-merge" ] || [ -d "$git_dir/rebase-apply" ] || [ -d "$git_dir/sequencer" ] || [ -f "$git_dir/CHERRY_PICK_HEAD" ] || [ -f "$git_dir/MERGE_HEAD" ] || [ -f "$git_dir/REVERT_HEAD" ]; then
  exit 0
fi

# 2. Exempt Obsidian vaults (.obsidian directory in worktree or main root).
if [ -d "$top/.obsidian" ] || [ -d "$main_root/.obsidian" ]; then
  exit 0
fi

# 3. Only enforce where this machine activated the lanes: the record exists in
# the git common dir, which every worktree of the repository shares and which
# `git clean` never reaches. Only the file's existence arms the guard; its
# content is data for rule 4. A different clone, such as the deploy clone, has a
# common dir of its own and so no record.
record="$common_dir/dotf-lanes-active"
[ -f "$record" ] || exit 0

# json_list prints the string elements of a named array, one per line. sed is
# line-based and the marker is indented JSON, where every array element sits on
# its own line, so the document is flattened before matching. Names are
# kebab-case identifiers, so the caller may word-split the result safely.
json_list() {
  tr -d '\r\n' <"$record" 2>/dev/null |
    sed -n "s/.*\"$1\"[[:space:]]*:[[:space:]]*\[\([^]]*\)\].*/\1/p" |
    tr ',' '\n' |
    sed -e 's/[[:space:]"]//g' -e '/^$/d'
}

# The record is the single source of truth: a list it does not declare is a rule
# that does not apply. No session host is hardcoded here, so a repository opts
# into exactly the roots its marker declares. The lane list is policy for
# `dotf lane init`, not an identity: this guard never matches a lane by name.
third_party_roots="$(json_list third_party_roots)"

# 4. Block commits inside a declared third-party session root (m4).
top_slash="$(printf '%s' "$top" | sed 's|\\|/|g')"
for root in $third_party_roots; do
  case "$top_slash" in
    */"$root"/*)
      echo "ERROR: commits inside the third-party session root '$root' ($top) are blocked in lane-managed repositories." >&2
      echo "FIX: claim a persistent named lane with: dotf lane switch <branch>" >&2
      exit 1
      ;;
  esac
done

# epoch_of prints the Unix time of an RFC 3339 timestamp, as dotf writes it into
# the lease (local zone: "Z" or an offset, optional fraction), or nothing when it
# does not parse. Pure arithmetic: `date -d` is GNU-only and `date -j` BSD-only.
# The days come from the proleptic Gregorian calendar (Hinnant's days_from_civil).
epoch_of() {
  parts="$(printf '%s' "$1" | sed -nE 's/^([0-9]{4})-([0-9]{2})-([0-9]{2})[Tt]([0-9]{2}):([0-9]{2}):([0-9]{2})(\.[0-9]+)?(Z|z|[+-][0-9]{2}:[0-9]{2})$/\1 \2 \3 \4 \5 \6 \8/p')"
  [ -n "$parts" ] || return 0
  IFS=' ' read -r ey em ed eh emi es ez <<PARTS
$parts
PARTS
  # Leading zeros would read as octal.
  em="${em#0}"; ed="${ed#0}"; eh="${eh#0}"; emi="${emi#0}"; es="${es#0}"
  [ "$ey" -ge 1970 ] || return 0
  ey=$((ey - (em <= 2)))
  eera=$((ey / 400))
  eyoe=$((ey - eera * 400))
  edoy=$(((153 * (em > 2 ? em - 3 : em + 9) + 2) / 5 + ed - 1))
  edays=$((eera * 146097 + eyoe * 365 + eyoe / 4 - eyoe / 100 + edoy - 719468))
  eoff=0
  case "$ez" in
    [+-]*)
      eoh="${ez#?}"
      eom="${eoh##*:}"
      eoh="${eoh%%:*}"
      eoh="${eoh#0}"; eom="${eom#0}"
      eoff=$((${eoh:-0} * 3600 + ${eom:-0} * 60))
      [ "${ez%"${ez#?}"}" = "-" ] && eoff=$((0 - eoff))
      ;;
  esac
  echo $((edays * 86400 + ${eh:-0} * 3600 + ${emi:-0} * 60 + ${es:-0} - eoff))
}

# warn_if_lease_expired warns, and never blocks, when the lease has run out: the
# work it claimed may be over, and sweep may reap or park the worktree under
# whoever is committing in it (#3438). A lease without an expiry, or whose expiry
# is the zero time, has not expired.
warn_if_lease_expired() {
  expires="$(sed -n 's/.*"lease_expires_at"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$lease" 2>/dev/null | head -n 1)"
  [ -n "$expires" ] || return 0
  expires_at="$(epoch_of "$expires")"
  [ -n "$expires_at" ] || return 0
  if [ "$expires_at" -le "$(date +%s)" ]; then
    echo "WARN: $top: lease expired $expires, so sweep may reap or park it under whoever works there." >&2
    echo "FIX: dotf worktree done $top once its branch landed (a lane: dotf lane park <lane>)" >&2
  fi
}

# The lease lives in the worktree's own git admin dir, $git_dir/dotf-worktree.json
# (ADR-062), which `git clean` never reaches; a lease a binary before the move
# left in the tree still counts, for one release. Rules 5 and 6 read this one
# resolved path, in the order the Go side reads it.
lease=""
if [ -f "$git_dir/dotf-worktree.json" ]; then
  lease="$git_dir/dotf-worktree.json"
elif [ -f "$top/.dotf-worktree.json" ]; then
  lease="$top/.dotf-worktree.json"
fi

# 5. Block commits inside a PARKED lane or a registered lane without a lease.
if [ -n "$lease" ]; then
  if grep -Eq '"state"[[:space:]]*:[[:space:]]*"PARKED"' "$lease" 2>/dev/null; then
    echo "ERROR: lane at $top is PARKED (detached idle state)." >&2
    echo "FIX: activate the lane on a feature branch first: dotf lane switch <type>/<issue>-<slug>" >&2
    exit 1
  fi
  warn_if_lease_expired
elif [ -f "$git_dir/dotf-lane" ]; then
  # A record that exists but cannot be read is not "no record": fail closed.
  if [ ! -r "$git_dir/dotf-lane" ]; then
    echo "ERROR: lane identity record $git_dir/dotf-lane is not readable." >&2
    echo "FIX: restore it with: dotf lane init" >&2
    exit 1
  fi
  # An empty record is no record, as lane.Identity reads it in Go; both sides
  # read the first line only.
  lane_name="$(sed -n '1s/^[[:space:]]*//;1s/[[:space:]]*$//;1p' "$git_dir/dotf-lane" 2>/dev/null)"
  if [ -n "$lane_name" ]; then
    echo "ERROR: lane '$lane_name' at $top is missing its lease metadata ($git_dir/dotf-worktree.json)." >&2
    echo "FIX: restore the lease and switch the lane with: dotf lane init && dotf lane switch <type>/<issue>-<slug>" >&2
    exit 1
  fi
fi

# 6. Block direct commits on main/master, and detached HEAD without a lease, in the
# main checkout or in any linked worktree. The wording follows what was detected.
branch="$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)"
if [ "$branch" = "main" ] || [ "$branch" = "master" ]; then
  # `$git_dir = $common_dir` is what makes a checkout the main one. A linked
  # worktree can hold main while the main checkout sits on another branch, and
  # must not be told it is "the main checkout".
  if [ "$git_dir" = "$common_dir" ]; then
    echo "ERROR: direct commits on '$branch' at the main checkout are blocked in lane-managed repository ($top)." >&2
    echo "FIX: use a persistent named lane instead: dotf lane switch <type>/<issue>-<slug>" >&2
  else
    echo "ERROR: direct commits on '$branch' in the linked worktree $top are blocked in lane-managed repository ($main_root)." >&2
    echo "FIX: leave '$branch' here: git switch -c <type>/<issue>-<slug>, or use a persistent named lane: dotf lane switch <type>/<issue>-<slug>" >&2
  fi
  exit 1
fi
if [ -z "$branch" ] && [ -z "$lease" ]; then
  # Same verdict at the main checkout and in a leaseless linked worktree; the
  # message names which one was detected, since the old one sent a user in a
  # scratch worktree to "the main checkout" (#3324).
  if [ "$git_dir" = "$common_dir" ]; then
    echo "ERROR: detached HEAD commits at the main checkout are blocked in lane-managed repository ($top)." >&2
    echo "FIX: use a persistent named lane instead: dotf lane switch <type>/<issue>-<slug>" >&2
  else
    # A linked worktree is no lane: `lane switch` claims a different worktree
    # and leaves this one as it is. Adopting keeps it and its commits; a
    # branch ends the detached state (#3436).
    echo "ERROR: detached HEAD without a lease at $top is blocked in lane-managed repository ($main_root)." >&2
    echo "FIX: keep this worktree detached with a lease: dotf worktree adopt" >&2
    echo "     or leave the detached state: git switch -c <type>/<issue>-<slug>" >&2
  fi
  exit 1
fi

exit 0
