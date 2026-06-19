---
tags: [spec, tasks]
created: "2026-06-18"
---

# Tasks - WIN-006-windows-dotf-install

> One PR off main. Closes #451. Surfaced during the ADR-025 deploy (#446/#450).

## Setup
- [x] Branch `feat/win-install-dotf` off the latest `origin/main` (release 0.6.0, `DOTF_VERSION=0.6.0`).

## Implementation
- [x] `scripts/install-dotf.ps1`: `Get-DotfArch`, `Get-DotfVersion`, `Install-Dotf` (fetch+sha256+extract→`~/.local/bin`, idempotent, never-throws), standalone run-guard.
- [x] Wire `setup-windows.ps1`: dot-source + `Install-Dotf` (non-fatal) before the hive/`dotf env` steps.
- [x] `tests/install-dotf-ps1.bats`: structural + convention (the repo's .ps1 test style).

## Closing
- [x] `install-dotf.ps1` + `setup-windows.ps1` parse clean; PSScriptAnalyzer (CI `.PSScriptAnalyzerSettings.psd1`) = 0.
- [x] Dot-source guard: dot-sourcing defines `Install-Dotf` without auto-installing.
- [x] Real smoke: `.\scripts\install-dotf.ps1` installs `dotf 0.6.0`; `dotf version` + `dotf env path VAULT_PATH` OK.
- [x] `bats tests/install-dotf-ps1.bats` → 11/11 green.
- [ ] PR opened referencing this spec folder; closes #451.

## Machine-readable features
`features.json` is emitted alongside; the harness sets `"state": "passing"` after a green `verification` command.
