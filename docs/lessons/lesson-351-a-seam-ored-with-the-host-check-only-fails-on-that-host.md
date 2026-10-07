---
id: "lesson-351-a-seam-ored-with-the-host-check-only-fails-on-that-host"
type: lesson
status: active
title: "A seam OR'd with the host check only fails on that host"
created: "2026-10-07"
---

# A seam OR'd with the host check only fails on that host

## Context
`dotf doctor` routes the target OS through `System.GOOS` so a test on one host can run every
OS branch. #2061 found `dirsProviding` reading `runtime.GOOS == "windows" || s.GOOS == "windows"`
and `checkPathFiles` reading `runtime.GOOS` alone.

## The Trap
The obvious test sets `GOOS = "windows"` and asserts the Windows behaviour. Mutating the fix
back showed that test passing with and without it: the `|| s.GOOS` half already honoured the
seam, so the Windows target worked everywhere. The bug is the other direction. On a Windows
host the host check overrides a POSIX target, and a macOS or Linux run cannot see that, because
there `runtime.GOOS` is never `"windows"`.

## The Solution
For a seam bug, test the direction in which the host value would win: a POSIX target asserting
POSIX behaviour, so the `windows-latest` leg fails if the host check returns. Back it with a
rule that holds on every host. Here, forbidigo bans `runtime.GOOS` in `internal/doctor`, and the
sites that ask about the host filesystem (case folding, the exec bit) carry a `nolint` that
says so.

To mutate on a host that cannot reproduce the original condition, model it there: in
`runtime.GOOS == "darwin" || s.GOOS == "windows"`, the host override fires on macOS.
