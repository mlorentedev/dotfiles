---
id: "lesson-508-a-mutation-that-breaks-the-build-reports-no-failing-test"
type: lesson
status: active
title: "A mutation that breaks the build reports no failing test"
created: "2026-10-21"
---

# A mutation that breaks the build reports no failing test

## Context

The mutation checks for MEMORY-008 slice 3 used a small shell loop. It applied a
`sed` edit, ran `go test`, and printed the names on the `--- FAIL` lines. An
empty list meant the mutant had survived.

## Problem

One mutation replaced the condition `agent == "" || w == "" || w == agent`
with `true`. That left `w` declared and unused, so the package no longer
compiled. `go test` printed `[build failed]` and no `--- FAIL` line, and the
loop reported no failing test. It read as a surviving mutant, a gap in the
tests, when the tests had never run. This is the same class as the
silent-failure rows in `.claude/CLAUDE.md`: a probe whose failure path prints
"found nothing".

## Solution

Treat a build failure as its own outcome, never as an empty result. Grep for
`build failed` alongside `--- FAIL`, and write each mutation so that it still
compiles. Here that meant replacing the comparison (`w == agent` →
`w != ""`), not the whole condition. Re-run any mutant reported as surviving
before believing it.
