---
id: "lesson-355-after-a-squash-merge-ask-the-merged-head-by-containment"
type: lesson
status: active
title: "After a squash merge, ask the merged head by containment"
created: "2026-10-07"
---

# After a squash merge, ask the merged head by containment

## Context
`dotf worktree done` refused every squash-merged branch whose remote GitHub had deleted. The
refusal read "N unpushed commit(s) ahead of origin/HEAD", so operators reached for `--force`
(#1608). That is the flag that would also discard real unpushed work.

## The Trap
Squash merging puts a new commit on the base. The branch's own commits never become ancestors
of the base, so `rev-list base..HEAD`, `merge-base --is-ancestor` and `git branch --merged` all
report the branch as unmerged forever. The obvious repair compares the local `HEAD` with the
merged PR's `headRefOid`. It is too strict. In six worktrees reaped by hand on 2026-09-27, the
local branch was *behind* its PR head in every case, because the PR had taken a rebase or a merge
of main from another checkout. "The PR merged", checked by branch name, is too loose: it accepts a
commit made on the branch after the merge.

## The Solution
Ask whether the tip is **contained** in the head of a merged pull request for the branch:
`merge-base --is-ancestor HEAD <headRefOid>`. If the head object is missing locally, fetch it by
SHA first. A checkout that is behind its PR head passes. A follow-up commit after the merge fails,
because no merged head contains it. When the PR cannot be listed (no gh, offline), the check fails
closed, and the refusal says why. Put `done`, `list` and `sweep` behind one predicate, so all
three give the same answer.
