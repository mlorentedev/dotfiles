---
id: "lesson-502-a-workflow-run-from-a-comment-triggered-run-has-lost-its-pull-request"
type: lesson
status: active
title: "A workflow_run from a comment-triggered run has lost its pull request"
created: "2026-10-21"
---

# A workflow_run from a comment-triggered run has lost its pull request

## Context

`review-attestation` re-evaluates a PR when the `pr-agent` reviewer finishes,
through `workflow_run`. It has to, because GitHub emits no workflow events for
comments created with `GITHUB_TOKEN`, and PR-Agent publishes with it: its
review comments wake nothing (measured on #1047, GUARD-003). The gate recovers
the PR from `workflow_run.head_sha` through `commits/{sha}/pulls`.

## Problem

A review asked for with `/review` runs `pr-agent` from `issue_comment`, and a
run from `issue_comment` belongs to the default branch. The `workflow_run` it
completes carries main's SHA and an empty `pull_requests[]`, so the lookup
finds no open PR and the gate skips. Measured 2026-10-20: slash run
39365425846 on an open PR, then attestation 39365734960, `no open pull request
for fd0be466`. Skipping is the right answer to "nothing to judge"; the defect
is that the identity was lost at the trigger, so no later step can recover it
(#1128).

The same SHA broke concurrency. The group was keyed on `head_sha`, so every
comment-triggered reviewer run, skipped ones included, shared one group, and
the comment that follows a `/review` cancelled its re-evaluation (39539977021).

## Solution

The run being judged hands its PR forward: `pr-agent`'s `issue_comment` path
uploads the issue number as the `reviewed-pr` artifact, before any step that
can end the job, and the gate reads it only when `workflow_run.event` is
`issue_comment`. That event always executes the default branch's workflow, so
the number comes from reviewed code. A `pull_request` run executes the PR's own
copy of the workflow, so its artifacts are never trusted there. The gate
accepts only a number that names an open PR; anything else is still nothing to
attest. Comment-triggered runs are grouped by run id rather than SHA.

## Takeaway

Inside a `workflow_run`, `head_sha` and `head_branch` describe the run that
triggered it, not the change that run was about. For a run triggered by a
comment, a schedule or a dispatch, they name the default branch. If the
consumer needs the PR, the producer must publish it, and the consumer must
trust that only where the producer's workflow file cannot be edited by the PR.
