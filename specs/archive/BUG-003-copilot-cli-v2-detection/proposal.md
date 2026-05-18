---
id: "BUG-003-copilot-cli-v2-detection"
type: spec
status: archived
created: "2026-05-18"
archived: "2026-05-18"
tags: [spec, proposal, bug, copilot, detect-and-act]
template_version: "1.0"
---

# BUG-003-copilot-cli-v2-detection

> **Retroactive spec.** PR #48 (commit `63b0716`) shipped before this spec folder was created — a deviation from `pattern-spec-driven-development.md` that directly triggered SDD-001 (PR #49). This document captures what was done, post-hoc, per the pattern's emergency-hotfix retroactive clause (line 121-122). Audit trail honesty over after-the-fact fiction.

## Why

Empirical discovery on the admin Windows machine 2026-05-18 while validating BUG-001 (#40): GitHub Copilot ships TWO different products under the same conceptual name:

1. **Legacy (v1, ~2023-2024)** — `gh extension install github/gh-copilot`. Wrapper around suggest/explain API.
2. **New (v2, 2025+)** — standalone `copilot` binary, winget `GitHub.Copilot`. Agentic interface (closer to Claude Code).

`dotfiles` setup scripts only detected v1 (`gh extension list | grep github/gh-copilot`). On machines with v2 installed and operative, the script logged `extension not installed, skipping` and never deployed `~/.copilot/copilot-instructions.md`. The AI-013 pointer-style refactor was functionally inert on those machines. Compounded by GitHub's own confusing CLI: trying to install the legacy extension on a machine with v2 yields `"copilot" matches the name of a built-in command or alias`.

## What

Seven-file diff (134/63 lines) across the dotfiles repo:

1. `setup-windows.ps1`: detect via `Get-Command copilot` instead of `gh extension list`; add `GitHub.Copilot` to the winget dev-tools auto-install array; refresh PATH after the winget loop so newly-installed binaries are visible to subsequent setup blocks (load-bearing for the Copilot deploy block).
2. `setup-linux.sh`: same detection swap via `command -v copilot`; no auto-install (Linux distros vary); idempotent `sed -i` cleanup of the stale `eval "$(gh copilot alias -- bash)"` line from `.zshrc`/`.bashrc` (the subcommand does not exist in v2 and errors silently on every shell startup).
3. `powershell/profile.ps1` + `.zsh/aliases.zsh`: rename `ghcs`/`ghce` → `cop`/`cops`. Old aliases wrapped `gh copilot suggest|explain` which do not exist in v2. Same names with different semantics would be a cognitive trap (`ghcs "delete logs"` v1 = suggest a command; v2 = the agent might execute `rm`).
4. `env-contract.json`: add `copilot` to `optional_binaries`; update `gh` purpose to reflect that Copilot is no longer a `gh` extension.
5. `tests/setup-windows.bats` + `tests/aliases.bats`: 9 new parity asserts locking the new detection path + alias rename + absence of legacy references.

## Out of scope (deferred)

- AWS Copilot CLI (`Amazon.CopilotCLI`) name collision — both install as `copilot`. If both present, `Get-Command` resolves to whichever is first on PATH. Documented as inline comment; <1% population. BUG-004 if it surfaces.
- Linux auto-install of `copilot` — distros vary (snap, apt, curl, brew). The detect-and-act info message points to docs.
- `copilot init` repo integration vs our `init-repo-agents.{sh,ps1}` — needs its own spec when relevant.
- v2 skills / MCP / hooks audit — opened as `AI-017` (skills surface) and `AI-018` (MCP deploy from `mcp-servers.json` SSOT), both pending. ADR-010 re-audit (matrix update) follows AI-017 + AI-018.

## Acceptance criteria (all green at archive)

- [x] Setup logs `[INFO] GitHub Copilot CLI already installed` (winget recognises v1.0.48) on the admin machine
- [x] After PATH refresh, `Get-Command copilot` resolves to the WinGet packages path
- [x] `[SUCCESS] copilot-instructions.md deployed successfully (verified pointer to AGENTS.md)` line appears
- [x] `[SUCCESS] GitHub Copilot CLI configured (aliases cop/cops in profile.ps1)` line appears (no `ghcs`/`ghce` references in any log line)
- [x] BUG-002 (#47) verify-strings regression-free in the same setup run (CLAUDE.md + GEMINI.md both green)
- [x] WIN-003 self-heal idempotent no-op when hook already correct
- [x] CI 5/5 green on PR #48: GitGuardian, integration, lint, lint-powershell, test (bats)

## References

- PR: [#48](https://github.com/mlorentedev/dotfiles/pull/48) (`fix(setup,aliases): detect new standalone Copilot CLI v2 (BUG-003)`)
- Sibling PR: [#47](https://github.com/mlorentedev/dotfiles/pull/47) (BUG-002, same audit session)
- Triggered: SDD-001 (PR #49) — the audit found this PR bypassed SDD discipline, leading to the discipline gate work
- Vault entry: `10_projects/dotfiles/11-tasks.md` "BUG-003-copilot-cli-v2-detection"
- Lesson: `10_projects/dotfiles/90-lessons.md` 2026-05-18 entry "detect-and-act scripts go silently inert when upstream products change their surface"
- Troubleshooting: `10_projects/dotfiles/50-troubleshooting/copilot-cli-v1-vs-v2-detection.md`
- ADR-010 (pending update): cross-agent harness parity matrix — v2 changes several Copilot column cells
