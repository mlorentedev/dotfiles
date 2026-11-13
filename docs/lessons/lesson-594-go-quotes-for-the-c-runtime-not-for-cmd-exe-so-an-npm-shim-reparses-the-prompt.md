---
id: "lesson-594-go-quotes-for-the-c-runtime-not-for-cmd-exe-so-an-npm-shim-reparses-the-prompt"
type: lesson
status: active
title: "Go quotes for the C runtime, not for cmd.exe, so an npm shim re-parses the prompt"
created: "2026-11-12"
---

# Go quotes for the C runtime, not for cmd.exe, so an npm shim re-parses the prompt

## Context

`lane.Exec` starts the successor agent on Windows with `exec.LookPath` and `exec.Command`, the prompt
as one free-text argument. The agents in the launch table were all native `.exe` files (claude, agy,
and dotf for pi), so nothing had run a batch file. An npm-installed agent is a `.cmd` shim (#3429).

## Problem

`os/exec` on Windows joins the arguments into one command line quoted for `CommandLineToArgvW`. Its own
documentation (Go 1.26.10, `exec.Command`) names `cmd.exe`, and so every batch file, as an exception:
cmd.exe unquotes differently, and nothing in `os/exec` or `syscall` escapes for it (no batch-file
handling anywhere in those packages). CreateProcess runs a `.cmd` or `.bat` target through cmd.exe, so
`% ^ & | < > "` in the prompt are re-parsed, and text after an `&` runs as a command (BatBadBut). It
fails silently for the benign case too: a prompt with `%PATH%` is expanded before the agent sees it.
A list of the characters to look for does not close it, because cmd.exe also reads newlines, `!` and
`,;=`.

The same shim also sits behind a native wrapper. pi's argv starts with `dotf secrets run`, a native
exe, so `lane.Exec` never sees `pi.cmd`; the second `exec.Command("pi", prompt)` inside `secrets run`
does.

## Solution

Decide on the resolved target, never on the arguments: refuse a `.cmd` or `.bat` outright, naming the
agent and the remedy (`refuseBatchShim`, no build tag so macOS tests it). Windows drops trailing dots and
spaces from a file name, so trim them before comparing the extension. Test the effect on the Windows
leg: a shim that writes a marker file when it runs, a prompt that tries to write another, and neither
file may exist. Any other code that starts a child from user-supplied argv on Windows has the same
exposure and needs the same check on its own resolved path.

## Related

- #3429, #3440 (the `secrets run` path), `cli/internal/lane/exec_shim.go`, `cli/internal/lane/exec_windows_test.go`
