---
id: "lesson-418-capturing-stdout-turns-every-terminal-check-into-a-pipe-check"
type: lesson
status: active
title: "Capturing a process's stdout turns every terminal check inside it into a pipe check"
created: "2026-10-14"
---

# Capturing a process's stdout turns every terminal check inside it into a pipe check

## Context
`dotf converge` now keeps a transcript of each applied run (#1098). It has to include what every child prints (uv, npm, winget, brew, the setup script), on every OS, without plumbing a writer into each reconciler. To do that, the run swaps `os.Stdout` and `os.Stderr` for pipes and tees them to the console and the log. Go code that prints to `os.Stdout`, and every child that inherits it, then lands in both.

## Problem
After the swap, `os.Stdout` is a pipe. Any `term.IsTerminal(int(os.Stdout.Fd()))` evaluated during the run answers "no terminal", even when a person is sitting at one. converge has one such check on its own path: `identityTerminal`, which `workstation-target` calls through `sshtarget.Sudo` to decide whether it may prompt for the sudo password. Under the capture it would have decided there is no terminal and failed fast, so the run would have reported `ErrNoTerminal` on a machine where the person could have typed the password.

Two other values bind the old file early and silently skip the capture. `convergeOptions()` stores `os.Stderr` in `tools.Installer.Out` when it is called. `cmd.OutOrStdout()` returns the real stdout file bound in `RunE`.

## Solution
- The run keeps the original stdout, and `consoleOut()` returns it while a capture is active. Terminal checks read `consoleOut()`, not `os.Stdout`, and a test asserts what a reconciler sees during a captured run.
- The capture starts before `convergeOptions()` is called, so writers bound at construction get the pipe. The report is written to `io.MultiWriter(w, log)` explicitly.
- The capture is undone before the identity epilogue, which walks the restore on a terminal.
- Draining is bounded (2 s). A daemon the run started can inherit the pipe's write end and keep it open, and the run must still exit.
- Children see a pipe, so when the console is a terminal the run sets `FORCE_COLOR=1` and `CLICOLOR_FORCE=1`, unless the person already set them, so tools that honour them stay coloured. The log strips the escapes.

## Why
Swapping the process's stdio is the only capture that reaches children on every OS without per-call plumbing, and it changes what "stdout" means for the whole process. Every question asked of `os.Stdout` during the swap has to be audited (`git grep 'IsTerminal\|Stdout.Fd()'`), along with every value that copied `os.Stdout` or `os.Stderr` before it.
