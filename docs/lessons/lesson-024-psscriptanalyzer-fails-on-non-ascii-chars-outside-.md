---
id: lesson-024-psscriptanalyzer-fails-on-non-ascii-chars-outside-
type: lesson
status: active
created: "2026-03-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 024: PSScriptAnalyzer fails on non-ASCII chars outside here-strings

**Context**: Added `-WorkSdk` mode to `init-project.ps1` with em dashes (`—`) in `Write-Host` strings and an arrow (`→`) in a `Write-Success` call. Also added an em dash in a comment line.

**Problem**: PSScriptAnalyzer flags any non-ASCII character that appears in `.ps1` files outside of here-strings (`@"..."@`). The lint-powershell CI job (`Invoke-ScriptAnalyzer -Severity Error,Warning`) caught it and exited 1. Syntax check (test 84) passed because PowerShell parses non-ASCII fine — only PSScriptAnalyzer rejects them.

**Solution**: Replace `—` with `-` and `→` with `->` in string literals, `Write-*` calls, and comments. Non-ASCII inside `@"..."@` here-strings is fine (pre-existing `→` on line 288 was already passing).

**Rule**: In `.ps1` files, keep all non-here-string code (comments, regular quoted strings, `Write-*` calls) to pure ASCII. Em dashes, arrows, and similar typography must live only inside `@"..."@` blocks (template content). When adding display text to a PowerShell script, use ASCII punctuation: `-` not `—`, `->` not `→`.
