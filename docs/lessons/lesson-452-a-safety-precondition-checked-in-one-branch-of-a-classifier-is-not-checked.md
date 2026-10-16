---
id: "lesson-452-a-safety-precondition-checked-in-one-branch-of-a-classifier-is-not-checked"
type: lesson
status: active
title: "A safety precondition checked in one branch of a classifier is not checked"
created: "2026-10-16"
owner: manu
tags: [lane, reconcile, testing, safety]
---

# A safety precondition checked in one branch of a classifier is not checked

**Context:** `dotf lane plan` (CLI-127, #2550) classes a merged worktree `auto`, `pending` or `ambiguous`. AC4 says a worktree with a live session is never `auto`, and the spec's live check expects it `ambiguous`.

**Problem:** `planLanded` tested liveness only inside the REAPABLE case. A merged worktree whose lease had not yet expired took the time-only `pending` branch first, which never looked at liveness, so the plan told the reader "wait and it goes" for a worktree with a live session. Every unit test passed: they were written from the same table, and the table's rows put liveness only on the REAPABLE row. Only the read-only run against the machine's 20 real lanes found two live worktrees classed `reap/pending`.

**Solution:** Liveness is the first case of the switch, ahead of every time-only reason, and the table has a row for a live session under a running lease. The AC4 check is stated over the whole plan (every live lane is ambiguous), not per row.

**Why:** A classifier's safety precondition belongs where every path passes through it, not in the branch where it first mattered. A property that must hold for every row is a loop over the real output, not a case in a table built from the author's own model of the rows. `auto` was never reachable for a live worktree, so the fix changed no reaping; it changed what the plan promised.
