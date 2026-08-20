---
tags: [spec, tasks, templates]
created: "2026-08-20"
---

# Tasks - HARNESS-041-ci-path-filtering

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/harness-041-ci-path-filtering`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] Write failing Bats test for CI path filtering (`tests/ci-path-filtering.bats`)
- [x] Add `changes` job (`dorny/paths-filter@v3`) to `.github/workflows/ci.yml`
- [x] Add `needs: [changes]` to matrix jobs (`lint`, `lint-powershell`, `test`, `test-windows`, `integration`)
- [x] Add conditional guards to all heavy steps in matrix jobs
- [x] Verify Bats test suite and Go test suite pass

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-041-ci-path-filtering/features.json`):

```json
[
  {
    "id": "HARNESS-041-ci-path-filtering-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
