---
id: lesson-302
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, bash, pipefail, sigpipe, flaky, guard]
---

# 302 — `producer | grep -q` under `pipefail` fails at random

## What happened

`scripts/check-lessons.sh` tested each lesson's membership in the index with `printf '%s\n' "$targets" | grep -qxF -- "$name"`, under `set -o pipefail`. On the real index it reported a correctly indexed lesson as "not in docs/lessons/_index.md" in 3 runs of 40, a different lesson each time. It blocked a pre-commit that added an unrelated lesson, and passed when run again.

`grep -q` exits at its first match. When `printf` still has output to write, it dies of SIGPIPE and returns 141. Under `pipefail`, the pipeline's status is that 141, not grep's 0, so a match reads as a miss. Whether `printf` has finished depends on scheduling and on how far into the input the match sits. That is why it looked random. Past the 64 KB pipe buffer, it fails every run.

## The rule

Under `pipefail`, never pipe into a reader that stops early (`grep -q`, `head`) when the result is decided by the pipeline's status. Feed it a here-string (`grep -q -- "$x" <<< "$data"`) or a file instead. A guard that fails at random gets re-run until it passes, and then it guards nothing.

Refs: HARNESS-161.
