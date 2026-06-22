---
id: "CLI-018-dotf-doctor-windows-repoint"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-21"
issue: "mlorentedev/dotfiles#380"
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-018-dotf-doctor-windows-repoint

> AUDIT-007 Phase A / PR2 — wire `dotf doctor` as the Windows post-setup diagnostic
> and retire `healthcheck.ps1` + `doctor.ps1`. Because `dotf doctor` was
> consolidated from the `.sh` twins, it is missing Windows-only checks — so this is
> **build-then-delete**, split into two PRs.

## Why

`dotf doctor` is built and runs on Linux; `dotf` is installed on Windows (WIN-006) but **nothing invokes it there** — Windows still deploys + runs `healthcheck.ps1` + `doctor.ps1`. They are divergent orphans (no `.sh` twin keeps them honest). Wiring `dotf doctor` on Windows and deleting the `.ps1` is the next strangler-fig step. But a parity scan found `dotf doctor` (consolidated from the `.sh`) is **missing 2 Windows-relevant checks** — deleting the `.ps1` as a plain repoint would silently drop coverage (exactly the failure the CLI-020 lesson warns about).

## What

Two PRs under this issue:

**PR-A (this branch, `feat/dotf-doctor-windows-checks`) — build-only, no deletion:**
- Port `healthcheck.ps1` **§13 Orca Copilot hook (DX-006)** into `dotf doctor` as a new cross-OS check (`checks_orca.go`): `orca.json` hook `timeoutSec >= 30`, `copilot-hook.ps1` uses `HttpWebRequest` not `Invoke-WebRequest`; SKIP when Orca absent.

**PR-B — repoint + delete (closes #380):**
- Wire `setup-windows.ps1` post-setup (invoke `dotf doctor`; drop the `healthcheck.ps1`/`doctor.ps1` deploy + invoke blocks), `ci.yml` (`test-windows`), and the profile `hc` alias to `dotf doctor`.
- Resolve the **§4 residual** (below) so no coverage is lost.
- `git rm scripts/healthcheck.ps1 scripts/doctor.ps1` + their Pester; guard-grep clean for `(healthcheck|doctor)\.ps1`.

## Parity scan — healthcheck.ps1 (13 sections) vs dotf doctor

| § | covered by `dotf doctor`? |
|---|---|
| 1-3, 5-10, 12 | ✅ core/versioned/version-match, env, optional, vault, secrets, tmux, opencode, Antigravity |
| 4 Key Files / Junctions | ✅ mostly: `.dotfiles`, `.ssh/config`, claude-mem **BUG-014 + BUG-015** (`checkSymlinks`/`checkClaudeMem`). **Residual** (PR-B): `$PROFILE`, `.claude/CLAUDE.md`, `.gemini/AGY.md` existence; the **BUG-012** marketplace junction (the `.ps1` itself marks this *secondary*, superseded by the BUG-014 primary that dotf doctor already has — claude-mem-heal owns the repair). |
| 11 Repo–Deploy Drift | ➖ deferred → **CLI-019 (#488)** |
| 13 Orca Copilot Hook | ❌ → **PR-A ports it** |

## Out of scope

- PR-B's repoint + deletion (separate PR).
- §11 repo-deploy drift (CLI-019 / #488).
- `doctor.ps1` is the env-contract verifier — already covered by `dotf doctor`'s contract sweep; no new check needed, just deletion in PR-B.

## Risks / open questions

- **§4 residual must be settled before PR-B deletes `healthcheck.ps1`** — either add `$PROFILE`/`CLAUDE.md`/`AGY.md` existence + (optionally) the BUG-012 junction to `dotf doctor`, or consciously accept their drop. Decide in PR-B; PR-A does not delete anything, so no coverage is lost yet.
- `$PROFILE` resolution in Go is Windows-PowerShell-path-specific (pwsh vs WinPS 5.1) — needs care in PR-B.

## Acceptance criteria (PR-A)

- [ ] `dotf doctor` runs an "Orca Copilot hook (DX-006)" check: FAIL on `timeoutSec < 30`, FAIL on `Invoke-WebRequest` in the hook, PASS otherwise, SKIP when Orca absent.
- [ ] The check is registered in the full sweep (not `--quick`) and is cross-OS (skips cleanly off-Windows).
- [ ] Table-driven `go test` covers all branches; `go test ./...` green.
- [ ] No deletion / no caller repoint in this PR (that is PR-B).

## References

- Issue: mlorentedev/dotfiles#380 (CLI-018)
- AUDIT-007 Phase A / PR2: `docs/adr/audit-007-cli-convergence-state.md`
- Source check: `scripts/healthcheck.ps1` §13; DX-006 lesson in `docs/lessons.md`
- ADR-020/021 (convergence); CLI-012 (#376) original `dotf doctor` port
