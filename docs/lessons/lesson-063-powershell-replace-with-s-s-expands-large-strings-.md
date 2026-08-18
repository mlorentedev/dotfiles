---
id: lesson-063-powershell-replace-with-s-s-expands-large-strings-
type: lesson
status: active
created: "2026-05-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 063: PowerShell -replace with [\s\S]*? expands large strings instead of replacing

**Context:** setup-windows.ps1 profile-section block used -replace regex to update dotfiles section in PowerShell profile
**Problem:** PowerShell's -replace operator with [\s\S]*? regex pattern EXPANDS large strings (>10KB) instead of replacing. A profile with 1 marker and 0 errors became 4 markers and 5 errors after a single -replace run, then 30+ markers on subsequent runs. The -replace and [regex]::Replace both failed — same behavior. Root cause: PowerShell regex engine backtracking on large strings with non-greedy [\s\S]*?.
**Solution:** Replace regex-based replace with index-based split/join: use IndexOf() to find marker positions, Substring() to extract before/after, and string concatenation to build the result. Also add -NoNewline to Set-Content to prevent trailing newline drift (2 bytes per run). Verified: 5 consecutive runs = 0 accumulation, 0 parse errors, constant size.
**Tags:** `#powershell` `#regex` `#replace` `#idempotency` `#profile` `#bug`
