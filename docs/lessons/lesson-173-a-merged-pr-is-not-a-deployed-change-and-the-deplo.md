---
id: lesson-173-a-merged-pr-is-not-a-deployed-change-and-the-deplo
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 173: A merged PR is not a deployed change, and the deploy takes whatever branch the checkout happens to be on

**Context**: #840 fixed a dispatcher bug that aborted `git commit` in every repo on the machine. It merged, `main` was released as 0.33.1, and the deploy was re-run. Everything reported success.

**Problem**: The fix had not reached the machine. `git commit` still failed in a clean throwaway repo, and `grep --hook-dir ~/.dotfiles/git-hooks/lib/chain-local-hook.sh` returned zero — with an mtime from minutes earlier, so the deploy *had* run and had faithfully copied the old file. The checkout was still on `chore/sync-harness-skill-records`, the branch of an unrelated PR merged an hour before, four commits behind `main`. `setup-linux.sh` deploys from the working tree it is invoked in; it has no opinion about whether that tree is current. Green CI, a merged PR and a cut release together say nothing about what is installed.

**Solution**: `git checkout main && git pull` before deploying, then verify in two tiers — the repo tier (`grep` the source) and the machine tier (`grep` the deployed copy plus an end-to-end probe: a real `git commit` in a temp repo). The end-to-end probe is what actually closed it; a file comparison alone would have missed a deploy that copied the right bytes to the wrong place.

**Rule**: The deploy's input is the working tree, not the branch you believe you are on — so "merged" and "deployed" are independent facts and only the second one runs on your machine. Never close a fix on the strength of a merge; re-check the branch, then assert against the installed artifact and, where the fix has an observable behaviour, against the behaviour itself. This is the same two-tier discipline as the dotfiles deploy rule, with the branch check added: the tier-1 source can be correct and still be the wrong source.
