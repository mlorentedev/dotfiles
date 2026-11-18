---
spec: "WIN-016-sysupgrade-partial-result"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "382ad32a85826f8984c7f95b54b5f57709ca7f82"
reviewer: "agy/gemini-4-hydrogen-high"
date: "2026-11-17"
---

## Adversarial review

**Scope**: `WIN-016-sysupgrade-partial-result` (`git diff 3045157e86f02dac68c581bbef3899b563c1a0b9...HEAD`, centered on spec commit `19be8d735339` and cross-checked against `a8b3baf6` and `75d77d28`)
**Sources**: `specs/WIN-016-sysupgrade-partial-result/{proposal,tasks,verification}.md`, `specs/WIN-016-sysupgrade-partial-result/features.json`, `specs/archive/2026-11-16-sysupgrade-windows-elevation/{review,verification}.md`, `cli/internal/converge/{sysupgrade,sysupgrade_test,converge,observe_test}.go`, `cli/internal/cmd/{update,update_skip_test,converge,converge_console_test}.go`, `cli/internal/update/update.go`, `cli/internal/spec/{archive,promotion,promotion_test,review}.go`

### Spec and task alignment
- **Contract digests & tags**: `dotf spec digest WIN-016-sysupgrade-partial-result` matches `review-request.json` across `features.json`, `proposal.md`, and `tasks.md`. Zero unresolved draft/suggestion tags exist in the spec folder, and `dotf spec audit` reports `[OK] 14 active spec(s), every one tracking an open issue` (#3573).
- **AC1 (confirming listing failure preserves partial result & waiting IDs)**: Verified in `cli/internal/converge/sysupgrade.go` (`windows` lines 213–216, `unconfirmed` lines 233–236) and downstream in `cli/internal/converge/converge.go` (`runStep` lines 248–250) and `cli/internal/update/update.go` (`upgradeQuietDay` lines 183–188, which checks `err != nil` before `skipped != ""`). Covered by `TestSystemUpgradeWindowsKeepsWhatItDidWhenTheConfirmingListingFails`.
- **AC2 (non-elevated Windows Probe makes no winget listing)**: Verified in `cli/internal/converge/sysupgrade.go` (`Probe` lines 93–102 returning `nil` before `r.outdated`). Covered by `TestSystemUpgradeWindowsProbeDoesNotListWithoutElevation`.
- **AC3 (every outdated ID waiting does not claim catalog is up to date)**: Verified in `cli/internal/converge/sysupgrade.go` (`upgradeSummary` lines 647–659, shared by `windows` and `upgradeDetail` for `darwin`/`linux`). Covered by `TestSystemUpgradeWindowsDoesNotCallTheCatalogUpToDateWhenEveryIdWaits` (Windows plan/apply, macOS sudo-cask apply, and clean Windows catalog).
- **AC4 (`winget list --scope machine` returning `NO_APPLICATIONS_FOUND`)**: Verified in `cli/internal/converge/sysupgrade.go` (`splitMachineWide` lines 278–295 tolerating `wingetNoneFound` / `0x8A150014`). Covered by `TestSystemUpgradeWindowsWithoutElevationReadsNoMachineWideInstallAsNone`.
- **Tasks & cross-commit interaction**: All implementation tasks in `tasks.md` match `19be8d735339`; the CI `lint` task (`gofmt` on CRLF checkout) is accurately left unchecked. Commit `a8b3baf6` archived the parent spec (`specs/archive/2026-11-16-sysupgrade-windows-elevation/`); `proposal.md` lines 19 and 48 still cite the pre-archive path `specs/2026-11-16-sysupgrade-windows-elevation` (historical path from when `19be8d735339` was authored; do not edit `proposal.md` under this verdict). Commit `75d77d28` adds step-dependency reporting tests in `observe_test.go` and `converge_console_test.go` without affecting `system-upgrade`.
- **Verification commands & mutation probes**:
  - All 4 `features.json` commands and all 27 `TestSystemUpgrade*` tests passed (`ok ... 1.040s` / `1.150s`).
  - Full `cli/internal/converge` suite passed standalone (`go test ./internal/converge/ -count=1 -timeout 8m` -> `ok ... 279.702s`; an earlier concurrent `-timeout 5m` run timed out in unrelated `TestRecordsMirror_PrunesALeftoverTheCheckoutDeleted`). `TestRunObserved*` passed (`0.998s`).
  - `go test ./internal/cmd/ -count=1 -timeout 5m -run 'Update|SystemUpgrade|Converge'` passed (`ok ... 106.397s`).
  - Cross-OS `go vet ./internal/converge/ ./internal/cmd/` and `go build ./...` exited 0 for `windows`, `linux`, and `darwin`.
  - Four mutation probes in `sysupgrade.go` (dropping `unconfirmed` on `confirmUpgraded` error, removing `return nil` in non-elevated Windows `Probe`, ignoring `held` in `upgradeSummary`, and dropping `!isExitCode(err, wingetNoneFound)`) all compiled and failed their target tests (`exit code 1`), then were cleanly reverted.
  - `golangci-lint run --no-config --enable=gocyclo,ineffassign,staticcheck,unused ./internal/converge/ ./internal/cmd/` passed with `0 issues.`; default `golangci-lint run ./internal/converge/` reports `gofmt` differences on CRLF checkout (`sysupgrade.go`, `sysupgrade_test.go`, and 3 unrelated files), leaving CI's LF `lint` job as the authoritative `gofmt` gate (`UNVERIFIED` locally). Real `winget` mid-run transient failure was not run on live hardware (`pkgFake` used).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | archive-gate | `verification.md` writes `## Promotion candidates` as a plain bullet (`- None. ...`) instead of the three `- [x] <question>? <yes: path / no: reason>` checkbox lines required by `CheckPromotions`, so `dotf spec archive` will refuse until `verification.md` is updated | `specs/WIN-016-sysupgrade-partial-result/verification.md:32-35`; `cli/internal/spec/promotion.go:26,55-70`; `cli/internal/spec/archive.go:502-510` | `TestPromotionsRefuseAMissingFileSectionOrLines` | spec artifacts (`specs/WIN-016-sysupgrade-partial-result/verification.md`, outside the contract set) |
| Minor | THEORETICAL | test-coverage | Two secondary combinations lack dedicated assertions: (a) `darwin` `dryRun=true` when a `needs_sudo` cask waits in `TestSystemUpgradeWindowsDoesNotCallTheCatalogUpToDateWhenEveryIdWaits`, and (b) `waiting` containing a failed-upgrade ID alongside `done` when `confirmUpgraded` fails | `cli/internal/converge/sysupgrade.go:125,198-216`; `cli/internal/converge/sysupgrade_test.go:731-752,807-812` | UNTESTED | tests (`cli/internal/converge/sysupgrade_test.go`, optional; disposition in `verification.md`) |
| Minor | SPECULATIVE | defensive-parsing | `splitMachineWide` still passes `string(out)` through `wingetIDs` when `winget list --scope machine` exits with `wingetNoneFound` (`0x8A150014`) instead of returning early, matching `wingetBehind` | `cli/internal/converge/sysupgrade.go:279-286,486-491` | UNTESTED | — (surface only; disposition in `verification.md`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All four ACs pass tests and mutation probes; two minor secondary combinations (`darwin` dry-run wait, failed-ID `waiting` + confirm error) lack dedicated cases. |
| Verification       | B | Reproducible Go tests and cross-OS builds pass; `verification.md` fails `CheckPromotions` format, `gofmt` relies on CI LF checkout, and live `winget` failure is mocked. |
| Scope              | A | Spec commit `19be8d735339` stays strictly within `sysupgrade` and `WIN-016` spec files with zero unrelated edits. |
| Reliability        | B | Preserves partial results and waiting IDs on confirmation error, removes unused probe listing, and handles `0x8A150014` (`splitMachineWide` still parses `out` on `0x8A150014`). |
| Maintainability    | B | `golangci-lint` (`gocyclo,ineffassign,staticcheck,unused`) reports 0 issues and shared `upgradeSummary` avoids duplication (`windows()` is 45 lines, declined in `proposal.md`). |
| Handoff-readiness  | B | Code and contract docs align cleanly, but `verification.md` `## Promotion candidates` must use checkbox grammar before `dotf spec archive` will succeed. |

### Verdict
PASS WITH GAPS

### Recommended next steps
- **Update `verification.md` (outside the contract set; do NOT edit `proposal.md`, `tasks.md`, or `features.json`, as the contract set is closed under this verdict)**:
  - Replace the plain bullet in `specs/WIN-016-sysupgrade-partial-result/verification.md` under `## Promotion candidates` with the three `- [x] ...? no: <reason>` candidate lines required by `cli/internal/spec/promotion.go` (`CheckPromotions`), and optionally align `## Archive checklist` with the template items.
  - Record dispositions (applied, ticketed, or declined with reason) in `verification.md` for the two remaining Minor findings (`darwin` dry-run / mixed `waiting` test cases and `splitMachineWide` early return on `wingetNoneFound`).
- Confirm CI's Linux/LF `lint` (`gofmt`) job passes on PR #3580, then run `dotf spec archive WIN-016-sysupgrade-partial-result`.
