---
spec: "2026-11-16-sysupgrade-windows-elevation"
verdict: "PASS"
reviewed_sha: "2060f6296cf0ad7a03ced3d2c770bf751b9fe1ac"
reviewer: "agy/gemini-4-hydrogen-high"
date: "2026-11-16"
---

## Adversarial review

**Scope**: `2026-11-16-sysupgrade-windows-elevation` (`git diff e06c73525bd6891c0bb5251dc9b07499103607d7...HEAD`, centered on spec commit `9036b17fe8df` and its interaction with subsequent `converge` commits `2060f629` and `d0c98f7d`)
**Sources**: `specs/2026-11-16-sysupgrade-windows-elevation/{proposal,tasks,verification}.md`, `specs/2026-11-16-sysupgrade-windows-elevation/features.json`, `cli/internal/converge/{sysupgrade,sysupgrade_test,elevated_windows,elevated_other,converge,execmanager_test,needs_registry}.go`, `cli/internal/cmd/{update,update_skip_test,update_test,converge,converge_runlog}.go`, `cli/internal/tools/install.go`, `cli/internal/update/{update,record}.go`, `docs/lessons/lesson-609-a-failure-the-account-cannot-fix-is-a-report-not-an-error.md`, `docs/runbooks/guide-self-deploy-timer.md`

### Spec and task alignment
- **Contract digests & tags**: `dotf spec digest 2026-11-16-sysupgrade-windows-elevation` matches `review-request.json` (`features.json`, `proposal.md`, `tasks.md`). Zero `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in the spec folder.
- **AC1 & AC2 (non-elevated Windows scope split, detail marker, fail-closed unknown scope)**: Verified in `cli/internal/converge/sysupgrade.go` (`windows`, `splitMachineWide`, `elevationNote`, `exitStatus`, `NeedsElevationNote`). Covered by `TestSystemUpgradeWindowsWithoutElevationSkipsMachineWideAndToleratesAFailure`, `TestSystemUpgradeWindowsPlanNamesMachineWideWithoutElevation`, and `TestSystemUpgradeWindowsWithoutElevationAttemptsNothingWhenScopeIsUnknown`. Read-only `winget list --scope machine --accept-source-agreements --disable-interactivity` verified directly on this Windows host (exited 0, cleanly emits `Id` column including `Docker.DockerDesktop`, `Microsoft.PowerShell`, `Amazon.AWSCLI`).
- **AC3 (Probe returns nil without elevation; silent no-op upgrades caught in Reconcile)**: Verified in `Probe` (`due = nil` when `env.GOOS == "windows" && !r.isRoot()`) and `confirmUpgraded` in `windows()`. Covered by `TestSystemUpgradeWindowsWithoutElevationSkipsMachineWideAndToleratesAFailure` and `TestSystemUpgradeWindowsWithoutElevationFailsAnUpgradeThatDidNotMove`.
- **AC4 (elevated Windows unchanged)**: Verified in `windows()` (`elevated := r.isRoot()` skips `splitMachineWide`, joins failures into step error, `Probe` enforces post-condition). Covered by `TestSystemUpgradeWindowsUpgradesPerIdAndIsolatesAFailure` and `TestSystemUpgradeWindowsElevatedUpgradesMachineWide`.
- **AC5 (`dotf update` records `upgrade-skipped`)**: Verified in `cli/internal/cmd/update.go` (`systemUpgradeSkip` checks `e.Status == converge.StatusSkipped || strings.Contains(e.Detail, converge.NeedsElevationNote)`) and `cli/internal/update/update.go` (`upgradeQuietDay`). Covered by `TestSystemUpgradeSkipReadsOnlyTheStepsSkip/ran_but_left_ids_waiting_for_elevation` and `TestUpdate_NothingToPullRunsOnlySystemUpgrade`.
- **AC6 (cross-OS build/vet & process token / euid)**: Verified `elevated_windows.go` (`windows.GetCurrentProcessToken().IsElevated()`) and `elevated_other.go` (`os.Geteuid() == 0`). `GOOS=windows|linux|darwin go build ./... && go vet ./...` exited 0 on all three targets.
- **Tasks & cross-commit interaction**: All implementation tasks in `tasks.md` match the diff (`9036b17fe8df`); closing tasks for CI lint (`gofmt` on LF checkout) and opening the PR are accurately left unchecked. Subsequent commit `2060f629` (`tools.BoundedCommand` via `ExecManager`) wraps `*exec.ExitError` with `%w` on normal errors so `isExitCode` continues to match, and `exitStatus` cleanly extracts the timeout head when `winget` is stopped. `d0c98f7d` keeps `system-upgrade` and `cache-gc` independent for quiet-day runs.
- **Verification commands & mutation probes**:
  - `features.json` `f1`, `f2`, `f3`, `f4` commands all ran in this session and passed.
  - Full `cli/internal/converge` suite passed (`ok ... 202.319s`); `cli/internal/cmd` `-run 'TestSystemUpgradeSkip|TestUpdate'` passed (`ok ... 45.963s`); `cli/internal/update` `-run 'TestRecord|TestSelfDeploy|TestSystemUpgrade'` passed (`ok ... 1.866s`).
  - Four mutation probes (skipping `splitMachineWide`, disabling the non-elevated Windows guard in `Probe`, skipping `confirmUpgraded`, and removing `NeedsElevationNote` in `systemUpgradeSkip`) all compiled and failed their target tests (exit 1), then were reverted cleanly.
  - `golangci-lint run --no-config --enable=gocyclo,ineffassign,staticcheck,unused ./internal/converge/ ./internal/cmd/` passed with 0 issues; default `golangci-lint run` fails on 3 unrelated files (`bind.go`, `cachegc.go`, `catalog.go`) due to `gofmt` on CRLF checkout (consistent with `verification.md`, leaving CI's LF lint job as the `gofmt` check — UNVERIFIED for `gofmt` locally).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | detail-wording | When all outdated winget ids on non-elevated Windows move into `waiting` (`len(due) == 0 && len(waiting) > 0`), `upgradeDetail` still emits `"catalog entries up to date"` immediately before `"; left, needs elevation: ..."`, and `converge` reports the step `[ OK ]` (matches existing darwin `needs_sudo` behavior) | `cli/internal/converge/sysupgrade.go:192,219,624-625` | UNTESTED | code + tests (`cli/internal/converge/sysupgrade.go`, disposition in `verification.md`) |
| Minor | THEORETICAL | error-handling | In `windows()`, if `confirmUpgraded(done)` returns an error on the second `winget upgrade` listing, line 213 returns `Result{}, err`, discarding `Changes`/`upgraded:` for ids already upgraded, any `waiting` ids, and any prior `failed` errors | `cli/internal/converge/sysupgrade.go:211-214` | UNTESTED | code + tests (`cli/internal/converge/sysupgrade.go`, disposition in `verification.md`) |
| Minor | THEORETICAL | test-coverage | `splitMachineWide` tolerating `wingetNoneFound` (`0x8A150014`) when no machine-wide packages exist is not directly tested; `Probe` on non-elevated Windows also runs a third `r.outdated` subprocess call before setting `due = nil`, so a transient listing error in `Probe` fails an already-succeeded `Reconcile` | `cli/internal/converge/sysupgrade.go:93-105,264-265`; `cli/internal/converge/sysupgrade_test.go:516-532` | UNTESTED | tests / code (`cli/internal/converge/sysupgrade_test.go`, disposition in `verification.md`) |
| Minor | SPECULATIVE | maintainability | `systemUpgrade.windows` is 44 lines (just over the 40-line guideline in `AGENTS.md`), and local `gofmt` check via `golangci-lint` is blocked by CRLF checkout on unrelated files | `cli/internal/converge/sysupgrade.go:178-221`; `golangci-lint run` output | N/A | CI lint check / `verification.md` |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All six ACs verified by unit tests, mutation probes, real `winget list` check, and cross-OS builds; minor untested edge paths (`wingetNoneFound` in `splitMachineWide`, `confirmUpgraded` listing error). |
| Verification | B | `verification.md` and `features.json` commands map each AC to named passing tests and real-host runs; local `gofmt` and full `go test ./...` rely on CI LF checkout. |
| Scope | A | Spec commit `9036b17fe8df` stays strictly within `sysupgrade`, `cmd/update`, docs, and spec files, and composes cleanly with `2060f629` and `d0c98f7d`. |
| Reliability | B | Fails closed when scope is unknown, avoids admin prompts, catches silent no-op upgrades via `confirmUpgraded`, and pairs with bounded winget timeouts; `confirmUpgraded` error path drops partial `Result`. |
| Maintainability | B | Clean build-tagged elevation helper and clear WHY comments; `golangci-lint` (`gocyclo,ineffassign,staticcheck,unused`) passes with 0 issues (`windows()` is 44 lines). |
| Handoff-readiness | A | `proposal.md`, `tasks.md`, `verification.md`, `features.json`, `lesson-609`, `guide-self-deploy-timer.md`, and `dotf update` help are complete and aligned. |

### Verdict
PASS

### Recommended next steps
- Disposition the minor findings above in `specs/2026-11-16-sysupgrade-windows-elevation/verification.md` (either ticketed as follow-ups or declined with reason; do not edit `proposal.md`, `tasks.md`, or `features.json`, as the contract set is closed under this PASS verdict).
- Confirm CI's Linux/LF `golangci-lint` job and full `go test ./...` pass on the PR before merging and running `dotf spec archive 2026-11-16-sysupgrade-windows-elevation`.
