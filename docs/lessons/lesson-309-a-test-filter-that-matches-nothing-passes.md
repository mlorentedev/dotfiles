---
id: lesson-309
type: lesson
status: active
created: "2026-09-27"
owner: manu
tags: [lesson, testing, bats, spec, vacuous-check]
---

# 309 — A test filter that matches nothing passes

## What happened

CI-004's `features.json` needed one verification command per acceptance criterion, and several criteria are "this named test passes". The obvious command is `bats -f '<test name>' <file>`. While writing the spec, each command was run before its test existed, to confirm it failed. None did.

`bats -f` with a filter that selects no test prints `1..0` and exits 0 (measured with bats 1.13.0). An empty selection is a successful run of zero tests. Every feature check written that way would have been green on the day the spec was written, before any code, and would have stayed green if the test were later renamed or deleted.

## The rule

A check that selects its subject by name has to assert that the subject was selected, not only that the selection passed. For bats, run the file and require the TAP line `ok N <name>`, so an absent test is a failure rather than an empty success. The same shape bites `go test -run`, `pytest -k` and any grep over output: zero matches is a result, and most runners report it as a pass.

Run every new verification command once before the thing it verifies exists. A check that passes then is not checking anything.

Refs: CI-004 (#1739), `specs/CI-004-testing-surface-optimization/features.json`.
