---
id: "HARNESS-111"
type: tasks
status: done
created: "2026-09-05"
---

# HARNESS-111 — tasks

- [x] Measure the payload in both units and establish the divergence (11974 chars / 12047 bytes / 12000 cap).
- [x] Identify the source of the gap (33 em-dashes, 4 section signs, an accented vowel, an ellipsis).
- [x] Measure the candidate folds and reject the ones that land back over the cap (`--` → 12009, ` - ` → 12042).
- [x] Fold typographic punctuation in `deploy_doctrine`, scoped to targets declaring a `char_cap`.
- [x] Write the substitutions as hex escapes after the literal form measured 3 SC1112 findings.
- [x] Report both units in the deploy's cap warning; use `if` rather than `(( )) &&` so a false test cannot abort under `set -e`.
- [x] Assert both units in `tests/skills-pipeline.bats`.
- [x] Prove the byte assertion red with the fold disabled, confirming the mutation landed first.
- [x] Confirm no new `shellcheck` findings against `main`.
- [x] Coordinate with the parallel session, since #1495 is parked on the same cap.
