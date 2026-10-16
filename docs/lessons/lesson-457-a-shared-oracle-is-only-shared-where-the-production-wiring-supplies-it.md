---
id: "lesson-457-a-shared-oracle-is-only-shared-where-the-production-wiring-supplies-it"
type: lesson
status: active
title: "A shared oracle is only shared where the production wiring supplies it"
created: "2026-10-16"
owner: manu
tags: [lane, reconcile, testing, safety]
---

# A shared oracle is only shared where the production wiring supplies it

**Context:** Lesson 454 made `lane apply` ask the forge, like `plan`, by giving the sweep a `lane.ForgeSweep` runner. The second independent review of CLI-127 (a different model, #2550) returned FAIL on what that fix left unguarded.

**Problem:** `TestApplyNeverReapsWhatPlanCalledAmbiguous` builds its own `ForgeSweep` inside the test, so it pins the fixture, not the construction in `realLaneVerbs`, the only production place the runner and the session stores are supplied. Deleting `Forge: src.Forge` there left the suite green (at runtime a nil forge panics); deleting `Sessions: sessionStores()` left it green too, and then sweep reaped a merged worktree whose pi session could still be resumed, because a resumed-but-idle session has no process for the process gate to see. Separately, the command `plan` printed for a worktree with a live session was `done --force`, which `done` refuses for a recent session whatever `--force` says: the row promised a one-command answer that exited non-zero. No test ran a printed command.

**Solution:** `TestRealLaneVerbsSweepHonoursTheForgeAndTheSessions` calls `realLaneVerbs(...).Sweep()` over a real repository with a real pi session on disk and asserts the live one survives and the other goes; dropping either argument now fails it. `TestPlanResolverForALiveWorktreeRunsThroughDone` takes the printed string and executes it through the real `worktree done`. The live-session row now carries `--ignore-recent-session` and says it strands the session; an adopt with nothing to run is a flag; a dirty row names the changes the command discards.

**Why:** Two halves agreeing in a fixture proves the fixture. The oracle is shared only if the wiring that hands it over is on the tested path, and the arguments easiest to drop are the ones whose absence is silent. A command a tool prints for a human is an interface: the test that matters runs the string, because the quoting test and the "apply will not reap" test each passed while the command did not resolve the row.
