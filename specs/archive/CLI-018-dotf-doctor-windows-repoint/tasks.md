---
tags: [spec, tasks]
created: "2026-06-21"
---

# Tasks - CLI-018-dotf-doctor-windows-repoint

> Build-then-delete, two PRs. PR-A (this branch) ports the missing Windows-only
> check, build-only. PR-B repoints callers + deletes the `.ps1`.

## Setup

- [x] Branch from main: `feat/dotf-doctor-windows-checks`
- [x] `proposal.md` complete; parity scan done (10/13 covered; §13 missing; §4 residual + §11 deferred to CLI-019)

## PR-A — port the Orca hook check (build-only)

- [x] Write table-driven `checks_orca_test.go` (skip-when-absent, timeoutSec pass/fail, transport pass/fail, combined)
- [x] Implement `checks_orca.go` `checkOrcaHook` (orca.json timeoutSec ≥ 30; copilot-hook.ps1 uses HttpWebRequest not Invoke-WebRequest; SKIP if absent)
- [x] Register `checkOrcaHook` in the full sweep in `doctor.go` (after `checkAntigravity`; not in `--quick`)
- [x] `go test ./internal/doctor/...` green; `go vet` clean; gofmt clean
- [ ] PR-A opened (references #380, does NOT close it)

## PR-B0 — port the §4 residual (build-only, this PR)

> Strangler-fig: like PR-A, port the missing coverage to `dotf doctor` and prove
> it on CI BEFORE the deletion PR removes `healthcheck.ps1`. Split decided this
> session (CLI-019 PR-A→PR-B precedent): build-only, no deletion, no repoint.

- [x] `checkProfileFiles` (`checks_profile.go`): existence of `.claude/CLAUDE.md`
      + `.gemini/AGY.md` (cross-OS) and the Windows `$PROFILE` (pwsh 7 / WinPS 5.1
      under Documents, incl. OneDrive-redirected). FAIL on missing; SKIP `$PROFILE`
      off-Windows (POSIX uses `.zshrc`/`.bashrc`, already in `checkSymlinks`).
      BUG-012 junction dropped (secondary, superseded by BUG-014 which `dotf
      doctor` already has).
- [x] Table test `checks_profile_test.go` (7 rows: posix pass + 2 fails, win pwsh
      / WinPS / OneDrive / missing); registered after `checkSymlinks` (not `--quick`).
- [x] `go test ./internal/doctor/...` green; gofmt + vet clean.
- [ ] PR-B0 opened (references #509, does NOT close it)

## PR-B — repoint + delete (separate PR, closes #380/#509)

- [x] §4 residual settled → ported in PR-B0 (above); no coverage lost on deletion.
- [ ] Repoint `setup-windows.ps1` post-setup (deploy + invoke blocks) → `dotf doctor`
- [ ] Repoint `ci.yml` `test-windows` + the profile `hc` alias → `dotf doctor`
- [ ] `git rm scripts/healthcheck.ps1 scripts/doctor.ps1` + their Pester
- [ ] Guard-grep clean for `(healthcheck|doctor)\.ps1`; `test-windows` CI green

## Closing

- [ ] All PR-A acceptance criteria covered by tests (`features.json`)
- [ ] `verification.md` filled
- [ ] Both PRs merged → archive spec, close #380

## Note

§11 (repo-deploy drift) belongs to **CLI-019 (#488)**, not this issue. The `.ps1`'s §11 coverage and CLI-019's drift absorption should be sequenced so PR-B's deletion does not open a drift-coverage gap.
