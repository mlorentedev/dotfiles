---
title: "A merged PR does not mean its branch is merged"
date: "2026-10-06"
---

# A merged PR does not mean its branch is merged

## Context
A pre-migration audit (2026-10-06) listed leftover branches to delete before moving to a new
workstation. One entry read: "remote branch `chore/nan-key-pr-agent-consumer`: PR #1768 merged →
deletable". The audit read PR state, which is the usual test for "this branch is done".

## The Trap
`a4e75c10` had been pushed to that branch about 30 minutes after #1768 merged. GitHub does not
reopen or update a merged PR when its branch moves, so the PR still said MERGED while the branch
carried a commit that never reached `main`. The commit fixed a comment in `secrets/registry.yaml`
whose wording on `main` dictated an `sh -c '... toolkit secrets set ...'` command that `dotf`
itself refuses. Deleting the branch on the strength of the PR state would have lost the only copy
of the fix, and nothing would have reported it: the audit would have finished clean.

A squash or rebase merge makes the obvious ancestry test useless too. Every squash-merged
branch reports commits "not in main", so `git log origin/main..<branch>` alone cannot separate a
finished branch from a stranded one.

## The Solution
Compare the branch tip with the PR's **head commit at merge**, not with the PR's state:

```bash
head=$(gh pr view <N> --json headRefOid -q .headRefOid)
git merge-base --is-ancestor <branch> "$head" && echo "nothing after the PR" \
  || git log --oneline "$head..<branch>"        # commits pushed after the merge
```

When the branch has commits beyond the PR head, check their content against `main` (for a
squash merge, reverse-apply the branch's diff against `main`'s tree with
`git apply --cached --check -R`). Land anything missing through a new PR before deleting the
branch. The same test applies to local branches, stray refs (`refs/tmp/*`, `refs/remotes/pr/*`)
and worktrees: "its PR merged" is a claim about the PR, not about the commits on the ref.
