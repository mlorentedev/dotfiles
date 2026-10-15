---
id: "lesson-439-a-cache-used-as-the-truth-side-needs-an-age"
type: lesson
status: active
title: "A cache used as the truth side of a check needs an age, not just a presence"
created: "2026-10-15"
owner: manu
tags: [doctor, cache, severity, drift]
---

# A cache used as the truth side of a check needs an age, not just a presence

**Context:** `checkModelLimits` compares the limits declared in `ai/pi/models.json` against opencode's models.dev cache (`~/.cache/opencode/models.json`), the catalog opencode itself routes on (#2434). An overstated limit FAILs, and a FAIL sets doctor's exit status, which gates such as PI-002 AC3 read.

**Problem:** On a Windows box where opencode had not run since June, the cache still held June's limits for `openrouter/deepseek/deepseek-chat` (128000 / 16000). The provider had since raised them to 163840 / 16384, the values the repo declared, so doctor reported two FAILs against a declaration that the live catalog confirmed that day. `readCatalog` already treated an absent or broken cache as "this machine cannot say" (SKIP or WARN, never FAIL), but it only asked whether the file existed and parsed. A cache that exists and parses is not the same as one that is current.

**Solution:** `readCatalog` takes the System, stats the cache, and SKIPs past `catalogMaxAge` (14 days, measured against `sys.Now()`), naming the age in days and `opencode models --refresh`. A table-driven test sets the cache's mtime with `os.Chtimes` against a frozen clock: 13 days still compares (2 FAIL), 15 and 126 days skip. The fresh row is what kills the "always skip" mutation. Refreshing the cache from `doctor --fix` is CLI-121 (#2540).

**Why:** A check is only as current as its truth side. When that side is a cache someone else refreshes as a side effect, its age belongs in the same severity rule as its presence: stale means the machine has no current answer, which is a SKIP, not a finding against the repo. Ask of every cached input to a check what refreshes it and when that last happened; a check that never asks will one day report the cache's history as the repo's defect.
