---
tags: [spec, tasks]
created: "2026-08-28"
---

# Tasks - CLI-064-doctor-profile-heal

## Setup

- [x] Branch: `fix/doctor-profile-heal` from `origin/main`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left in `proposal.md`

## Implementation

- [x] [AC1] Failing test: oversized profile and duplicated marker pair FAIL; healthy PASSes (`TestCheckProfileFiles_DetectsBUG020Corruption`)
- [x] [AC1] `profileCorruption` reads the two BUG-020 signals; `checkProfileFiles` reports them
- [x] [AC2] Failing test: heal path follows `SCRIPTS_DIR` from the contract (`TestCheckProfileFiles_HealPathFollowsTheContract`)
- [x] [AC2] `profileHealPath` resolves through the contract, never a literal
- [x] [AC3] Failing test: `--fix` runs the heal via the seam and is verified by consequence (`TestCheckProfileFiles_FixRunsTheHealAndVerifiesByConsequence`)
- [x] [AC3] `repairProfile` shells out through `System` and re-reads the profile
- [x] [AC4] `scripts/profile-heal.ps1` marker threshold aligned to doctor's (>1)
- [x] [AC5] `TestCheckProfileFiles` (existing) unchanged on the Linux branch
- [x] Mutation check: `profileCorruption` returning nil fails AC1 and AC3 tests

## Verification

- [x] Go loop: build, vet (host, `GOOS=windows`, `GOOS=linux`), test, golangci-lint
- [x] `verification.md` records the evidence
