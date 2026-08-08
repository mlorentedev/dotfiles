---
tags: [spec, tasks, templates]
created: "2026-08-08"
---

# Tasks - BUG-042-closing-keyword-markdown

## Setup

- [x] Worktree from `origin/main`: `dotfiles-wt-773`, branch `fix/closing-keyword-markdown-aware`
- [x] Gating issue open: dotfiles#773
- [x] `proposal.md` complete, acceptance criteria testable

## Implementation

- [x] [AC1][AC2][AC4] Write failing tests for both quoted shapes and the
      delimiter-aware fence case
- [x] [AC3] Write a test that stays green before AND after — a preserved
      behaviour, not a new one; it is the assertion that would catch an anchoring
      regression
- [x] [AC5] Write a failing test for the colon form
- [x] [AC1][AC2][AC4] `_strip_markdown_code` — fence tracking with matching
      delimiter and length, inline span removal
- [x] [AC5] Optional colon in the keyword pattern
- [x] Regex patterns held in single-quoted variables: an unquoted backtick inside
      `[[ =~ ]]` would be command substitution
- [x] Rebased onto #808 after it merged; resolved the append conflict in
      `spec-gate-archive.bats` by rebuilding from main's version rather than
      splicing the conflict hunks, which had cut a test in half

## Closing

- [x] Every acceptance criterion covered by at least one test (30/30 in file)
- [x] `shellcheck` + `bash -n` clean
- [x] `check-bats-names` clean
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder; spec archived in the same PR
