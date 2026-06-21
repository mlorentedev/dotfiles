---
id: "CLI-020-dotf-init-windows-repoint"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-21"
issue: "mlorentedev/dotfiles#489"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-020-dotf-init-windows-repoint

> AUDIT-007 Phase A / PR4 — the first *real* `.ps1` deletion of the CLI-convergence
> strangler-fig. Repoint the two Windows callers of the init scripts to `dotf init`
> and delete the three init `.ps1` + their bats.

## Why

<!-- from issue #489: CLI-020: dotf init — Windows repoint + delete init .ps1 (first real deletion) -->

`dotf init` is built at parity (ADR-022 / CLI-014) and `dotf` is installed on Windows (WIN-006), yet Windows still **deploys and invokes** the init `.ps1` twins. This is the exact asymmetry AUDIT-007 found: Linux repointed its callers and deleted the init `.sh` (CLI-014), but the Windows `.ps1` were kept "until `dotf` exists on Windows" — and that prerequisite is now met. While both coexist, the `.ps1` count never drops and the twins are orphans (no `.sh` peer keeps them honest). This is the cleanest first deletion because the Go side is already at parity — only two callers need repointing.

## What

After this PR, on Windows:

- The `project-init` profile function (`powershell/profile.ps1:87`) invokes **`dotf init`** instead of shelling out to `scripts/init-project.ps1` (`profile.ps1:110`).
- `setup-windows.ps1` no longer **deploys** `init-project.ps1` to `$ScriptsDir` / `$ClaudeHome` (`:1436-1442`); instead it **removes** any prior orphan copy, mirroring the Linux precedent (`setup-linux.sh` `rm -f "$HOME/.claude/init-project.sh"`, CLI-014).
- The three init scripts are deleted — their behavior is now `dotf init` / `dotf init agents` / `dotf init github`:
  - `scripts/init-project.ps1` → `dotf init [path] --stack <x>`
  - `scripts/init-repo-agents.ps1` → `dotf init agents` (was called by init-project.ps1:440)
  - `scripts/init-repo-github-defaults.ps1` → `dotf init github` (was called by init-project.ps1:448)
- `tests/init-project-ps1.bats` + `tests/init-repo-github-defaults.bats` removed; `ci.yml` no longer references them (`:55`, `:253`).

## Out of scope

- The other nouns (`doctor` CLI-018/019, `vault`, `secrets`, …) — separate roadmap PRs.
- Changing `dotf init`'s Go behavior beyond what the parity gate requires. If a Windows gap is found, fix it FIRST in its own change, then resume the deletion.
- The Linux init side — already done (CLI-014).

## Risks / open questions

- **Parity gate — RESOLVED ✅ (green).** Empirically verified on this Windows box with an isolated `VAULT_PATH=$tmp dotf init <tmp> --stack go --skip-github`: `ResolveVault` honored `$VAULT_PATH` (cascade env → machine.json → contract, `vault.go:45`), the entry was created at `<VAULT>/10_projects/<repo>` (`project.go:62`), seeding `00-context.md` + `10-roadmap.md` + `memory/MEMORY.md` — superset of, and more correct than, the `.ps1` (which seeds a stale `11-tasks.md`, an ADR-018 violation). The Windows memory junction is delegated to `claude-session-start.ps1:236` `Ensure-MemoryJunction` (runs every session); the transient gap before the first session is harmless (the junction's only consumer is a Claude session). **No Go change required — pure repoint+delete.**
- **Sub-command parity.** `dotf init agents` / `dotf init github` must cover what `init-repo-agents.ps1` / `init-repo-github-defaults.ps1` did (the non-fatal degradation at init-project.ps1:443/454 — "continue on failure"). Confirm.
- **Arg mapping.** `project-init <name> <stack>` (profile.ps1:98-100) must map cleanly to `dotf init <path> --stack <stack>`. Confirm flag names/values.
- **Binary freshness.** The Windows `dotf` is 0.9.1; repo is 0.9.4. Ensure setup installs a `dotf` new enough to carry `init` at parity before the profile repoint is load-bearing.

## Acceptance criteria

- [ ] `powershell/profile.ps1` `project-init` calls `dotf init` (path + `--stack` mapped); zero reference to `init-project.ps1`.
- [ ] `setup-windows.ps1` drops the init-project.ps1 deploy block and adds orphan cleanup (`rm` prior `$ScriptsDir\init-project.ps1` + `$ClaudeHome\init-project.ps1`).
- [ ] `scripts/init-project.ps1`, `scripts/init-repo-agents.ps1`, `scripts/init-repo-github-defaults.ps1` deleted; `tests/init-project-ps1.bats` + `tests/init-repo-github-defaults.bats` deleted; `ci.yml:55,253` updated.
- [ ] Parity verified with evidence: `dotf init` on Windows produces the same `10_projects/<repo>` vault entry as the retired `.ps1` (captured in `verification.md`).
- [ ] Guard-grep clean for `init-(project|repo-agents|repo-github-defaults)\.ps1` (excluding `CHANGELOG.md`, archived `specs/`, `docs/adr/audit-*`).
- [ ] `test-windows` CI job green.

## References

- Issue: mlorentedev/dotfiles#489
- AUDIT-007 Phase A / PR4: `docs/adr/audit-007-cli-convergence-state.md`
- ADR-020 (convergence boundary), ADR-021 (roadmap), ADR-022 / CLI-014 (`dotf init` parity), ADR-025 (cross-machine path resolution)
- Linux precedent: `setup-linux.sh` init-project.sh orphan cleanup (CLI-014)
- Parity target: `cli/internal/vault/project.go`

<!-- archived 2026-06-21 — PR: https://github.com/mlorentedev/dotfiles/pull/500 -->
