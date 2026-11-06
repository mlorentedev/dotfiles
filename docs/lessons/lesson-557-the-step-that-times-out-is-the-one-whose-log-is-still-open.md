---
id: "lesson-557-the-step-that-times-out-is-the-one-whose-log-is-still-open"
type: lesson
status: active
title: "The step that times out is the one whose log is still open"
created: "2026-11-05"
---

# The step that times out is the one whose log is still open

## Context

`dotf vm windows` runs each converge in the guest through a scheduled task. The task runs `guestrun.ps1`, which pipes
install.ps1's output into `<step>.log` with `Out-File`. When the step ends, the host reads that log through the
guest agent's `guest-file-open`.

## Problem

Two from-zero runs hit the 1h step limit and printed nothing from the guest. The host read the log only after the
wait, and it dropped the read error. While the step is still running, `Out-File` holds the file open, and the agent's
open cannot share it. So the code path that relays output failed only for the step that never ended, which is the
one step where the output is the only evidence. With `--keep`, `Get-Content` in the guest read the same file in the
middle of the run (94 KB). The guest had not hung: the link was slow, the tools step took 50 minutes, and the Claude
installer was still downloading.

## Solution

On a timeout, read the log with a shared read (`Get-Content` through the agent's exec), and list the processes still
running under the step's task, before the guest is discarded. In general, a "show what we have" path needs a test for
the failure it exists to explain: here the fake guest let a running step's log be opened, so the test passed while the
real guest never showed a byte. The step limit is now 2h, sized from the measured 1h23m.
