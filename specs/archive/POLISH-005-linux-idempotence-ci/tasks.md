---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - POLISH-005-linux-idempotence-ci

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/polish-005-linux-idempotence-ci`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] Add formal idempotence tests in `tests/verify-setup.bats` Section 12 (asserting exit 0, zero config diff, no duplicate entries, clean git repo)
- [x] Fix declarative convergence in `setup-linux.sh` (normalize `permissions.allow` sorting on template bootstrap, re-enforce rc files at end of setup)
- [x] Validate integration container in Docker: 63/63 tests passing with 0 diff

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/POLISH-005-linux-idempotence-ci/features.json`):

```json
[
  {
    "id": "POLISH-005-linux-idempotence-ci-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
