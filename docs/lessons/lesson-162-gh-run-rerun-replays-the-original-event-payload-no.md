---
id: lesson-162-gh-run-rerun-replays-the-original-event-payload-no
type: lesson
status: active
created: "2026-08-07"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 162: `gh run rerun` replays the original *event payload*, not just the original workflow file

**Context**: A PR failed a label-gated check. The label was added, and the natural next step was to re-run the failed job rather than push a no-op commit.

**Problem**: The re-run failed identically. `gh run rerun` (and the UI button) replays the run against the **event payload frozen when the run was first triggered** — so the job still saw the label set as it was *before* the label was added. Nothing in the output distinguishes this from "the label did not fix it", which is the trap: the evidence looks like a refuted hypothesis rather than a stale input. The same freezing applies to `github.sha` on a `pull_request` event, so a re-run after a dependency PR merges still checks out the old merge commit and still sees the old base.

**Solution**: Trigger a fresh event on the current state — push a commit (even an empty one), or close/reopen the PR — rather than re-running history. Only a new event rebuilds the payload.

**Rule**: This is the second face of the same coin as the 2026-06-07 lesson above, which covers the *workflow definition* being replayed from the original commit. Both reduce to: a re-run is a replay, not a re-evaluation — the workflow file, the labels, the merge sha, and everything else in the event context are all pinned to the moment the run was created. Whenever a re-run is meant to test a change made *after* the run started, it cannot; use a fresh event and say so, or the green/red you get back is answering a question you did not ask.
