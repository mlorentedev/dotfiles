---
tags: [spec, tasks]
created: "2026-06-21"
---

# Tasks - CLI-020-dotf-init-windows-repoint

> Repoint + delete (strangler-fig contact), not new-feature TDD. The "red" is the
> pre-deletion parity gate; the "green" is the guard-grep oracle + CI. One task ≈
> one focused change.

## Setup

- [x] Branch created from main: `feat/dotf-init-windows-repoint`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] No open questions in `proposal.md` — the parity gate is **resolved green** (empirical isolated `dotf init` run)

## Implementation

- [x] **Parity gate (pre-deletion):** isolated `VAULT_PATH=$tmp dotf init <tmp> --stack go --skip-github` — confirmed VaultPath resolves on Windows, entry at `10_projects/<repo>` with `00-context.md` + `10-roadmap.md` + `memory/MEMORY.md` (superset of, and more correct than, the `.ps1`; junction delegated to `Ensure-MemoryJunction`)
- [x] Repoint `powershell/profile.ps1` `project-init` → `dotf init $ProjectName --stack $Stack` (was `& init-project.ps1`)
- [x] `setup-windows.ps1`: replace the `init-project.ps1` deploy block with orphan cleanup of all 3 init `.ps1` from `$ScriptsDir`/`$ClaudeHome` (mirrors `setup-linux.sh` CLI-014)
- [x] `tests/setup-windows.bats`: flip the deploy assertion to a cleanup assertion (no `Copy-Item init-project.ps1`)
- [x] Delete `scripts/init-project.ps1`, `scripts/init-repo-agents.ps1`, `scripts/init-repo-github-defaults.ps1`
- [x] Delete `tests/init-project-ps1.bats`, `tests/init-repo-github-defaults.bats`
- [x] Remove `ci.yml` references: PSScriptAnalyzer list (`:55`) + bats run list (`:253`)
- [x] Honesty pass on docs: `README.md` tree, `docs/runbooks/ai-tools-setup.md`, `docs/troubleshooting/ai-tools.md`, `powershell/profile.ps1` PATH comment
- [x] Guard-grep clean for `init-(project|repo-agents|repo-github-defaults)\.ps1` (only the intentional cleanup list + updated bats + historical records remain)

## Closing

- [ ] `test-windows` CI green (bats + PSScriptAnalyzer)
- [x] No scope creep — `cli/internal/initrepo/templates/agents-spec-section.md:18` (stale `init-repo-agents.ps1`/#380 mention) **deliberately left**: it is a vault-SSOT, drift-tested template (`drift_test.go:32`); fixing it belongs to **#461** (re-vendor templates), not here
- [ ] `verification.md` filled with evidence
- [ ] PR opened referencing this spec folder + closing #489

## Known residual (tracked, not this PR)

- `cli/internal/initrepo/templates/agents-spec-section.md:18` → fold into #461.
- `docs/adr/adr-009-multi-agent-runtime.md:73` → historical ADR narrative (past AI-011 bootstrap event); left as audit-trail.
