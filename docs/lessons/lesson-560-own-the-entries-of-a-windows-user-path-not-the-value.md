---
id: "lesson-560-own-the-entries-of-a-windows-user-path-not-the-value"
type: lesson
status: active
title: "Own the entries of a Windows user Path, not the value"
created: "2026-11-05"
---

# Own the entries of a Windows user Path, not the value

## Context

PLAT-001c moves the Windows CLIs from winget to mise. winget put each tool's directory on the user Path.
mise puts its shims in `%LOCALAPPDATA%\mise\shims`, and nothing registered that directory or `~/.local/bin`.

## Problem

On a clean Arm64 guest, a process started without a profile resolved neither `dotf` nor `mise`. That covers
a Scheduled Task, `pwsh -NoProfile` and the CI doctor gate. Even the PowerShell profile could not help, because
it activates mise only when it finds `mise` on PATH. `install.ps1` places `dotf.exe` in `~/.local/bin` and
registers nothing. On a box where Claude's installer had run, that installer added `~/.local/bin`, which hid
the gap.

The obvious fix, writing the user `Path` through the existing `HKCU\Environment` store, would have caused two
new problems:

- The store writes REG_SZ. The user Path is REG_EXPAND_SZ, so installers' `%USERPROFILE%\...` entries would
  become literals that no process expands.
- `DOTF_MANAGED_ENV` owns whole variables, so a sweep based on it would own the user's whole Path.

## Solution

- Treat Path as the user's value and record only the entries dotf adds (`DOTF_MANAGED_PATH`).
- Put those entries at the head, and keep every other entry in place and spelled as stored.
- Remove a retired entry only if the record names it.
- Compare entries the way Windows compares directories: case-insensitive, ignoring a trailing separator,
  after `%VAR%` expansion. Treat empty segments as no change, so a second run writes nothing.
- Write with `SetExpandStringValue`.
- Have the probe read the registry, because the running process never sees its own registry write.

## Key Takeaway

A shared environment value such as PATH, written by many installers, needs ownership of entries, not of the
value. Measure from a clean box: a box where another installer patched the value hides the gap.
