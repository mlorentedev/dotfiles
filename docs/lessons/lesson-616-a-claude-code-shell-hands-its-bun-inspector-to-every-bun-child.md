---
id: "lesson-616-a-claude-code-shell-hands-its-bun-inspector-to-every-bun-child"
type: lesson
status: active
title: "A Claude Code shell hands its Bun inspector to every Bun child"
created: "2026-11-23"
---

# A Claude Code shell hands its Bun inspector to every Bun child

## Context

Claude Code is a Bun program. On Windows it runs its own inspector on a localhost port and exports
`BUN_INSPECT` to the shells it starts. `dotf spec review` started the `pi` reviewer, also a Bun
program, with the launcher's whole environment.

## Problem

`pi` read the inherited `BUN_INSPECT`, tried to bind the port Claude Code already held, printed
`Failed to start inspector ... EADDRINUSE` and exited before reading anything. The launcher
recorded verdict ERROR. It happened on every launch from an agent shell and never from the
owner's own terminal, so it looked like a flaky reviewer rather than an environment leak. The port
was 9220 on one day and 9221 on another, so a check keyed on the port would not have held.

## Solution

Drop the inspector variables where the child is started, not in the caller's shell:
`bunenv.WithoutInspector` removes `BUN_INSPECT` and every `BUN_INSPECT_*`, by prefix and
case-insensitively, and `spec.RunWithLimits` applies it to every runner (#3615). The test re-runs
the test binary as the child so it holds on Windows, where the failure was measured, and it goes
through `RunWithLimits`, so deleting the `c.Env` line turns it red. Before starting any Bun-based
tool from code an agent runs, filter the environment the same way; the remaining launch paths are
#3622.

Filter `c.Environ()`, called after `c.Dir` is set, not `os.Environ()`. `os/exec` adds
`PWD=<Dir>` only when it builds the environment itself; an `Env` taken from `os.Environ()` hands
the child the launcher's `PWD`. The test that pins this reads the variable with `printenv`: the
first version ran `sh -c 'printf %s "$PWD"'` and passed against the bug, because a POSIX shell
replaces an inherited `PWD` that does not name its directory. A shell child hides exactly the
stale value the test exists to catch.
