---
id: lesson-069-don-t-commit-to-a-shared-vault-that-has-another-se
type: lesson
status: active
created: "2026-05-30"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 069: Don't commit to a shared vault that has another session's staged work — use an isolated worktree

**Context:** SDD-011 close-out. The vault (`~/Projects/knowledge`) was being edited in parallel by another session that had a feature branch (`rfd-001/pdf-modifier-mcp`) checked out with *staged* work (a pollex restructure). I committed a scoped 2-file SDD-011 change there.
**Problem:** A scoped `git commit -- <paths>` correctly isolated my 2 files (it did NOT sweep the other session's staged work), but the commit landed on the *other session's* feature branch. When that session switched branches, the vault working tree reverted and my change was stranded off `master` (the canonical branch). Committing into a repo another agent/human is actively branch-switching is racy and misplaces the work.
**Solution:** When you must read or operate on a specific commit of a repo another session is using, use `git worktree add --detach <tmp> <commit>` — an isolated checkout that shares only the object store, never the live HEAD/index. To place a change on the right branch without disturbing the active checkout: `git worktree add <tmp> master` + `git cherry-pick <commit>`, then `git worktree remove`. Detect the hazard early: a `git status` showing *staged* changes you did not make means another session owns this checkout — do not commit there.
**Tags:** `#vault` `#git` `#worktree` `#parallel-sessions` `#sdd`
