---
tags: [spec, tasks]
created: "2026-06-14"
---

# Tasks - CLI-013-dotf-doctor-quick

> TDD order. Frozen once `implementing` began.

## Setup

- [x] Branch `feat/dotf-doctor-quick` from main
- [x] `proposal.md` complete; work-gate #380 verified OPEN

## Implementation (TDD)

- [x] Write `TestRun_QuickSkipsHeavySections` — full mode fails on missing tools, quick mode passes (contract-only) and omits the heavy section headers
- [x] Add `Quick bool` to `doctor.Options`; gate the healthcheck sweep + heals behind `if !opts.Quick`; header announces `[quick]`
- [x] Add `--quick` flag to the cobra `doctor` command (report-only; ignores `--fix`)
- [x] Re-wire `scripts/claude-session-start.sh` to surface drift via `dotf doctor --quick`, gated on `dotf` on PATH AND a deployed `env-contract.json` (hermetic-test-safe)
- [x] Restore the SessionStart parity bats to the CLI-013 reality (linux → `dotf doctor --quick`, windows → `doctor.ps1`)
- [x] Document `--quick` in `cli/README.md`

## Closing

- [x] `go test ./...` green; `go vet` + `gofmt` clean; `golangci-lint` 0 issues
- [x] `shellcheck` clean on `claude-session-start.sh`; `session-start-false-positives.bats` + `setup-linux.bats` green (hermetic isolation preserved)
- [x] Smoke: `dotf doctor --quick` runs only the 4 contract sections, exit 0, **0.00s** (no compile-harness fork) vs ~2.8s full
- [x] `verification.md` filled
- [ ] PR opened referencing this spec folder
