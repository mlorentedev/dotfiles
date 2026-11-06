---
id: "lesson-558-check-a-tool-where-the-code-runs-it-not-on-the-path"
type: lesson
status: active
title: "Check a tool where the code runs it, not on the PATH"
created: "2026-11-05"
---

# Check a tool where the code runs it, not on the PATH

## Context

doctor's core-tools check expected `bash` on the PATH on every OS. On a from-zero Windows guest it reported
`[FAIL] bash not in PATH` after a clean converge, and no setup step could clear it (#3214).

## Problem

Git for Windows adds only `Git\cmd` to the PATH. The repo decided not to put Git Bash on the PATH in general
(BUG-122, #2346). dotf never runs a bare `bash` there anyway: `tools.HomeRunners` finds Git Bash from
`git --exec-path`. So the check asked a question the code does not depend on. On a real box a bare `bash` on the PATH
can be WSL's `System32\bash.exe`, and the check would have passed on the wrong shell. CI never saw any of this,
because the hosted runner image ships bash on the PATH (lesson 552).

## Solution

On Windows, doctor resolves bash the way dotf runs it: one shared `tools.GitBash` turns the `git --exec-path` output
into `<root>\bin\bash.exe`. `HomeRunners` and doctor both call it, so the check and the code cannot drift apart.
In general, a health check should look for a dependency through the same resolution the code uses. A PATH lookup
answers a different question, and that answer depends on the machine's history.
