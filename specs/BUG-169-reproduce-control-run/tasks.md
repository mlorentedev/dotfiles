---
tags: [spec, tasks, templates]
created: "2026-11-12"
id: "BUG-169-reproduce-control-run"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-169-reproduce-control-run

## Implementation

- [x] [AC1] [AC2] Failing test: `TestRunWithAProbeNeedsTheCleanHeadAsItsControl` covers four cases, the plain and the inverted check each reading the same both ways or only with the probe
- [x] [AC1] [AC2] Add `Result.Control` and `ControlExit`, and run the control after the revert when the probe reproduced
- [x] [AC2] `TestRunWithAProbeRunsNoControlForARefutation`
- [x] [AC3] The row names the control: `TestReproductionRowNamesTheControl`; the help text says when the control runs and where the check runs

## Closing

- [x] `go test ./...`, `golangci-lint run`, `GOOS=windows` vet and lint
- [ ] `verification.md` filled in; independent review; archive
