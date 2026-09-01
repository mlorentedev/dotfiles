---
tags: [spec, tasks, templates]
created: "2026-08-29"
---

# Tasks - CLI-066-doctor-profile-target

> TDD order. One task = one focused commit. Tick as you go. Frozen at the start of `implementing`.

## Setup

- [x] Branch created from main: `feat/doctor-profile-target`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1][AC2] `doctor`: `profileTarget(sys, home)` asks `pwsh -NoProfile -Command $PROFILE` through `CommandOutputBounded` (10 s), trims the answer, and falls back to `findPowerShellProfile` when pwsh is absent, fails or times out; the row names the path and says `resolved by pwsh` or `enumerated`. Tests: a profile outside the four roots found through the fake answer (CRLF-terminated); no pwsh → enumeration; pwsh errors → enumeration with the reason in the row.
- [x] [AC3] `doctor`: `repairProfile` runs the heal through `CommandOutputBounded` (60 s) as `pwsh -NoProfile -File <heal> -ProfilePath <profile>` and re-measures `profile`; the heal test's fakes answer the `$PROFILE` query and assert the new argv; the FAIL and FIX lines say the heal rewrites from the SSOT and keeps the rest only in the backup.
- [x] [AC4] `scripts/profile-heal.ps1`: `-ProfilePath` parameter, `$PROFILE` when absent; synopsis says so. `tests/profile-heal-ps1.bats` asserts the parameter exists and the default. ASCII only.
- [x] [AC5] `doctor`: `TestProfileHealThresholdsMatchTheScript` reads `scripts/profile-heal.ps1` and asserts `-gt 1MB` beside `profileMaxBytes == 1<<20`, and `StartMarkers -gt 1 -or $state.EndMarkers -gt 1` beside doctor's `> 1` rule.
- [x] [AC6] Box: `dotf doctor` names the pwsh-resolved path; `profile-heal.ps1 -ProfilePath <scratch copy with two marker pairs>` heals the copy; the real profile's hash is unchanged before/after.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test (AC6 by the box transcript in `verification.md`)
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `go build ./... && go vet ./... && go test ./...`, `GOOS=windows go vet ./...`, `golangci-lint run` (pinned), PSScriptAnalyzer clean on the script, bats for the script
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
