---
tags: [spec, tasks, templates]
created: "2026-08-15"
---

# Tasks - HARNESS-072-pr-stewardship

> TDD order. One task = one focused commit. `[AC<n>]` maps a task to an acceptance criterion in `proposal.md`; `[P]` marks a task with no dependency on another unchecked one.

## Setup

- [x] Branch created from main: `feat/harness-072-pr-stewardship`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — the two
      kubelab objections are resolved into AC3 and AC4; the coverage guard is
      decided (b) rather than left open

## Implementation

Ordered so the guard exists before the region it protects — the region was added
under an assertion that would have caught a partial injection, not after one.

- [x] [AC3] [AC4] [AC5] Revise the acceptance criteria for the two objections,
      and rewrite the region draft to match (`37398c1`)
- [x] [AC2] Correct AC2: `--check` cannot verify coverage, and was named as the
      mitigation for the one risk it is blind to (`f1641c4`)
- [x] [AC8] Write `check_coverage` + three bats cases; observe red first, on the
      real tree — `pr-sizing` was doctrine-only by an undeclared convention
      (`bf2eda9`)
- [x] [AC1] [AC3] [AC4] [AC5] Add the `## PR Stewardship` section to the vault
      pattern beside `definition-of-done` (vault `2e351f1a`)
- [x] [AC1] [AC2] Declare the region, inject it into both targets and the
      doctrine payload, `--refresh` (`334c1a7`)
- [x] [AC6] Retrigger `pr-review-triage` on whichever of checks and reviewers
      lands later (vault `35d93f0a`, record `fdbe0b4`)
- [x] Keep six unrelated drifted records out of the change's diff (`b678103`)

## Closing

- [x] Every acceptance criterion is covered by at least one feature with a
      non-vacuous verification command
- [x] Every acceptance criterion has a matching entry in `features.json`
- [x] Lint passes (`shellcheck scripts/compile-harness.sh`, `bash -n`)
- [x] Tests pass (`bats tests/compile-harness.bats` — 47/47)
- [x] No unrelated changes in the diff — the record sync is its own commit
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] Adversarial review passes before archive (`dotf spec review HARNESS-072-pr-stewardship`)

## Machine-readable features

`features.json` sits beside this file, one feature per acceptance criterion
(f1–f8), each with a shell command whose exit 0 is the pass condition.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the
harness, after running `verification` and capturing exit code 0, may set that
terminal state. Every entry here is `pending` with empty `evidence`; the
session's observed output lives in `verification.md`, which is a claim about a
run, not a substitute for one.
