---
id: lesson-117-resolving-windows-profile-from-go-must-include-the
type: lesson
status: active
created: "2026-06-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 117: Resolving Windows `$PROFILE` from Go must include the OneDrive-redirected Documents root

**Context**: Porting healthcheck §4 (`$PROFILE` existence) into `dotf doctor`. Go has no `$PROFILE` intrinsic, so the check enumerates candidate paths.

**Problem**: A naive `~\Documents\{PowerShell,WindowsPowerShell}\Microsoft.PowerShell_profile.ps1` check false-FAILs on corporate Windows, where Documents is frequently redirected to `~\OneDrive\Documents` by Known Folder Move. The profile is present; the check reports "missing".

**Solution**: Enumerate `{Documents, OneDrive\Documents} × {PowerShell (pwsh 7), WindowsPowerShell (5.1)}` and PASS on any hit. Chosen over shelling out to `pwsh -Command '$PROFILE'` — pure-Go + deterministic fits the doctor's temp-tree test model and avoids a SKIP-when-pwsh-absent branch.

**Rule**: Any Go (or non-PowerShell) check that reconstructs a Windows user-profile path must account for OneDrive Known Folder redirection of Documents — `%USERPROFILE%\Documents` alone is wrong on managed/corporate boxes. Enumerate both roots, or ask PowerShell for the real path.
