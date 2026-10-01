---
id: lesson-322
type: lesson
status: active
created: "2026-09-30"
owner: manu
tags: [lesson, ci, paths-filter, guards]
---

# 322 — An allow-list path filter classifies every unlisted path as safe to skip

## What happened

HARNESS-041 (#1105) gave `ci.yml` a `changes` job whose `code` filter lists what counts as code. A PR that touches nothing on the list skips the heavy steps of every test and lint job, and the jobs still report green. The filter never said "these paths are docs". It said "these paths are code", so every path missing from it was docs by default.

Nothing checked that the list was complete. Round 1 of the spec's adversarial review scanned the 374 commits since the change: 40 would skip CI while touching files the suite reads. Six PRs touched only `secrets/registry.yaml`, which four test files read. #1211 touched only `ssh/config`. #1465 touched only `.claude/CLAUDE.md`, the file `scripts/check-doc-paths.sh` exists to check, and that lint sits behind the same filter.

The pre-merge gate had turned into a red-main gate for these paths: `push` runs the full matrix, so breakage surfaced after the merge.

The tests did not catch it either. They grepped the workflow for the guard's text anywhere in the file, so deleting one job's `needs:` or one step's `if:` left them green.

## Rule

- Treat an allow-list filter as a classification of the whole tree, and test it that way. Every tracked top-level entry is either on the list or on an explicit exclusion list with a reason. A new directory then fails a test instead of defaulting to "skip".
- Widen the list by allow-list additions. Rewriting it as a deny-list fails open: one wrong negation skips every heavy step, and CI still reports green.
- Test a workflow's structure by parsing it, never by grepping it. A grep proves the text exists somewhere; a parse proves which job or step it guards. Kill each test's mutation (delete the `needs:`, delete the guard) before trusting it.

## References

- `.github/workflows/ci.yml`, job `changes`, filter `code`
- `tests/ci-path-filtering.bats`
- `specs/archive/HARNESS-041-ci-path-filtering/review-round-1.md`
