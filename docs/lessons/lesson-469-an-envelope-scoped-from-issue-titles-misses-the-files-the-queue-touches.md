---
id: "lesson-469-an-envelope-scoped-from-issue-titles-misses-the-files-the-queue-touches"
type: lesson
status: active
title: "An envelope scoped from issue titles misses the files the queue touches"
created: "2026-10-17"
---

# An envelope scoped from issue titles misses the files the queue touches


**Context:** The overnight autopilot run night-20261017T0750Z armed its envelope (`.agent/unattended.yml`) for five tickets. `scope.write_paths` was written from the issue bodies: `harness/bind*.json` for #2543, the deploy and env packages for #2402.

**Problem:** Two of the five fixes needed files outside those globs. #2543's core change is `harness/manifest.json`, which no `harness/bind*.json` glob matches. #2402's guard required updating literal-spelling asserts in `tests/copilot-config.bats` and `tests/opencode.bats`. Both show up only once the change is made and the suite runs. The envelope cannot be widened mid-run without voiding the grant, so both PRs (#2671, #2673) were finished but left for the owner to land. The run had done the work and could not merge it.

**Solution:** At arming, derive each item's write set from the code rather than the ticket. Do this for every queued item: `git grep -l` the symbols and literals the fix changes, including tests that assert the old value (for a manifest key, `git grep -l '<old literal>' -- tests cli`). Put the files that turn up in `write_paths`. A file that should stay frozen marks its item owner-lands from the start, which is a decision made at arming, not a discovery at 3 a.m.

**Why:** A path glob is a promise about where a change will land, made before the change exists. The ticket names the defect's location, not every file that pins the current behaviour, and the pins are usually literal asserts in tests. The cheap moment to find them is before the envelope is committed.
