---
id: "lesson-566-pid-liveness-is-not-process-ownership"
type: lesson
status: active
title: "PID liveness is not process ownership"
created: "2026-11-07"
---

# PID liveness is not process ownership

## Context

The first read-only bridge status implementation treated a recorded PID as
owned whenever that PID was alive. Process mutation was deferred, so this
looked harmless while `stop` did not exist.

## Problem

Operating systems reuse PIDs. A stale record can therefore point at an
unrelated live process, making status report a foreign process as the managed
bridge and turning the `unowned` state into dead code. Adding `stop` later
would make the same shortcut dangerous.

## Solution

Establish ownership before any health classification or signalling. Compare
the PID, executable and a platform-native process-start identity; treat a
mismatch as unowned and an inspection failure as unavailable. Use the same
identity seam for read-only status and later lifecycle mutations so safety
does not depend on which command is running.
