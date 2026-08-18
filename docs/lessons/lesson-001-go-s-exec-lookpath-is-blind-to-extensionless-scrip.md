---
id: lesson-001-go-s-exec-lookpath-is-blind-to-extensionless-scrip
type: lesson
status: active
created: "2026-08-10"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 001: Go's exec.LookPath is blind to extensionless scripts on Windows

**Context**: `dotf doctor`'s `has()` (a `command -v` emulation) used `exec.LookPath` to detect optional tools. On a healthy Windows box it reported `bats` as missing though `~/.local/bin/bats` is on PATH and runs fine (BUG-052, #804).

**Problem**: On Windows `exec.LookPath` only resolves names carrying a `PATHEXT` extension (`.exe/.cmd/.bat/...`). An extensionless POSIX script laid down on PATH is invisible to it — a false-negative on a presence check, which is worse than no check (it trains the operator to ignore a non-zero doctor exit).

**Solution**: When `LookPath` misses *and* `GOOS == "windows"`, fall back to a manual PATH scan for a regular file whose name matches exactly. POSIX hosts resolve these through `LookPath` already, so the fallback is Windows-only — filesystem-scanning on POSIX would mask a genuinely-missing tool.

**Rule**: Any `command -v`/`LookPath` presence check that must see shell scripts (not just native `.exe`) on Windows needs an extensionless-PATH-scan fallback. Never rely on bare `exec.LookPath` for cross-platform tool detection when the tool ships as an extensionless script.
