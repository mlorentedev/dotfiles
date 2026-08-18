---
id: lesson-116-git-branch-merged-answers-is-the-tip-an-ancestor-n
type: lesson
status: active
created: "2026-06-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 116: `git branch --merged` answers "is the tip an ancestor?", not "is the content backed up" — verify before deleting

**Context**: Housekeeping a 5-week-old orphan branch (`fix/win-sessionstart-hook-path`). `git branch --merged origin/main` would have green-lit deleting it — its single commit does not conflict with `main`.

**Problem**: "Doesn't conflict" ≠ "redundant". The commit only *adds* files — 7 spec scaffolds — and 3 of them (AI-012, AI-013, WIN-003) exist nowhere in `main` (neither active `specs/` nor `specs/archive/`). `--merged` merely tests whether the branch tip is reachable from `main`, which is trivially true for a branch whose unique commit adds new files. Deleting on that signal silently loses unmerged work.

**Solution**: Per-artifact verification — for each path the branch adds, confirm it is implemented (in `main`), ticketed (an issue), or specified (a spec dir, active or archived). Three scaffolds failed all three → kept the branch.

**Rule**: "Safe to delete" means "all its content is backed up elsewhere", not "git says merged". Diff the branch's added content against `main` (`git log origin/main..branch`, then confirm each added path exists in main/archive/an issue) — never trust `--merged` for a delete decision.
