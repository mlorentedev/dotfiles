---
id: "lesson-524-killing-limactl-kills-neither-the-guest-command-nor-its-ssh"
type: lesson
status: active
title: "Killing limactl kills neither the guest command nor its ssh"
created: "2026-10-30"
---

# Killing limactl kills neither the guest command nor its ssh

## Context

`dotf vm linux exec --timeout` (PLAT-005) needs a deadline on a command running in a Lima guest. The obvious shape is
`exec.CommandContext` around `limactl shell`.

## Problem

Measured on Lima 2.2.1 by killing `limactl shell` mid-command:

- The guest command ran to the end. In a session held until `down`, a timed-out command keeps running, and a caller
  that retries runs it twice.
- The ssh child became an orphan of PID 1 and kept stdout and stderr open. Go's `Wait` on a non-`*os.File` writer
  blocks until those pipes close, so the deadline would not have ended the call either.

A unit test with a fake `Limactl` shows neither: the fake returns when its context ends.

Also measured: `limactl shell` returns 255 for a command that dies by a signal, the same code ssh uses for a lost
connection, and a stopped instance exits 1 like a guest command would.

## Solution

Three parts, one for each layer. The guest runs the command under `timeout -k 5 <limit>`, which is what stops it. The
host deadline starts first, so it fires first and decides the status, and a command that exits 124 by itself is not
read as a timeout. `cmd.WaitDelay` bounds the wait on the orphan's pipes. Liveness is checked with `limactl list`
before the shell, so a stopped guest is `stale`, not exit 1. The 255 stays ambiguous and the verb's help says so.

`limactl copy` has the same shape of problem. Its default rsync sends a guest path through a remote shell and copies a
directory with no `-r` to nowhere with exit 0. `--backend=scp` makes the path literal and the directory a failure.

## Rule

A deadline around a remote client bounds the client, not the remote work. Measure what killing the client leaves on
both ends, and give the remote side its own limit. Any `exec.CommandContext` with a writer that is not an `*os.File`
needs `WaitDelay`, or a surviving grandchild outlives the deadline. Run a wrapped tool on the awkward input too: a space in a path, a directory, a value it may expand.
