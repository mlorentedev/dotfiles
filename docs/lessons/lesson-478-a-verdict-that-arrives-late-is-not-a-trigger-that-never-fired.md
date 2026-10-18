---
id: "lesson-478-a-verdict-that-arrives-late-is-not-a-trigger-that-never-fired"
type: lesson
status: active
title: "A verdict that arrives late is not a trigger that never fired"
created: "2026-10-18"
---

# A verdict that arrives late is not a trigger that never fired


**Context:** #2724: `dotf pr land 2718` refused with `review-attestation: fail` although pr-agent had reviewed. The ticket's own diagnosis was a missing trigger (a comment made with the Actions token starts no workflow), and its fix list was a rerun in `dotf pr land` or a second trigger.

**Problem:** The `workflow_run` trigger had fired. Run 39064824391 was created two seconds after pr-agent completed, resolved PR 2718 and the right head, and published `success`. It started its job 4 min 41 s after it was created, so the status stayed at an earlier, correct `failure` for 4 min 55 s after the review landed. Measured over the same workflow: a `workflow_run` run waits a median 203 s in a busy window and 130 s in a quiet one. The manual rerun the ticket credits only repeated a status already published. The hypothesis survived for a day because the first look compared "status red" with "reviewer commented" and never put the run's own timestamps beside them.

**Solution:** Before rewiring a trigger, list the runs it produced and read three times for each: created, job started, status written. If the trigger ran and wrote the right verdict, the defect is in who reads the status and when, not in the wiring. Here the reader (`dotf pr land`) has to treat a status older than the newest reviewer comment as in flight.

**Why:** A check that is eventually right looks the same as one that is wrong when you read it once. The question that tells them apart is not "did it fire" but "when did it write, relative to the event that should have changed it".
