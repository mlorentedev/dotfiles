#!/bin/sh
#
# lane-guard.sh — Scoped pre-commit guard for persistent named lanes (HARNESS-186).
#
# POSIX sh on purpose: this runs under bash, zsh, dash and BusyBox alike, on
# Linux, macOS and Git-for-Windows. No bash-only expansion is permitted here.
#
# Opt-in is the committed marker .dotf/lanes.json, written by `dotf lane init`
# and located in the current worktree or the main repository root. The marker
# carries no paths and no host names, so it is identical on every OS, agent host
# and seat, and no other feature's config file can switch this guard on.
#
# It exempts:
#   - Obsidian vaults (.obsidian/ present)
#   - Active git rebase or cherry-pick operations (rebase-merge, rebase-apply,
#     or CHERRY_PICK_HEAD present in git-dir, fixing B6)
#
# In a repository carrying .dotf/lanes.json, it blocks:
#   1. Direct commits on main/master at the main checkout (directs to `dotf lane switch`)
#   2. Commits inside a PARKED persistent lane (.dotf-worktree.json state == "PARKED")
#   3. Commits inside a registered lane that carries no lease (.dotf-worktree.json
#      absent). A lane is recognised by the `dotf-lane` record `dotf lane init`
#      leaves in the worktree's own git admin dir ($git_dir), never by its
#      directory name: git moves and prunes that record with the worktree, so a
#      renamed lane stays a lane and a stranger named after one is not.
#   4. Commits inside a declared third-party session root (third_party_roots)

set -u

top="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
[ -n "$top" ] || exit 0

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

# 3. Only enforce where the repository declares the named lane system.
marker=""
for candidate in "$top/.dotf/lanes.json" "$main_root/.dotf/lanes.json"; do
  if [ -f "$candidate" ]; then
    marker="$candidate"
    break
  fi
done
[ -n "$marker" ] || exit 0

# json_list prints the string elements of a named array, one per line. sed is
# line-based and `dotf lane init` writes indented JSON, where every array element
# sits on its own line, so the document is flattened before matching. Names are
# kebab-case identifiers, so the caller may word-split the result safely.
json_list() {
  tr -d '\r\n' <"$marker" 2>/dev/null |
    sed -n "s/.*\"$1\"[[:space:]]*:[[:space:]]*\[\([^]]*\)\].*/\1/p" |
    tr ',' '\n' |
    sed -e 's/[[:space:]"]//g' -e '/^$/d'
}

# The marker is the single source of truth: a list it does not declare is a rule
# that does not apply. No session host is hardcoded here, so a repository opts
# into exactly the roots it declares. The marker's lane list is policy for
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

# 5. Block commits inside a PARKED lane or a registered lane without a lease.
if [ -f "$top/.dotf-worktree.json" ]; then
  if grep -Eq '"state"[[:space:]]*:[[:space:]]*"PARKED"' "$top/.dotf-worktree.json" 2>/dev/null; then
    echo "ERROR: lane at $top is PARKED (detached idle state)." >&2
    echo "FIX: activate the lane on a feature branch first: dotf lane switch <type>/<issue>-<slug>" >&2
    exit 1
  fi
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
    echo "ERROR: lane '$lane_name' at $top is missing .dotf-worktree.json lease metadata." >&2
    echo "FIX: restore the lease and switch the lane with: dotf lane init && dotf lane switch <type>/<issue>-<slug>" >&2
    exit 1
  fi
fi

# 6. Block direct commits on main/master or detached HEAD at the main repository checkout.
branch="$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)"
if [ "$branch" = "main" ] || [ "$branch" = "master" ]; then
  echo "ERROR: direct commits on '$branch' are blocked in lane-managed repository ($top)." >&2
  echo "FIX: use a persistent named lane instead: dotf lane switch <type>/<issue>-<slug>" >&2
  exit 1
fi
if [ -z "$branch" ] && [ ! -f "$top/.dotf-worktree.json" ]; then
  # Same verdict at the main checkout and in a leaseless linked worktree; the
  # message names which one was detected, since the old one sent a user in a
  # scratch worktree to "the main checkout" (#3324).
  if [ "$git_dir" = "$common_dir" ]; then
    echo "ERROR: detached HEAD commits at the main checkout are blocked in lane-managed repository ($top)." >&2
  else
    echo "ERROR: detached HEAD without a lease at $top is blocked in lane-managed repository ($main_root)." >&2
  fi
  echo "FIX: use a persistent named lane instead: dotf lane switch <type>/<issue>-<slug>" >&2
  exit 1
fi

exit 0
