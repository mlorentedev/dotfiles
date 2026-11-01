---
id: "lesson-526-a-required-check-a-workflow-can-cancel-belongs-in-a-commit-status"
type: lesson
status: active
title: "A required check that a workflow can cancel belongs in a commit status"
created: "2026-10-30"
---

# A required check that a workflow can cancel belongs in a commit status

## Context

`spec-gate.yml` runs on six `pull_request` event types under a concurrency group. One push to a PR can fire two of
them in the same second, for example `synchronize` with an `edited` from the board automation (#2866).

## Problem

A concurrency group holds one pending run, and a newer pending run cancels the older one, whatever
`cancel-in-progress` says. The cancelled run stays on the head commit as a check run named `spec-gate`, next to the
green one. Branch protection counted it and reported the PR BLOCKED with every check green; `dotf pr land` refused,
and only a manual rerun of the cancelled run cleared it (#2849, 2026-10-21).

## Solution

The job is no longer named after the required context. Its last step posts a commit status under the fixed
context `spec-gate`, with `if: success() || failure()`: a cancelled run reaches neither, so it posts nothing, and
the context only ever holds a finished verdict. `review-attestation.yml` already used this shape for a different
reason. `tests/spec-gate-status.bats` holds the condition, the job name and the `statuses: write` permission.

## Why

`always()` is the obvious condition for "publish the verdict whatever happens", and it is the wrong one: it also
runs on cancellation, and would post a failure for a gate that never finished. No condition at all is wrong the
other way: the step would be skipped on a red gate, and the context would stay pending instead of failing.
