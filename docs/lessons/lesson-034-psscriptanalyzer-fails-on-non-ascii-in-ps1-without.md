---
id: lesson-034-psscriptanalyzer-fails-on-non-ascii-in-ps1-without
type: lesson
status: active
created: "2026-05-16"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 034: PSScriptAnalyzer fails on non-ASCII in .ps1 without BOM

**Context:** Editing PowerShell scripts (`setup-windows.ps1`, `powershell/profile.ps1`, `scripts/*.ps1`) in this repo. CI runs PSScriptAnalyzer in the `lint-powershell` job with default rules.
**Problem:** Any non-ASCII character (em dash `—`, en dash `–`, arrows `→`, smart quotes `"" ''`, ellipsis `…`) in a `.ps1` file without a BOM trips the rule `PSUseBOMForUnicodeEncodedFile`, fails the lint-powershell CI job, and blocks the PR. Hit twice in two months: commit `464eecf` (Mar 2026, em dash in `setup-windows.ps1`) and commit `9d284b9` (May 2026 PR #36, em dash in `powershell/profile.ps1`). The bug surfaces only in CI — local edits and grep look fine, the byte sequence is just multi-byte UTF-8.</problem>
<parameter name="solution">Project policy: **ASCII-only in `.ps1` files; do not add a BOM**. Substitutions when writing/editing: em dash -> `--`, arrows -> `->`, smart quotes -> ASCII `'` `"`, ellipsis -> `...`. Pre-commit safety net: `grep -nP '[^\x00-\x7F]' powershell/*.ps1 scripts/*.ps1 setup-windows.ps1` must return zero hits. Comments in `.ps1` are as constrained as code — an em dash in a comment block is enough to fail CI. Note `.sh`, `.md`, and vault files are NOT subject to this constraint.</solution>
<parameter name="tags">["powershell", "ci", "lint", "psscriptanalyzer", "encoding"]
**Solution:**
