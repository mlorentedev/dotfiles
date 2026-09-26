---
id: lesson-305
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, systemd, shell, measurement]
---

# 305 — `systemd-run` expands `${VAR}` in its own arguments before the shell sees them

## What happened

Tests ran under a memory cap with `systemd-run --user --scope … bash -c '…; echo "test-rc=${PIPESTATUS[0]}"'`. systemd substitutes `$VAR` and `${VAR}` in the command line it is given, using its own environment. It printed `Invalid environment variable name evaluates to an empty string: PIPESTATUS[0]`, and the line read `test-rc=`, with the exit status gone. The run looked finished and green, and the one number it existed to report was empty.

## The rule

Do not pass shell code that contains `$` to `systemd-run` inline. Put the command in a script file and run the script, or escape each `$` as `$$`. Treat an empty value in a measurement's output as a failed measurement, never as a result.

Refs: #1626 (W1.4).
