---
id: "lesson-332-a-guard-kept-in-a-spec-folder-runs-nowhere"
type: lesson
status: active
title: "A guard kept in a spec folder runs nowhere"
created: "2026-10-02"
owner: manu
tags: [lesson, guards, specs, ci, harness]
---

# A guard kept in a spec folder runs nowhere

## What happened

HARNESS-046 wrote `check-roster-consistency.py` to catch drift between the vault's `ROSTER.md` and the persona definitions. It lived in `specs/HARNESS-046/`, and its `features.json` entry ran it from there. No bats file, workflow or hook called it. Five weeks later, its independent review ran it and found it red on `HEAD`: curator's skills had been reordered in one file, and the guard compared lists, so it called that drift. Nobody had seen the failure, because nothing had run the guard since the day it was written. Archiving the spec would also have moved the script and broken the `features.json` path.

## Rule

A check written to catch the *next* occurrence is only a guard once something runs it without being asked. Put it where the suites live (`scripts/` plus a bats file), not beside the spec that motivated it. When the check reads state that CI does not have, such as the vault, split it the way the stub/real pairing rule already asks: a fixture suite pins the logic everywhere, and a `-real` sibling runs against the live state wherever it resolves, skipping by name elsewhere.

## Guard

`tests/roster-consistency.bats` (fixtures, runs in CI) and `tests/roster-consistency-real.bats` (real `dotf` parser and live vault). `tests/stub-real-pairing.bats` refused the first version, which kept the live case inside the stubbed suite.
