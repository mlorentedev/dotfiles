---
id: "BUG-005-setup-ps7-reexec"
type: spec
status: archived
created: "2026-05-19"
archived: "2026-05-19"
merged_pr: 58
tags: [spec, proposal, bug, windows, powershell, portability]
template_version: "1.0"
---

# BUG-005-setup-ps7-reexec

## Why

<!-- from 11-tasks.md: BUG-005-setup-ps7-reexec *(P1, opens 2026-05-19, Windows-only)* — Merge-ClaudeSettings uses ConvertFrom-Json -AsHashtable which is PS 7+ only; silently no-ops on Windows PowerShell 5.1. -->

SDD-002 (PR #51) introduced `Merge-ClaudeSettings` in `setup-windows.ps1`. The helper calls `ConvertFrom-Json -AsHashtable` — a parameter added in PowerShell 7.0 that **does not exist in Windows PowerShell 5.1**. The default `PowerShell` interpreter shipped with Windows resolves to 5.1; modern `pwsh` (7+) is a separate install. When the user invokes `PowerShell -ExecutionPolicy Bypass -File .\setup-windows.ps1` (the natural Windows command), `Merge-ClaudeSettings` hits a `ParameterBindingException` inside its `try { ... } catch { Write-Warn "...not valid JSON..." ... return }` block — the catch is too wide and misclassifies the parameter error as a JSON parse failure. The function returns early; the per-key merge from `ai/claude/settings.json` is **silently skipped** on PS 5.1. Existing `~/.claude/settings.json` survives untouched, so the bug is invisible until a fresh-machine bootstrap depends on the template merge. Empirically observed 2026-05-19: `[WARNING] Claude settings template is not valid JSON after placeholder substitution: A parameter cannot be found that matches parameter name 'AsHashtable'` in the setup output of an admin Windows 11 machine running Windows PowerShell 5.1.22621.

## What

`setup-windows.ps1` gains a preamble (after `[CmdletBinding()]` and `param()`, before any helper definitions) that detects `$PSVersionTable.PSVersion.Major -lt 7`. If `Get-Command pwsh -ErrorAction SilentlyContinue` resolves to PowerShell 7+, the script **re-executes itself under pwsh** with `-NoProfile -ExecutionPolicy Bypass -File $PSCommandPath` plus `@args` (forwards any positional arguments) and exits with the child's exit code. If pwsh is not installed, the script prints an actionable error (`Install via: winget install Microsoft.PowerShell`) and exits with code 1 (fail-loud, no silent skip).

Observable post-PR behaviour:

1. `PowerShell -ExecutionPolicy Bypass -File .\setup-windows.ps1` on a Windows 5.1 machine with pwsh installed: prints one `[INFO] Windows PowerShell 5.1 detected; re-executing under pwsh ...` line then proceeds normally; `Merge-ClaudeSettings` succeeds (no more "not valid JSON" warning).
2. Same command without pwsh installed: prints `[ERROR]` lines + winget install hint + exits 1. No partial deploy.
3. `pwsh -NoProfile -File .\setup-windows.ps1` (already under PS 7+): preamble no-ops; script proceeds normally. No double-reexec.

## Out of scope

- Backfilling PS 5.1 compatibility into `Merge-ClaudeSettings` itself (PSCustomObject → hashtable conversion, line-based JSON ops). Cleaner to require PS 7 and re-exec.
- Auto-installing `pwsh` from inside this script. Out of scope: install is a one-time user action; the error message points at `winget install Microsoft.PowerShell`.
- Linux side: `setup-linux.sh` uses bash + jq, no shell-version coupling. Verified immune by construction.
- Bundling with BUG-004 (truncate guard). Separate atomic PR; this one is Windows-only and ~30 LOC.

## Risks / open questions

- **Re-exec arg forwarding**: `$PSCommandPath` resolves to the current script's absolute path. `@args` forwards positional arguments. Named parameters defined in `param()` would need explicit forwarding via `$PSBoundParameters` if any were added later. Current `setup-windows.ps1` has an empty `param()`, so `@args` suffices.
- **Re-exec infinite loop**: if pwsh is misconfigured (e.g. aliased back to PS 5.1 on PATH), the re-exec could loop. Guard: the preamble's `$PSVersionTable.PSVersion.Major -lt 7` check runs again under the child; if the child is also 5.1, it cannot find pwsh either (since the parent already failed that branch), so it exits 1. Loop impossible in practice.
- **Hook system**: SessionStart hook command in deployed `~/.claude/settings.json` uses `pwsh -NoProfile -File ...` already (set by SDD-002 template), so the hook is unaffected. The setup script itself was the only PS 5.1 entry point that could hit BUG-005.
- **Existing PS 5.1 users**: anyone running `PowerShell -File setup-windows.ps1` without pwsh installed gets a hard error after this PR vs. a silent skip before. This is a behavior change, but it's the correct behavior — the silent skip was the bug.

## Acceptance criteria

- [ ] `setup-windows.ps1` invoked under Windows PowerShell 5.1 with `pwsh` (7+) installed re-executes itself under pwsh, observable by an `[INFO]` line in the output, and produces `[SUCCESS] Claude settings.json merged from template ...` (no `[WARNING] ... AsHashtable` line).
- [ ] `setup-windows.ps1` invoked under Windows PowerShell 5.1 with `pwsh` NOT installed exits with code 1 and prints an actionable `[ERROR]` line referencing `winget install Microsoft.PowerShell`.
- [ ] `setup-windows.ps1` invoked under pwsh (7+) directly does NOT re-exec (preamble is no-op); existing behavior preserved.
- [ ] bats grep asserts the preamble exists and references the three branches (PS 5.1 + pwsh, PS 5.1 no pwsh, PS 7+ direct).
- [ ] PSScriptAnalyzer clean on changes.
- [ ] No Linux changes (`setup-linux.sh` untouched, asserted by absence of any diff in that file).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (BUG-005 backlog entry)
- Related: SDD-002 (PR #51) — introduced `Merge-ClaudeSettings` with the `-AsHashtable` dependency
- Sibling: `specs/BUG-004-claude-mem-truncate-guard/` (PR #57) — different bug, same setup-script root file
- Upstream: [PowerShell 7.0 release notes](https://learn.microsoft.com/en-us/powershell/scripting/whats-new/what-s-new-in-powershell-70) — `ConvertFrom-Json -AsHashtable` added
