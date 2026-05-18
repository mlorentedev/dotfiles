---
tags: [spec, tasks, archived]
created: "2026-05-18"
archived: "2026-05-18"
---

# Tasks - BUG-003-copilot-cli-v2-detection

> Retroactive checklist. All items completed in PR #48 (commit `63b0716`) before this folder was created. Reverse-engineered from the merged diff for audit-trail consistency.

## Setup (post-hoc)

- [x] Branch created from main: `fix/BUG-003-copilot-cli-v2-detection`
- [ ] ~~Vault entry in 11-tasks.md added BEFORE branch~~ — SKIPPED at the time (the discipline violation that triggered SDD-001). Added post-merge as part of vault hygiene catchup, same 2026-05-18 session.
- [ ] ~~Spec folder scaffolded via init-spec.ps1 BEFORE coding~~ — SKIPPED at the time (this folder created retroactively).
- [x] User empirically confirmed (a) `copilot --version` returns "GitHub Copilot CLI 1.0.48"; (b) `~/.copilot/` has skills/, mcp-config.json, settings.json, permissions-config.json, copilot-instructions.md; (c) `gh extension install github/gh-copilot` errors with "matches built-in alias"; (d) `winget search copilot` resolves the new product as `GitHub.Copilot` (Moniker: copilot).

## Implementation (TDD order, as executed)

### Tests first (bats parity asserts)

- [x] `tests/setup-windows.bats`: new asserts — detects via `Get-Command copilot` (not `gh extension list`); winget tool array contains `GitHub.Copilot`; no `github/gh-copilot` literal references remain; parity with `setup-linux.sh` on `command -v copilot`
- [x] `tests/aliases.bats`: new asserts — `cop`/`cops` defined in both `.zsh/aliases.zsh` and `powershell/profile.ps1`; ghcs/ghce removed (anchored to definition forms only, not comments); cross-OS parity

### Cross-OS detection

- [x] `setup-windows.ps1`: replace `gh extension list | match 'github/gh-copilot'` block with `Get-Command copilot`. Add `@{ Name = "GitHub Copilot CLI"; Cmd = "copilot"; Id = "GitHub.Copilot" }` to the winget tools array. After the winget loop, refresh PATH: `$env:PATH = [Environment]::GetEnvironmentVariable("PATH", "Machine") + ";" + [Environment]::GetEnvironmentVariable("PATH", "User")` so the new binary is visible to the subsequent Copilot block (otherwise Get-Command misses it until next shell start).
- [x] `setup-linux.sh`: replace `gh extension list | grep -q "github/gh-copilot"` with `command -v copilot`. No auto-install (distros vary). Idempotent `sed -i` cleanup of stale `eval "$(gh copilot alias -- bash)"` from `.zshrc`/`.bashrc` if present (the subcommand does not exist in v2; would error silently every shell startup).

### Aliases rename

- [x] `powershell/profile.ps1`: drop `ghcs`/`ghce` functions. Add `Set-Alias -Name cop -Value copilot` + `function cops { copilot -p "$args" --allow-all-tools -s }`. Comment block explaining the breaking-change rationale (v2 has no `suggest`/`explain` subcommands; same names + different semantics = cognitive trap).
- [x] `.zsh/aliases.zsh`: same. Add `alias cop="copilot"` + `cops() { copilot -p "$*" --allow-all-tools -s; }`.

### Contract update

- [x] `env-contract.json`: add `{ "name": "copilot", "purpose": "GitHub Copilot CLI (agentic assistant; winget GitHub.Copilot on Windows, manual install on Linux)" }` to `optional_binaries`. Update `gh` purpose to "GitHub CLI (PR/issue management; Copilot CLI is the standalone `copilot` binary, not a gh extension since BUG-003)".

### Local verification

- [x] PSScriptAnalyzer (Error+Warning) on `setup-windows.ps1` and `powershell/profile.ps1` → clean
- [x] `bash -n setup-linux.sh && bash -n .zsh/aliases.zsh` → clean
- [x] `jq empty env-contract.json` → valid
- [x] Simulated bats grep-by-grep for all 9 new asserts → green
- [x] Empirical smoke on admin Windows: `pwsh -NoProfile -ExecutionPolicy Bypass -File setup-windows.ps1` → `[SUCCESS] copilot-instructions.md deployed successfully (verified pointer to AGENTS.md)` + `[SUCCESS] GitHub Copilot CLI configured (aliases cop/cops in profile.ps1)`

## Closing

- [x] All acceptance criteria from `proposal.md` (retroactive) marked green
- [x] PR #48 description references this work; commit message convention `fix(setup,aliases): detect new standalone Copilot CLI v2 (BUG-003)`
- [x] PR #48 merged 2026-05-18 (commit `49bb58e` merge of `63b0716`); branch deleted
- [x] Follow-up specs opened: AI-017 (Copilot skills port), AI-018 (Copilot MCP deploy), ADR-010 update
- [x] Spec folder created retroactively at `specs/archive/BUG-003-copilot-cli-v2-detection/` (this PR, `chore/BUG-003-retro-spec`)

## Decisions (post-hoc)

- **Aliases renamed (breaking change) rather than kept as ghcs/ghce with new internals**: user explicitly chose Option C of three (Socratic exchange captured in session transcript). Rationale: v2 has no `suggest`/`explain` subcommands; keeping the same alias names with different semantics ("`ghcs "delete logs"` v1 = print suggestion; v2 = agent might run rm") would be a cognitive trap. Breaking-change documented in commit body.
- **Auto-install via winget enabled** for Windows: user explicitly opted in (Option A of multi-select). Trade-off: less friction on new machines vs adds a winget dep at setup time. The dev-tools winget block already handles 5 other tools so the pattern was familiar.
- **No Linux auto-install**: distros vary too much (snap vs apt vs curl vs brew). Detect-and-act only; install URL in the info message.
- **AWS Copilot collision NOT defensively handled**: <1% population (almost no one has both AWS Copilot CLI and GitHub Copilot CLI). Documented inline as comment + verification.md note. BUG-004 if it surfaces.
