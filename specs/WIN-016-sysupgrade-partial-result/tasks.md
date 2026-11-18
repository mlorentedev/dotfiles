---
tags: [spec, tasks, templates]
created: "2026-11-17"
id: "WIN-016-sysupgrade-partial-result"
type: spec-tasks
status: active
owner: manu
---

# Tasks - WIN-016-sysupgrade-partial-result

## Setup

- [x] Issue #3573 assigned, worktree `fix-3573-sysupgrade-partial-result`

## Implementation

- [x] [AC1] Failing test: the confirming listing fails on its second run (`failAt` in the test fake)
- [x] [AC2] Failing test: the probe lists nothing without elevation
- [x] [AC3] Failing test: every id waits, plan and apply
- [x] [AC4] Test for `NO_APPLICATIONS_FOUND` from the machine-scope listing (passed before the change: coverage only)
- [x] [AC1] `unconfirmed` result, error wrapped as `confirming the upgrades`
- [x] [AC2] `Probe` returns before listing on Windows without elevation
- [x] [AC3] `upgradeSummary` takes whether anything is held; `upgradeDetail` and the Windows path share it

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass (`go vet` for Windows and Linux)
- [ ] Lint passes (CI's `lint` job; a CRLF checkout fails `gofmt` locally on every Go file)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#3580)

## Machine-readable features

See `features.json`.
