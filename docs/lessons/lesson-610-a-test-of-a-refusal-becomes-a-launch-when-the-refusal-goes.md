---
id: "lesson-610-a-test-of-a-refusal-becomes-a-launch-when-the-refusal-goes"
type: lesson
status: active
title: "A test of a refusal becomes a launch when the refusal goes"
created: "2026-11-15"
---

# A test of a refusal becomes a launch when the refusal goes

## Context

`dotf eval run` printed a plan and refused to launch anything until the launcher existed
(HARNESS-101). One test drove the real command, with the real dependencies, without `--plan`, and
asserted the refusal.

## Problem

That test was safe only because the command refused. The slice that wired the launch removed the
refusal. If the test had not been rewritten in the same commit, `go test ./...` would have checked
out a replay and started a paid model run on every machine and in CI. The test's name described
what it checked. The danger was in what the dependencies it used could do once the guard was gone.

## Solution

Before removing a guard, list every test that reaches the guarded code through real dependencies
(`git grep` for the command's invocation), and move each one onto injected ones first. The eval
command's steps after the plan (corpus check, preflight, run directory, launch) are each a field of
`evalRunDeps`, so its tests use a spy that records the order and launches nothing. The real
`execute(...)` tests keep only paths that stop before any of them: `--plan` and a matrix above the
cap.
