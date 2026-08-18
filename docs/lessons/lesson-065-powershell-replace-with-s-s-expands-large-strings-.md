---
id: lesson-065-powershell-replace-with-s-s-expands-large-strings-
type: lesson
status: active
created: "2026-05-27"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 065: PowerShell -replace with [\s\S]*? expands large strings instead of replacing

**Context**: `setup-windows.ps1` profile-section block used `-replace` regex to update dotfiles section in PowerShell profile.

**Problem**: PowerShell's `-replace` operator with `[\s\S]*?` regex pattern **expands** large strings (>10KB) instead of replacing. A profile with 1 marker and 0 errors became 4 markers and 5 errors after a single `-replace` run, then 30+ markers on subsequent runs. Both `-replace` and `[regex]::Replace` failed identically.

**Solution**: Replace regex-based replace with index-based split/join: use `IndexOf()` to find marker positions, `Substring()` to extract before/after, and string concatenation. Also add `-NoNewline` to `Set-Content` to prevent trailing newline drift (2 bytes per run). Verified: 5 consecutive runs = 0 accumulation, 0 parse errors, constant size.

**Rule**: Never use PowerShell's `-replace` operator with `[\s\S]*?` on strings >10KB. Use `IndexOf()` + `Substring()` + concatenation instead.### [2026-05-27] Obsidian CLI package name @vorillaz/obsidian-cli does not exist on npm

**Context**: Both `setup-linux.sh` and `setup-windows.ps1` attempted `npm install -g '@vorillaz/obsidian-cli'` but the package returns 404 on npm registry.

**Problem**: The Obsidian CLI was referenced with a wrong package name (`@vorillaz/obsidian-cli`) that does not exist on npm. The correct package is `obsidian-cli` (no scope). Both Linux and Windows setup scripts silently failed to install it, causing `FAIL: Obsidian CLI not in PATH` in healthcheck.

**Solution**: Updated both setup scripts to use `npm install -g 'obsidian-cli'`. Updated all bats tests that grep for the old package name.

**Rule**: Always verify npm package names exist before committing them to setup scripts. `npm view <package> name version` before using in automated install.
