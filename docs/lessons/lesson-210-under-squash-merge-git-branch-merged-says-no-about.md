---
id: lesson-210-under-squash-merge-git-branch-merged-says-no-about
type: lesson
status: active
created: "2026-08-16"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 210: Under squash-merge, `git branch --merged` says no about every branch that landed

**Context**: end-of-session cleanup, deciding which local branches were safe to delete. The repo's standing rule is to remove a worktree and branch as soon as its PR merges, so the question is answered many times a week.

**Problem**: `git branch --merged origin/main` reported **every** local branch as unmerged, including four whose PRs had demonstrably merged minutes earlier (#1001, #1009, #1017, #1019). The one branch it *did* report as merged belonged to a parallel session and was the only one nobody had squashed. Acting on that output would have inverted the decision exactly: keep the landed branches, delete the live one. `--merged` asks whether the branch tip is an **ancestor** of the target, and squash-merge creates a brand-new commit with the same *content* and no ancestry link — so the true answer to the question git was asked is "no", while the answer to the question being asked is "yes". Same root as #970, where the spec-archive gate false-positives on the documented squash-rebuild pattern, and adjacent to a `reviewed_sha` that stops existing once its branch is squashed away.

**Solution**: ask the forge, not the graph. `gh pr list --state all --head <branch> --json number,state,mergedAt` reports what actually happened to the change, and it stays correct under squash, rebase and merge-commit alike. Delete only when some PR on that head has a non-null `mergedAt` **and** none is still `OPEN` — a branch that was reused after its first PR merged carries both, and "a merged PR exists" alone would delete live work. Verified each of the four against its PR number before deleting, and left the parallel session's branch alone because it had no merged PR — the opposite of what git advised in both cases.

**Rule**: a merge strategy that rewrites commits destroys ancestry, and every tool that reasons about ancestry silently changes meaning under it — `git branch --merged`, `git log A..B`, `cherry-pick` detection, and any check pinned to a pre-merge sha. Before trusting one, ask what it would say about a change that landed by squash. The failure is dangerous rather than annoying because it fails toward *"not done"*: the output looks like a conservative, safe answer, so it invites acting on it. When the question is "did this change land", the forge is the source of truth and the local graph is a cache that squash-merge invalidates.

**Tags**: `git`, `squash-merge`, `verification`, `cleanup`, `false-negative`
