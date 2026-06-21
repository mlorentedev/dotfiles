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

## PR-B — repoint + delete (separate PR, closes #380)

- [ ] Settle the §4 residual: add `$PROFILE` / `.claude/CLAUDE.md` / `.gemini/AGY.md` existence (+ optionally BUG-012 junction) to `dotf doctor`, or consciously accept the drop
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
