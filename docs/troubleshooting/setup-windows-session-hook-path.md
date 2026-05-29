---
id: "dotfiles-troubleshoot-setup-windows-session-hook-path"
type: troubleshooting
status: resolved
tags: [troubleshooting, dotfiles, claude-code, windows, hooks, setup]
created: "2026-05-15"
owner: manu
---

# Troubleshooting: SessionStart hook on Windows points to non-existent path

`setup-windows.ps1` deployed `claude-session-start.ps1` to `~\scripts\` but registered the Claude Code `SessionStart` hook in `~\.claude\settings.json` pointing at `~\.dotfiles\scripts\claude-session-start.ps1`. The script never landed where the hook expected it, so every Claude Code session on Windows started with a non-blocking PowerShell error and the hook silently never ran — losing vault detection, hive project context, `specs/` state surfacing, and `claude-mem-heal`.

Tracked in [mlorentedev/dotfiles#20](https://github.com/mlorentedev/dotfiles/issues/20).

## Symptom

On `claude` invocation:

```
Failed with non-blocking status code: The argument
'C:\Users\<user>\.dotfiles\scripts\claude-session-start.ps1' is not
recognized as the name of a script file. Check the spelling of the name,
or if a path was included, verify that the path is correct and try again.
```

Sessions continue (non-blocking), but the hook never executes.

## Root cause

Two inconsistent paths for the same script inside `setup-windows.ps1`:

| Line | Code | Effect |
|------|------|--------|
| 39 | `$ScriptsDir = "$env:USERPROFILE\scripts"` | Deployment target |
| 621 | `Copy-Item $sessionStartSource "$ScriptsDir\" -Force` | Lands at `~\scripts\claude-session-start.ps1` |
| 692 | `$sessionStartCmd = "$DotfilesDest\scripts\claude-session-start.ps1"` | Hook points at `~\.dotfiles\scripts\…` (never deployed there) |

Per [adr-005-two-directory-sync](../adr/adr-005-two-directory-sync.md), only `load-secrets.ps1` is intentionally placed under `~\.dotfiles\scripts\` because shell profiles dot-source it from that fixed location. Everything else lives in `~\scripts\`. The hook registration was the lone outlier.

### Compounding bug — sticky skip

The hook registration was guarded by an **unconditional skip if a `SessionStart` entry already exists**:

```powershell
if ($settings.hooks -and $settings.hooks.SessionStart) {
    Write-Info "SessionStart hook already configured, skipping"
}
```

So once a wrong path landed in `settings.json`, re-running `setup-windows.ps1` would silently leave the broken hook intact — no self-healing.

## Fix

Two surgical changes in `setup-windows.ps1` (see commit on `main`):

1. **Point the hook at the deploy directory** (`$ScriptsDir`, not `$DotfilesDest\scripts`).
2. **Replace the unconditional skip with an "update if differs" check** that compares the existing hook command against the expected one and rewrites it when they diverge. Existing buggy installs now self-heal on the next `setup-windows.ps1` run.

```powershell
$sessionStartCmd = "$ScriptsDir\claude-session-start.ps1"
$expectedHookCommand = "pwsh -NoProfile -File `"$sessionStartCmd`""

# ...

if ($existingHookCommand -eq $expectedHookCommand) {
    Write-Info "SessionStart hook already correctly configured, skipping"
} else {
    if ($existingHookCommand) {
        Write-Info "SessionStart hook points to '$existingHookCommand'; updating to '$expectedHookCommand'"
    }
    # ...rewrite the hook entry...
}
```

## Local workaround (before re-running setup)

If you cannot re-run `setup-windows.ps1` immediately, copy the script to where the broken hook currently expects it:

```powershell
Copy-Item "$env:USERPROFILE\Projects\dotfiles\scripts\claude-session-start.ps1" "$env:USERPROFILE\.dotfiles\scripts\" -Force
```

This unblocks the hook for the current install. Re-running `setup-windows.ps1` from the fixed branch makes the workaround obsolete and rewrites the hook to the canonical `~\scripts\` path.

## Verification

After re-running `setup-windows.ps1`:

```powershell
(Get-Content "$env:USERPROFILE\.claude\settings.json" -Raw | ConvertFrom-Json).hooks.SessionStart[0].hooks[0].command
# expected: pwsh -NoProfile -File "C:\Users\<user>\scripts\claude-session-start.ps1"

Test-Path "$env:USERPROFILE\scripts\claude-session-start.ps1"  # expected: True
```

Start a new `claude` session — no PowerShell error, hook output appears at session start.

## Follow-ups

- BATS suite (`tests/setup-windows.bats`) currently only asserts the strings `'claude-session-start.ps1'` and `'SessionStart'` appear in the script — too weak to catch deploy-path / hook-path divergence. Worth a follow-up test asserting that the hook command path matches the deploy target. Tracked alongside the two-tier deploy lesson (`lesson_dotfiles_two_tier_deploy`, maintainer's knowledge store).
- WIN-002 smoke-sweep sprint is the class of QA pass meant to surface bugs like this ahead of release.

## Related

- [adr-005-two-directory-sync](../adr/adr-005-two-directory-sync.md) — the two-directory split that motivates the `load-secrets` special case.
- `lesson_dotfiles_two_tier_deploy` — broader lesson on the two-tier deploy convention (maintainer's knowledge store).
