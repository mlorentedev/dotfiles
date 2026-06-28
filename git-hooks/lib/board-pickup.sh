#!/usr/bin/env bash
#
# board-pickup.sh — self-assign the linked issue at branch pickup (knowledge#140).
#
# The forcing-function: a real mechanical trigger (branch creation = start of work) for the
# step that actually gets forgotten — self-assigning the ticket. It does NOT move the board
# itself: assigning fires the existing bitacora-status Action (HARNESS-010, on
# issues:[assigned]) which flips the item to In Progress. Single source of truth for the
# status transition stays the Action; this hook only automates the self-assign that precedes it.
#
# Called by the global post-checkout dispatcher, BACKGROUNDED and SILENCED — so it must be
# entirely fail-silent: any missing tool/auth/issue is a no-op, never an error, never slow.
# `gh issue edit --add-assignee @me` is idempotent: GitHub fires the `assigned` event only when
# an assignee is newly added, so re-checking-out an already-picked-up branch does nothing.
#
# Resolution (ADR-018): the branch carries an issue number; try the CURRENT repo first, then the
# knowledge bitácora home (where foreign-repo tickets live). POSIX-friendly bash.

set -u

FALLBACK_OWNER="mlorentedev"
FALLBACK_REPO="knowledge"

# git passes: $1 = prev HEAD, $2 = new HEAD, $3 = branch-checkout flag (1 = branch checkout).
flag="${3:-0}"
[ "$flag" = "1" ] || exit 0                       # file checkout, not a branch switch → ignore

command -v gh >/dev/null 2>&1 || exit 0

branch="$(git symbolic-ref --quiet --short HEAD 2>/dev/null)" || exit 0
[ -n "$branch" ] || exit 0

# Issue number = leading digits after the type prefix: feat/140-slug, fix/52-x, chore/7-y.
issue="$(printf '%s' "$branch" | sed -n 's#^[a-z]\{1,12\}/\([0-9]\{1,7\}\)-.*#\1#p')"
[ -n "$issue" ] || exit 0

# Current repo owner/name from origin (best-effort; may be empty for a local-only repo).
origin_url="$(git config --get remote.origin.url 2>/dev/null || true)"
cur_owner="$(printf '%s' "$origin_url" | sed -E 's#^.*[:/]([^/]+)/[^/]+$#\1#; s#\.git$##')"
cur_repo="$(printf '%s'  "$origin_url" | sed -E 's#^.*[:/][^/]+/([^/]+)$#\1#; s#\.git$##')"

# Self-assign the issue, only if it is OPEN — never resurrect a closed/Done ticket by assigning
# it. `gh issue edit --add-assignee @me` is idempotent at the GitHub event layer.
try_assign() {
  local owner="$1" repo="$2" num="$3" state
  [ -n "$owner" ] && [ -n "$repo" ] || return 1
  state="$(gh issue view "$num" --repo "$owner/$repo" --json state -q .state 2>/dev/null)" || return 1
  [ "$state" = "OPEN" ] || return 1
  gh issue edit "$num" --repo "$owner/$repo" --add-assignee @me >/dev/null 2>&1
}

# Current repo first, then the knowledge bitácora home (ADR-018 foreign tickets).
try_assign "$cur_owner" "$cur_repo" "$issue" && exit 0
[ "$cur_repo" = "$FALLBACK_REPO" ] || try_assign "$FALLBACK_OWNER" "$FALLBACK_REPO" "$issue"
exit 0
