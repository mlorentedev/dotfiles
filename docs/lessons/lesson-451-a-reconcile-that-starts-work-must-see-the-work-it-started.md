---
id: "lesson-451-a-reconcile-that-starts-work-must-see-the-work-it-started"
type: lesson
status: active
title: "A reconcile that starts work must see the work it started, or changed=0 never arrives"
created: "2026-10-16"
owner: manu
tags: [lane, reconcile, idempotence, testing]
---

# A reconcile that starts work must see the work it started, or changed=0 never arrives

**Context:** `dotf lane plan` (CLI-127, #2550) classes each action `auto`, `pending` or `ambiguous`; `lane apply` will run only the `auto` ones and must report `changed=0` on a second run (ADR-050 decision 4). Decision D1 makes "start the next queued item" an `auto` action.

**Problem:** Read literally, that action is not idempotent. Item 1 starts on the first apply; on the second apply item 1 has a worktree, so item 2 is now the first open one, and it starts too. A queue of N items takes N runs to settle, and a reconcile that never reaches `changed=0` is a loop, not a reconciler. Separately, a test that proves "plan changes nothing" over a plan that is empty proves nothing, and a fixture with an untracked lease file read as dirty, which silently turned the one `auto` reap it was built for into an `ambiguous` flag.

**Solution:** A start is planned only while no lane is in flight (no row in Running, Review or Landing), and only for the first pending item in queue order. The lane the first apply starts is Running, so the second apply plans nothing. `TestPlan/a start waits while any lane is in flight...` pins it. `TestPlanMutatesNothing` runs the whole read path against a real git fixture, asserts the plan contains a `reap/auto` and a `flag/ambiguous`, and only then compares a hash of the refs, worktree registry, statuses and lease files before and after. The fixture excludes `.dotf-worktree.json` the way dotf does.

**Why:** A reconcile converges only if each of its actions removes the condition that proposed it. An action that adds work (starting a lane) has to read the work it already added as a reason to stop. And a no-mutation test needs a plan with something in it: assert the plan is non-empty before comparing hashes, or an empty plan passes for the wrong reason.
