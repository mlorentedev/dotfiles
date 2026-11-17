---
id: "lesson-610-a-tool-managers-path-for-a-command-is-not-a-file-windows-will-start"
type: lesson
status: active
title: "A tool manager's path for a command is not a file Windows will start"
created: "2026-11-16"
---

# A tool manager's path for a command is not a file Windows will start

## Context

On Windows `dotf` refuses to start an agent whose launcher is a `.cmd` or `.bat` file: `cmd.exe` re-parses the
prompt (#3442). After that refusal no pi-runner reviewer could be launched from a Windows machine (#3569). The
first `pi` on PATH was a batch shim, and mise held a native `pi.exe` one directory away. The fix asks
`mise which <name>` and starts that file instead of the shim.

## Problem

The first version checked mise's answer the way it checked PATH's: refuse a name ending in `.cmd` or `.bat`.
Run against the real machine before the commit, it accepted two answers it should not have:

- `mise which opencode` -> `...\installs\opencode\1.16.2\opencode`
- `mise which bats` -> `...\installs\npm-bats\1.13.0\bats`

Neither ends in `.cmd`. Both are the extensionless POSIX stub npm writes, with `opencode.cmd` and `bats.cmd`
beside them. Started by that path, Windows appends PATHEXT and runs the batch file: the launch the check exists
to prevent, arrived at through the path that was supposed to avoid it. A test with a fake answer named `pi.exe`
could not have shown this. Only the real tool manager's answers did.

## Solution

Accept what a native launch needs, not everything that is not known to be a shim: an absolute path to a regular
file whose name ends in `.exe`. The answer is treated as input, like PATH's: a bare command name only, `--`
before it, a bounded call, and a mise that is a native executable itself (`cli/internal/lane/exec_native.go`).

## Rule

When a check moves from one source of paths to another, re-derive it for the new source. "Not a batch file" was
a complete test for what `exec.LookPath` returns on Windows, because LookPath only returns names carrying a
PATHEXT extension. It is not one for a path another program prints. Ask the real program on the real machine
what it answers before trusting a deny-list written for a different caller.
