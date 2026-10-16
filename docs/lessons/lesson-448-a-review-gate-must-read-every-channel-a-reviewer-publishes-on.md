---
id: "lesson-448-a-review-gate-must-read-every-channel-a-reviewer-publishes-on"
type: lesson
status: active
title: "A review gate must read every channel a reviewer publishes on, not the one it was first seen on"
created: "2026-10-16"
owner: manu
tags: [pr, review, triage, merge-grant]
---

# A review gate must read every channel a reviewer publishes on, not the one it was first seen on

**Context:** `dotf pr triage-queue` and `dotf pr land` decide whether reviewer output is awaiting a disposition. That is the fourth condition of the standing merge grant. Both read one endpoint per PR, `GET /issues/{n}/comments`, and count a declared reviewer's comment only when it carries one of that reviewer's markers.

**Problem:** GitHub has two places a reviewer can publish. PR-Agent posts an issue comment with a heading, so the gate saw it. CodeRabbit posts a clean review as an issue comment but files a review **with findings** through the reviews API, with inline comments and no marker. The gate never saw those reviews. It answered "clear" on #1683 (2026-09-25, filed as #1686) and again on #2581 (2026-10-16), where CodeRabbit filed three valid findings 82 seconds before `dotf pr land` merged the PR. The ticket had the root cause for three weeks. The gate kept passing because a clean queue and a blind one return the same exit code.

**Solution:** The gate also reads `GET /pulls/{n}/reviews`. Any submitted review by a declared reviewer counts as output by its type, with no marker, timed by `submitted_at`. Dismissed and pending reviews are ignored. A full page is refused the same way the comments axis refuses one. Inline comments need no separate read, because each belongs to a review (measured on #2581: all five inline comments carried a `pull_request_review_id`). The test fixture uses #2581's own timestamps.

**Why:** A reviewer's output channel is decided by the reviewer, and it can change with the verdict. CodeRabbit uses a different channel for "clean" than for "findings", so a gate keyed to the channel it first saw will see the clean result and miss the findings. When a gate counts the output of tools it does not control, enumerate the channels the platform offers and read all of them. When a gate gets "nothing pending" wrong, it says so with exit 0. Treat a known blind spot in a merge gate as a defect in the merge path, not as backlog.
