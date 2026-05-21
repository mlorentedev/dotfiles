---
tags: [spec, tasks, opencode, multi-agent, cross-os-parity]
created: "2026-05-21"
---

# Tasks - AI-014-opencode-windows-bootstrap

## Setup

- [x] Branch `feat/AI-014-opencode-windows-bootstrap` (off main).
- [x] Spec scaffolded via `init-spec.ps1` (vault gate satisfied — entry exists in `10_projects/dotfiles/11-tasks.md`).
- [x] Empirical pre-flight 2026-05-21: `winget search opencode` → `SST.opencode v1.15.6` available; `opencode.ai/install.ps1` returns HTTP 404 (no PowerShell installer upstream); npm registry has `opencode-ai v1.15.7` as alternative. Winget is the preferred channel.

## Implementation (TDD order)

### Tests first

- [ ] `tests/setup-windows.bats`: assertion locks `SST.opencode` entry in the `$tools` array (section 1c).
- [ ] `tests/setup-windows.bats`: assertion locks the reconcile-not-skip block for `opencode.jsonc` (greps `Get-FileHash` or `Compare-Object` near `opencode.jsonc` path).
- [ ] `tests/setup-windows.bats`: assertion locks the commands sync block (greps `opencode\\commands` and an orphan-removal loop).
- [ ] Run bats — should FAIL on all three (red).

### Implementation

- [ ] `setup-windows.ps1` section 1c: append `@{ Name = "OpenCode"; Cmd = "opencode"; Id = "SST.opencode" }` to the `$tools` array. Idempotent via existing `Get-Command opencode` guard in the foreach.
- [ ] `setup-windows.ps1` new section 2d (after 2c Obsidian CLI): deploy `opencode.jsonc` reconcile-not-skip pattern. Compute SHA256 of source and destination; copy only on mismatch. Bootstrap branch creates `$env:USERPROFILE\.config\opencode\` if missing.
- [ ] `setup-windows.ps1` section 2d: sync `ai\opencode\commands\*.md` → `$env:USERPROFILE\.config\opencode\commands\` (add new, leave unchanged, remove orphans). Mirror the Linux loop counts (`cmds_added`/`cmds_skipped`/`cmds_removed`).
- [ ] `scripts/healthcheck.ps1` sec 10/12: keep current SKIP wording (Test-Command 'opencode' auto-flips to PASS on next run once installed). Remove only the `AI-014 pending` parenthetical.
- [ ] Run bats — assertions GREEN.

### Lint + cross-check

- [ ] PowerShell AST `[Parser]::ParseFile` on `setup-windows.ps1` + `scripts/healthcheck.ps1` clean.
- [ ] `Invoke-ScriptAnalyzer -Settings .PSScriptAnalyzerSettings.psd1 -Severity Error,Warning` clean on both files (no em-dash regression).
- [ ] Manual empirical run on user's Windows: `setup-windows.ps1` exit 0; `opencode --version` returns a 1.15.x string; `~/.config/opencode/opencode.jsonc` SHA256 matches `ai/opencode/opencode.jsonc`; `ls ~/.config/opencode/commands/` shows 12 `.md` files.

## Closing

- [ ] `verification.md` filled with empirical evidence (winget install transcript, version output, file-hash equality).
- [ ] PR opened referencing `specs/AI-014-opencode-windows-bootstrap/`. Body calls out the winget-vs-curl simplification.
- [ ] Post-merge: archive `specs/AI-014-…` to `specs/archive/`.
- [ ] Post-merge: tick vault `11-tasks.md` AI-014 entry → ✓ with PR link.
- [ ] Post-merge: update `40-runbooks/guide-opencode-go-setup.md` with a small "Windows install delta" section (winget package ID, expected default location, profile alias `oc` activation).
