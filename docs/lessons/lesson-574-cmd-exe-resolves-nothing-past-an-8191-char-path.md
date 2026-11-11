---
id: "lesson-574-cmd-exe-resolves-nothing-past-an-8191-char-path"
type: lesson
status: active
title: "cmd.exe resolves nothing past an 8191-char PATH"
created: "2026-11-10"
---

# cmd.exe resolves nothing past an 8191-char PATH

## Context

#3174's test-windows doctor gate failed `node` and `python` with `'mise' is not recognized as an internal or
external command`. Yet the same step's own probe showed pwsh resolving `mise.exe` from `~\.local\bin`, the 78th of 199
PATH entries.

## Problem

Every mise shim on Windows is a `.cmd` that calls `mise`, so cmd.exe does the lookup, not pwsh. cmd.exe
resolves nothing from a PATH longer than 8191 characters, and the position of the entry does not matter.
This was measured on the Windows guest: a `.cmd` calling a `.cmd` that sits in the first PATH entry ran at 8180
characters and was "not recognized" at 8210. The gate had built its PATH as Machine + User + the process PATH, and
the process PATH already holds both of those, so most entries appeared twice: 8588 characters. pwsh resolves on its
own and is unaffected, so the two layers disagreed about the same binary.

## Solution

The gate joins the three without repeats (`Join-PathEntries`; first spelling wins, compared case- and
trailing-slash-insensitively). The same runner PATH became 120 entries and 5727 characters, and the gate went
from five FAILs to none. On a real box, doctor's "Core tools in PATH" section now FAILs a PATH past the limit and names its repeated
entries (#3285): one row instead of a "not recognized" per tool.

When a Windows command is "not recognized" but `Get-Command` finds it, check `$env:PATH.Length` before the
install. Rebuilding PATH from the registry has to deduplicate, never concatenate.
