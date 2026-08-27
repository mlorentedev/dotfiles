---
tags: [spec, tasks, templates]
created: "2026-08-27"
---

# Tasks - HARNESS-086-cyclomatic-complexity-evals

## Setup

- [x] Branch created: `feat/cyclomatic-and-harness-engineering`
- [x] `proposal.md` complete and acceptance criteria defined

## Implementation

- [x] [AC1] Add `cyclomatic-complexity` skill record in `harness/skills/cyclomatic-complexity/`
- [x] [AC1] Add attribution row in `harness/skills/ATTRIBUTION.md`
- [x] [AC2] Register `code-complexity-and-refactor` trigger in `harness/triggers.json` and sync with `cli/internal/harness/triggers.json`
- [x] [AC3] Add benchmark evaluation suite in `tests/evals/harness-evals.json`
- [x] [AC4] Run `compile-harness.sh --refresh` and `compile-harness.sh --check`
- [x] [AC4] Verify all 118 Bats tests and Go unit tests pass

## Closing

- [x] Every acceptance criterion covered by tests
- [x] `features.json` filled
- [x] `verification.md` filled
