---
id: "lesson-454-a-preview-and-the-verb-that-executes-it-must-ask-the-same-oracle"
type: lesson
status: active
title: "A preview and the verb that executes it must ask the same oracle"
created: "2026-10-16"
owner: manu
tags: [lane, reconcile, testing, safety]
---

# A preview and the verb that executes it must ask the same oracle

**Context:** `dotf lane plan` classes what a reconcile would do, and `dotf lane apply` runs the `auto` part through the existing `worktree sweep` (CLI-127, #2550). The first independent review of the spec (a different model, on its own fixture) returned FAIL.

**Problem:** `plan` decided "is this branch merged?" from the forge's REST endpoint, because lesson 450 showed ancestry reads a fresh branch as merged. `apply` called `worktree.Sweep`, whose runner still asked ancestry. So a clean, leased, expired worktree with no PR at the base commit was classed `ambiguous` (a human decision, with `worktree done --force` as its command) and then reaped by `apply` anyway. Lesson 450's own reasoning, that sweep re-checks every gate itself, is exactly what let apply out-reach the plan. The suite could not see it: `TestApply` drove a recording fake, `TestPlanMutatesNothing` proved only that `plan` writes nothing, and the two oracles never met in one test. Two smaller findings had the same shape: the printed resolving command quoted nothing (lesson 449's class, in the same diff), and a queued issue was matched to its lane by derived branch name only.

**Solution:** One oracle. `lane.ForgeSweep` is the sweep runner whose `IsPRMerged` is the forge's answer, the same function `ForgeGit` uses, and `apply` runs `worktree.SweepWithRunner` with it, sharing the cache `status` filled. The test that was missing now exists: `TestApplyNeverReapsWhatPlanCalledAmbiguous` builds a real repository with one worktree plan classes `auto` and one it classes `ambiguous`, runs the real sweep, and asserts the second is still on disk. Replacing `ForgeSweep.IsPRMerged` by the embedded ancestry answer makes it fail with `changed=2`.

**Why:** A verb that previews and a verb that acts are one decision made twice, and every input they read separately is a place for the two to disagree. Fixing an oracle in the previewing half (lesson 450) moves the problem to the acting half unless the acting half is changed in the same commit. The check that catches it runs both halves over one real fixture and compares what the preview promised with what the verb did; a fake in place of either half cannot.
