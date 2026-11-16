---
tags: [spec, verification, templates]
created: "2026-11-16"
id: "2026-11-16-sysupgrade-windows-elevation"
type: spec-verification
status: active
owner: manu
---

# Verification - 2026-11-16-sysupgrade-windows-elevation

## Evidence

Tests are in `cli/internal/converge/sysupgrade_test.go` unless a path is given.

- [x] AC1 -> `TestSystemUpgradeWindowsWithoutElevationSkipsMachineWideAndToleratesAFailure` (no `winget upgrade`
      for the machine-wide id, the failing id does not fail the step, the third id upgrades),
      `TestSystemUpgradeWindowsPlanNamesMachineWideWithoutElevation`
- [x] AC2 -> the same test (marker, `machine-wide install`, exit status without winget's output) and
      `TestSystemUpgradeWindowsWithoutElevationAttemptsNothingWhenScopeIsUnknown`
- [x] AC3 -> the same test's `Probe` assertion
- [x] AC4 -> `TestSystemUpgradeWindowsUpgradesPerIdAndIsolatesAFailure` (now pinned to an elevated run),
      `TestSystemUpgradeWindowsElevatedUpgradesMachineWide`
- [x] AC5 -> `TestSystemUpgradeSkipReadsOnlyTheStepsSkip/ran_but_left_ids_waiting_for_elevation` in
      `cli/internal/cmd/update_skip_test.go`
- [x] AC6 -> `GOOS=windows|linux|darwin go vet ./...` exit 0, 2026-11-16

## Test status

- `go test ./internal/converge/` -> `ok` (382s on the Windows workstation), 2026-11-16.
- `go test ./internal/cmd/ -run 'SystemUpgradeSkip|Update'` -> `ok`.
- `go build ./...`, `go vet ./...` for windows, linux and darwin -> exit 0.
- `golangci-lint` 2.12.2 (the pin): the only findings are `gofmt` on files this change does not touch, from the
  CRLF checkout of this workstation (`core.autocrlf=true`). CI lints an LF checkout.
- `go test ./...` on this workstation is not green and was not before this change: `doctor`, `evals`, `lane`,
  `reproduce`, `spec` and `worktree` fail or hit the 10-minute package timeout here (symlink tests without the
  privilege to create one, and a slow non-elevated machine). None of those packages is touched. CI is the
  reference for the full suite.
- Manual, on the non-administrator workstation of #3553:
  - First version of the fix (tolerate the failure only): `go run ./cmd/dotf converge --only system-upgrade`
    exited 0 in 5m38s, and raised an administrator prompt for the VC++ redistributable and for Docker Desktop.
    That is what made "never attempt" a requirement.
  - Final version: `go run ./cmd/dotf converge --plan --only system-upgrade` -> exit 0 in 7.6s, detail
    `left, needs elevation: Microsoft.VCRedist.2015+.x64 (machine-wide install), Microsoft.PowerShell
    (machine-wide install), Docker.DockerDesktop (machine-wide install)`. An apply was not run again on that
    machine: all three outdated ids are machine-wide, so it would attempt nothing, which the unit test asserts
    on the fake's command log.

## Decisions made during implementation

- The rule is the process token, not a list of winget exit codes: `0x8a15008e` is an install-technology
  mismatch, not an elevation error, and the probe keeps no state to remember which ids failed.
- Tolerating the failure was not enough; the attempt raises the prompt. The scope is read first from
  `winget list --scope machine`, which is read-only. An id listed under both scopes counts as machine-wide.
- When the scope listing fails, nothing is attempted. The alternative (attempt everything) can prompt.
- The waiting ids reach `last.json` through a marker in the step's detail (`NeedsElevationNote`) rather than a
  new field on `converge.Result`: one reader, and the runner's contract stays as it is.
- The runner's fail-fast order is not changed here; filed as #3554.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-609-a-failure-the-account-cannot-fix-is-a-report-not-an-error.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: narrows one step's failure rule to match what linux and darwin already do; the converge-wide question is #3554
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: one project so far

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/2026-11-16-sysupgrade-windows-elevation/` -> `specs/archive/2026-11-16-sysupgrade-windows-elevation/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
