---
tags: [spec, tasks, templates]
created: "2026-09-03"
---

# Tasks - CI-002

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main (PR 1: #1482, `91e38344`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] [AC3] The skip guard, first in the reconcile chain and loud about what it did not verify. Shipped in both twins in #1482; it moved into `dotf pi packages apply` (`cli/internal/cmd/pi.go`, `skipEnv`) with HARNESS-139 (#1628), which both twins now call.
- [x] [AC4] [AC5] The `pi` path filter and the `pull_request`-only `env:` line on `test-windows` (#1482).
- [x] [AC6] Mutation pass: five mutations in #1482. Re-proven at close against the Go guard: f5 probes for pi before the skip, and the test fails.
- [x] [AC2] Close-out fix: `TestPiPackagesApplySkipIsFirstAndLoud` could not fail. Its fixture answers every probe, so a probe before the skip passed unseen. Any probe now fails the test.
- [x] PR 2, the wider filter audit: measured and **declined**. See `verification.md`, "PR 2".

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CI-002/features.json`):

```json
[
  {
    "id": "CI-002-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
