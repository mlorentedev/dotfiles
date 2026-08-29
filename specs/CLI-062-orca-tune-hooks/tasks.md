---
tags: [spec, tasks, templates]
created: "2026-08-29"
---

# Tasks - CLI-062-orca-tune-hooks

> TDD order. One task = one focused commit. Tick as you go. Frozen at the start of `implementing`.

## Setup

- [x] Branch created from main: `feat/orca-tune-hooks`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] `orca`: `TuneHooks(config, script, floor, check, now)` — the timeout regex and tuner move here from doctor (`TimeoutBelow`, `TuneTimeout`), the POST swap is `TuneScript` with the script's exact `HttpWebRequest` block and captured indentation, `TuneScriptFile` is the per-file entry doctor shares, `writeTuned` backs up (`<file>.bak.<stamp>`) and writes atomically. The six Pester cases ported to `hooks_test.go` (nothing to do, both repaired with backups, idempotent, generous timeout untouched, check reports and writes nothing, unrecognised POST left alone).
- [x] [AC1] [AC2] `cmd`: `dotf orca tune-hooks [--check] [--timeout-sec] [--hook-config] [--hook-script]`; `--check` exits non-zero on drift; output names backups, fixes and the unrecognised case.
- [x] [AC3] `doctor`: `checkOrcaHook` reads the package; `--fix` repairs the script half through `TuneScriptFile`; both FAIL lines name `dotf orca tune-hooks`; the test's helper call follows the move.
- [x] [AC4] `setup-windows.ps1` invokes `dotf orca tune-hooks` (guarded on `dotf`) where it copied and ran the script; `orca-hook-tune.ps1` moves from `$deployedScripts` to `$retiredScripts` so old copies are swept. `git rm`: `scripts/orca-hook-tune.ps1`, `scripts/orca-tune.sh`, `tests/orca-hook-tune-ps1.bats`, `tests/orca-hook-tune.Tests.ps1`. `audit-007` row and floor list amended.
- [x] [AC4] bats: `setup-windows.bats` asserts the call and the retired-list entry, and that no `orca-hook-tune.ps1` is deployed.
- [x] [AC5] Box: `dotf orca tune-hooks --check` against the real files (exit 0, already tuned); scratch copies with `timeoutSec: 5` + `Invoke-WebRequest` tuned through `--hook-config/--hook-script`, backups beside them, `--check` clean afterwards.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test (AC5 by the box transcript in `verification.md`)
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `go build ./... && go vet ./... && go test ./...`, `GOOS=linux go vet ./...`, `golangci-lint run` (pinned), PSScriptAnalyzer on `setup-windows.ps1`, bats
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
