---
id: "lesson-581-a-test-that-reads-github-step-summary-on-a-runner-reads-the-job-s"
type: lesson
status: active
title: "A test that reads GITHUB_STEP_SUMMARY on a runner reads the job's"
created: "2026-11-11"
---

# A test that reads GITHUB_STEP_SUMMARY on a runner reads the job's

## Context

The spec gate's parity helper (CI-018) runs `scripts/check-spec-gate.sh` and `dotf gate spec` on the same fixture
and compares what they print. The adjacency report also appends a table to the step summary, so the helper was
taught to compare that too: when `GITHUB_STEP_SUMMARY` was set, it checked that the file held dotf's table once
per shell.

## Problem

Locally the variable is set only by the one test that asserts on a summary, and all 166 cases passed. On the
runner every step has `GITHUB_STEP_SUMMARY` pointing at the job's real summary file. Every call of the helper,
in every suite, then compared that shared file, which already held the tables of earlier tests: five archive-suite
cases went red with three tables where two were expected. The script had also been appending its tables to the
job's real summary all along, unnoticed, because nothing read it.

## Solution

The helper gives each side a summary file of the call's own (`GITHUB_STEP_SUMMARY="$sh_sum" run_both_shells …`,
`--summary "$dotf_sum"`), compares those, and appends the script's table to the caller's file only when the caller
named one. The comparison now runs on every call, so it no longer depends on which environment the suite runs in.

## Takeaway

`GITHUB_STEP_SUMMARY`, `GITHUB_OUTPUT`, `GITHUB_ENV` and `GITHUB_ACTIONS` are set on a runner and unset on a
laptop. A test that branches on one, or lets the code under test write to one, behaves differently in CI. Set them
explicitly per test, to a private path or to empty, and never let a test touch the job's own files.
