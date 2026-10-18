---
id: "lesson-477-an-event-payload-is-a-snapshot-a-gate-that-skips-on-it-can-delete-the-only-run-left"
type: lesson
status: active
title: "An event payload is a snapshot: a gate that skips on it can delete the only run left"
created: "2026-10-18"
---

# An event payload is a snapshot: a gate that skips on it can delete the only run left


**Context:** #2591: a PR opened as a draft and marked ready two seconds later (the order AGENTS.md prescribes) was never reviewed by pr-agent. `pr-agent.yml` runs once per `pull_request` event, and both events share one concurrency group with `cancel-in-progress`.

**Problem:** The `ready_for_review` run (payload `draft: false`) was cancelled by the `opened` run (payload `draft: true`), which then skipped its review on its own payload. GitHub does not order the two runs. No run was left to review, and `review-attestation` stayed red until someone typed `/review`. The workflow already had a step that re-reads the PR from the API (#1923, for a PR merged or drafted while queued), but the job `if` skipped the run before any step could run.

**Solution:** The job gate no longer reads `github.event.pull_request.draft`; the existing live read decides, whichever run survives. A bats test parses the workflow and fails if the job `if` mentions the payload's draft flag or if the step stops reading `draft` from `pulls/<n>`.

**Why:** When runs can cancel each other, a skip decided from a payload is a decision about a past moment, made by the one run that cannot be replaced. A cancel makes a stale snapshot permanent. Decide from live state, in the step, and leave the job `if` to what cannot change between the event and the run (actor, event name, fork).
