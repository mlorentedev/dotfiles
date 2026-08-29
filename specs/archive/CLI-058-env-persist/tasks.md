---
tags: [spec, tasks]
created: "2026-08-28"
---

# Tasks - CLI-058-env-persist

## Setup

- [x] Branch: `feat/env-persist` from `origin/main`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left in `proposal.md`

## Implementation

- [x] [AC1] Failing test `TestPersist_TouchesOnlyWhatDiffers` (fake store; second run writes nothing)
- [x] [AC1] `env.Persist` + `env.UserEnvStore`; Windows store on `HKCU\Environment` (`x/sys/windows/registry`) with `WM_SETTINGCHANGE` broadcast; `env.ResolveVars` shared with `generate`
- [x] [AC2] `env.Drift` (+ `TestDrift_NamesMissingAndDifferent`); `dotf env persist --check`
- [x] [AC3] `NewUserEnvStore` returns `ErrUserEnvUnsupported` off Windows; the command prints the no-op and exits 0
- [x] [AC4] Failing test `TestCheckPersistedEnv_ByStatus` (four rows by status + no-seam case); `checkPersistedEnv` through the `System.UserEnv` seam, registered after the contract env vars
- [x] [AC5] `setup-windows.ps1` calls `dotf env persist` after `dotf env generate`; bats ordering guard
- [x] [AC6] Box: `--check` → 10 drifted; `persist` → 10 changed; `--check` ok; `persist` → 0 changed; `Start-Process -UseNewEnvironment` child sees the values
- [x] Mutation checks: Persist writing everything fails AC1's test; Drift never reporting fails AC4's test

## Verification

- [x] Go loop: build, vet (host, `GOOS=windows`, `GOOS=linux`), test, golangci-lint (0 issues)
- [x] bats: setup-windows; parse + ASCII delta 0 on `setup-windows.ps1`
- [x] `verification.md` records the evidence
