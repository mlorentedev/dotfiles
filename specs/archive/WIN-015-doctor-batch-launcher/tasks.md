---
tags: [spec, tasks, templates]
created: "2026-11-17"
id: "WIN-015-doctor-batch-launcher"
type: spec-tasks
status: active
owner: manu
---

# Tasks - WIN-015-doctor-batch-launcher

## Setup

- [x] Issue #3572 assigned, worktree `feat-3572-doctor-batch-launcher`

## Implementation

- [x] [AC1-AC6] Failing tests in `cli/internal/doctor/checks_agent_launchers_test.go`
- [x] [AC6] `System.NativeBehind`, `lane.MiseNative` in production, nil in every test System
- [x] [AC1-AC5] `checkAgentLaunchers`, registered as a machine check

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass (`go vet` for Windows and Linux)
- [ ] Lint passes (CI's `lint` job; a CRLF checkout fails `gofmt` locally)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#3589)

## Machine-readable features

See `features.json`.
