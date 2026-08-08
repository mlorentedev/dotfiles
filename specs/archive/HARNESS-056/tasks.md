---
tags: [spec, tasks, templates]
created: "2026-08-08"
---

# Tasks - HARNESS-056

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/definition-of-done`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] Risks recorded with their mitigations (theatre, second-source-of-truth, size)

## Implementation

> Replace these with the actual steps for this feature. Keep them small (one commit each) and in TDD order.
> The `[P]` / `[AC<n>]` markers are optional — see the legend above. Behaviors 1 and 2 below are independent, so their *first* test task carries `[P]`.

- [x] [P] [AC1] Author the Definition of Done in `pattern-change-lifecycle.md` under a stable anchor
- [x] [AC2] Register `definition-of-done` as an enforced id and add it to every inject list, doctrine included
- [x] [P] [AC2] Test: every enforced target and the compact payload inject the id
- [x] [AC2] Test: the injected region reaches the committed instruction files
- [x] [AC4] Test: the deployed compact payload carries it and stays under each documented char cap
- [x] [AC3] Extend `verification-before-completion` with the closing pass (verdict per item, skip = stated decision)
- [x] [P] [AC5] Test: the skill binds the standing orders instead of restating them
- [x] [AC4] `--refresh` then `--check`: no drift, caps respected

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Lint passes (`shellcheck scripts/compile-harness.sh`, clean)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-056/features.json`):

```json
[
  {
    "id": "HARNESS-056-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
