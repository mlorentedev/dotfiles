---
id: lesson-143-a-guard-test-that-names-its-own-trigger-string-can
type: lesson
status: active
created: "2026-07-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 143: A guard test that names its own trigger string can match itself once tracked

**Context**: GUARD-002 (#669) added `tests/sensitive-hygiene.bats`, asserting `git grep -l "docs/SECRETS.md"` returns no matches, to catch a future resurrection of a dead doc reference.

**Problem**: The test file's own source contains the literal string `"docs/SECRETS.md"` as the grep pattern argument. `git grep` only searches tracked files, so the assertion passed locally against the untracked new test file (written but not yet `git add`-ed) — then failed in CI once the file was committed and became a match for its own search pattern.

**Solution**: Exclude the guard file itself from its own search scope via a git pathspec exclusion (`git grep -l "pattern" -- . ':!tests/sensitive-hygiene.bats'`).

**Rule**: Before trusting a "no references to X" guard test locally, make sure your local run sees the same tracked state CI will — stage the new test file first, or otherwise verify against the committed tree, not the working tree. An untracked new file is invisible to plain `git grep` and can produce a false-clean local pass that only breaks in CI. Separately: any guard whose search pattern literally appears in its own source needs an explicit self-exclusion, or it will eventually fail on itself.
