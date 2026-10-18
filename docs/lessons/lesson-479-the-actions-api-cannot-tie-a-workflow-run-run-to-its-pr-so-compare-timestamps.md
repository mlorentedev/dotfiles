---
id: "lesson-479-the-actions-api-cannot-tie-a-workflow-run-run-to-its-pr-so-compare-timestamps"
type: lesson
status: active
title: "The Actions API cannot tie a workflow_run run to its PR, so compare timestamps"
created: "2026-10-18"
---

# The Actions API cannot tie a workflow_run run to its PR, so compare timestamps


**Context:** #2724 option A: `dotf pr land` should wait for a `review-attestation` verdict that is on its way instead of refusing on the older red one. The first design asked GitHub for runs of that workflow that are still queued or running on the PR's head.

**Problem:** There is no such list. A `workflow_run` run, the one that carries the verdict after pr-agent finishes, has the base branch's commit as `head_sha`, an empty `pull_requests` and `display_title: review-attestation`; an `issue_comment` run has the base commit too. During the gap that failed #2718 (pr-agent done, verdict not yet written) no run on the head was active at all, so a wait keyed on runs would have refused exactly where it was needed. Only the reviewer's own `pull_request` run is addressable by commit.

**Solution:** Two signals. A pr-agent run on the head still queued or running (addressable). Otherwise, the newest comment or review on the PR is newer than the newest `review-attestation` commit status: every comment, and every pr-agent completion, makes the workflow recompute and rewrite the status, so a status older than the last word on the PR has not caught up. Both are bounded by one budget (10 minutes, past the longest verdict measured, 457 s), and the refusal names what it waited for. A red verdict with neither signal refuses at once. The second signal assumes that every comment the workflow does not react to is followed by a pr-agent completion, which does rewrite the status: measured on the last four PRs, the only comments by the Actions token are pr-agent's, and triage comments are posted with the user's token. A comment by another Actions-token job would hold the wait for the full budget before refusing, never past it.

**Why:** When the system will not say what is in flight, compare what it has already written with what happened after it. A timestamp on the output is observable where the process that will write the next one is not.
