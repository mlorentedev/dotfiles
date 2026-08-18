---
id: lesson-079-a-late-push-to-a-pr-branch-can-miss-the-squash-ver
type: lesson
status: active
created: "2026-06-03"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 079: A late push to a PR branch can miss the squash — verify the commit is on the PR head, and each deliverable is on main post-merge

**Context:** AI-020. Opened PR #216 with the decision commit, then pushed a SECOND commit (the migration runbook, AC6) to the same branch. The user clicked "Update branch" in the GitHub UI and squash-merged.
**Problem:** The squash captured only what the PR head had at merge time — the decision + AI-023 scaffold — NOT the runbook. The UI "Update branch" + concurrent merge consolidated a head that raced my runbook push, so the runbook silently never reached main. Nothing errored; AC6 was just absent. Caught only by an explicit post-merge `git cat-file -e origin/main:docs/runbooks/...` check, AFTER I'd already torn down the worktree + branch (`-D`).
**Solution:** After adding a commit to a PR that may merge imminently — especially when the other party is merging concurrently — confirm it actually landed on the PR head (`gh pr view <n> --json commits` shows the SHA) before treating it as shipped. When closing a multi-commit PR, verify each deliverable file exists on `origin/main` post-merge; don't assume the squash took everything. Recovery is cheap because the orphaned commit survives in the local object store (`git cat-file -e <sha>` → cherry-pick onto fresh main), but only if you notice — so delay `git branch -D` of a just-merged branch until the deliverables are confirmed on main.
**Tags:** `#git` `#pr` `#squash-merge` `#verify-before-completion` `#race` `#red-team-thyself`
