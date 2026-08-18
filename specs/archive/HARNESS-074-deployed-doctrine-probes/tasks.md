---
id: "HARNESS-074-tasks"
type: spec
status: draft
owner: manu
tags: [spec, tasks, templates]
created: "2026-08-18"
---

# Tasks - HARNESS-074-deployed-doctrine-probes

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Worktree created from main (`dotfiles-wt-router`, branch `feat/harness-router-and-slimming`)
- [x] Proposal initialized and issue #1035 linked

## Implementation (TDD Sequence)

- [x] [P] [AC1] **TDD (Red)**: Write unit tests in `cli/internal/doctor/checks_deployed_doctrine_test.go` asserting doctor fails when an enforced region is missing from a deployed surface.
- [x] [AC1] **TDD (Green)**: Implement `checkDeployedDoctrine` in `cli/internal/doctor/checks_deploy.go` and hook into `checkHarnessDrift`.
- [x] [AC2] **TDD (Verify Green)**: Run `go test ./cli/internal/doctor/...` and verify tests pass on complete deployed payload.
- [x] [AC3] **Mutation Testing**: Drop an enforced section from mock deployed target and assert doctor reports failure naming the missing section.

## Closing

- [x] All tests in `cli/` pass (`go test ./...`)
- [x] `features.json` populated with verification commands and captured evidence
- [x] `verification.md` filled in with mutation test evidence

