---
id: lesson-295
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, verification, guard, fail-closed, retrieval, sdd]
---

# 295 — A check written without reading the guard lessons rediscovered three of them

## What happened

SKILL-001's AC6 rests on one check: that no retired skill remains in any harness surface. Four independent review rounds were needed before it passed. Every finding was a rule this directory already held:

| Round | What the check did | The lesson it repeated |
|---|---|---|
| 1 | Read skill names only, so a foreign skill named `audit` made it red. | Match provenance, not shape. |
| 2 | Hand-listed three of the six instruction files and missed opencode's and pi's. | 289: a list tested by looping over itself cannot see a missing member. |
| 3 | Passed when jq emitted CRLF, when the manifest listed no target, or when the manifest could not be read. Its bats test walked the same manifest, so it shrank with the check. | 290: fail closed; 289 again. |
| 4 | The test for one surface skipped when that surface was empty. | 287: a guard that skips is a guard that passes. |

## The rule

1. **Before writing a check or a guard, read the guard lessons**: at least 287, 289 and 290. Capture was not the problem here, since the lessons were on file. What failed is that nobody opened them.
2. **A check fails closed.** If it could not look, because of a missing tool, an unreadable input or an empty target list, it exits non-zero, never 0.
3. **Derive targets from the source of truth**, here `harness/manifest.json`, and test by seeding every target the source declares. A test must also fail when the source declares none.

Refs: SKILL-001 (#1692, #1724); `scripts/check-retired-skills.sh`. The retrieval gap is item 6 of the vault research note `10_projects/dotfiles/research/2026-09-25-knowledge-capture-routing.md`.
