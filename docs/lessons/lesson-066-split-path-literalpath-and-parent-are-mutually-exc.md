---
id: lesson-066-split-path-literalpath-and-parent-are-mutually-exc
type: lesson
status: active
created: "2026-05-27"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 066: Split-Path -LiteralPath and -Parent are mutually exclusive parameter sets in PowerShell

**Context:** Fixing setup-windows.ps1 post-run errors. utils.ps1:58 used Split-Path -LiteralPath $Destination -Parent which threw Parameter set cannot be resolved.
**Problem:** Split-Path -LiteralPath and -Parent are mutually exclusive parameter sets in PowerShell 5.1 and 7.x. Using both together throws: "Parameter set cannot be resolved using the specified named parameters."
**Solution:** Split-Path has distinct parameter sets: LiteralPathSet (accepts -LiteralPath WITHOUT -Parent) and ParentSet (accepts -Path WITH -Parent). Replace -LiteralPath with -Path when using -Parent. The fix: $dstDir = Split-Path -Path $Destination -Parent (1 line change).
**Tags:** `#powershell` `#bugfix` `#cross-platform` `#split-path`
