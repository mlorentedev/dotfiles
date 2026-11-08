#!/usr/bin/env bash
#
# lane-guard.sh — Scoped pre-commit guard for persistent named lanes (HARNESS-186).
#
# Only enforces lane rules in repositories that opt into the lane system via
# .dotf/lane.json (located in the current worktree or main repository root),
# and exempts:
#   - Obsidian vaults (.obsidian/ present)
#   - Active git rebase or cherry-pick operations (rebase-merge, rebase-apply,
#     or CHERRY_PICK_HEAD present in git-dir, fixing B6)
#
# In a .dotf/lane.json repository, blocks:
#   1. Direct commits on main/master at the main checkout (directs to `dotf lane switch`)
#   2. Commits inside a PARKED persistent lane (.dotf-worktree.json state == "PARKED")
#   3. Commits inside unleased copilot-worktrees/ directories (fixing m4)

set -u

top="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
[ -n "$top" ] || exit 0

git_dir="$(git rev-parse --git-dir 2>/dev/null)" || exit 0
common_dir="$(git rev-parse --git-common-dir 2>/dev/null)" || common_dir="$git_dir"
case "$common_dir" in
  /*|[A-Za-z]:/*|[A-Za-z]:\\*) ;;
  *) common_dir="$top/$common_dir" ;;
esac
common_slash="${common_dir//\\//}"
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

# 3. Only enforce in repositories with .dotf/lane.json.
if [ ! -f "$top/.dotf/lane.json" ] && [ ! -f "$main_root/.dotf/lane.json" ]; then
  exit 0
fi

# 4. Block commits in copilot-worktrees/ for lane-managed repos (m4).
top_slash="${top//\\//}"
case "$top_slash" in
  */copilot-worktrees/*)
    echo "ERROR: commits inside copilot-worktrees/ ($top) are blocked in lane-managed repositories." >&2
    echo "FIX: claim a persistent named lane with: dotf lane switch <branch>" >&2
    exit 1
    ;;
esac

# 5. Block commits inside a PARKED lane or an uninitialized canonical lane directory.
if [ -f "$top/.dotf-worktree.json" ]; then
  if grep -Eq '"state"[[:space:]]*:[[:space:]]*"PARKED"' "$top/.dotf-worktree.json" 2>/dev/null; then
    echo "ERROR: lane at $top is PARKED (detached idle state)." >&2
    echo "FIX: activate the lane on a feature branch first: dotf lane switch <type>/<issue>-<slug>" >&2
    exit 1
  fi
else
  lane_base="${top_slash##*/}"
  case "$lane_base" in
    feat-lane-primary|feat-lane-secondary|fast-track-lane)
      echo "ERROR: canonical lane directory $top is missing .dotf-worktree.json lease metadata." >&2
      echo "FIX: initialize and switch the lane with: dotf lane init && dotf lane switch <type>/<issue>-<slug>" >&2
      exit 1
      ;;
  esac
fi

# 6. Block direct commits on main/master or detached HEAD at the main repository checkout.
branch="$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)"
if [ "$branch" = "main" ] || [ "$branch" = "master" ]; then
  echo "ERROR: direct commits on '$branch' are blocked in lane-managed repository ($top)." >&2
  echo "FIX: use a persistent named lane instead: dotf lane switch <type>/<issue>-<slug>" >&2
  exit 1
fi
if [ -z "$branch" ] && [ ! -f "$top/.dotf-worktree.json" ]; then
  echo "ERROR: detached HEAD commits at the main checkout are blocked in lane-managed repository ($top)." >&2
  echo "FIX: use a persistent named lane instead: dotf lane switch <type>/<issue>-<slug>" >&2
  exit 1
fi

exit 0
