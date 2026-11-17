---
id: "lesson-607-a-pipe-reader-that-stops-early-outlives-every-deadline"
type: lesson
status: active
title: "A pipe reader that stops early outlives every deadline"
created: "2026-11-15"
---

# A pipe reader that stops early outlives every deadline

## Context

The eval launcher hands a runner's stdout to `spec.SinkTranscript` through an `io.Pipe`, so the
transcript the stall detector watches is written as the runner speaks (HARNESS-101). The runner is
started by `spec.RunWithLimits` under a deadline.

## Problem

When the sink could not create the transcript it returned at once, and nothing read the pipe again.
`exec.Cmd` copies a non-file stdout from an OS pipe in its own goroutine, and that copier blocked on
`pw.Write`. The deadline killed the runner, but `cmd.Wait` waits for the copier too, so the attempt
never returned: the test sat past a 3-second deadline until its own 30-second guard. A deadline
bounds the process, not the goroutines that move its output.

## Solution

Whoever reads a pipe closes it when they stop: the sink's goroutine ends with
`pr.CloseWithError(err)`, which turns the copier's next write into an error, so the runner gets
`EPIPE` and the run ends. A test drives the real `RunWithLimits` with a shell runner that floods
stdout while the transcript path is a directory. A fake runner cannot show this, because it never
goes through the copier.
