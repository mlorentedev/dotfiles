---
tags: [spec, tasks, templates]
created: "2026-10-17"
---

# Tasks - CLI-094-stopped-review-leaves-a-record

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [ ] Branch created from main: `feat/cli-094-review-record`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions" (the `N` measurement is the one that gates the wedge increment)

## Implementation

- [x] [AC6] Measure the longest legitimate tool call in the surviving local transcripts (`durationMs` on `tool_execution_end` events), pick `N`, record both in `verification.md` (done 2026-10-17: 96.9s, N = 10 minutes). Every wedge-watcher task below depends on this one.
- [x] [P] [AC5] Write failing test: `dotf spec archive` on a `review.md` with `verdict: PENDING` names a provisional, unsigned record
- [x] [AC5] Teach the review parser a PENDING verdict that always refuses, with that wording
- [x] [P] [AC3] Write failing test: `ReviewPromptIn` asks both signers for a provisional `review.md` with `verdict: PENDING` first
- [x] [AC3] Add the write-first instruction to `ReviewPromptIn`
- [x] [AC1] [AC2] (after task 1) Write failing tests with a fake runner: silent transcript is stopped at the stall timeout (exit not 124, stderr names the last tool call); a growing transcript, and one silent for 97s, are never stopped
- [x] [AC1] [AC2] Add the stall watcher beside the cap (`RunWithLimits`; `RunWithDeadline` is now its cap-only wrapper), with `--stall-timeout` on `spec deadline` and `spec review`
- [x] [AC4] Write failing test: a stopped run prints "provisional review.md kept" or "review.md was not rewritten" and exits non-zero
- [x] [AC4] Report which in the stop path of `cli/internal/cmd/spec.go`
- [x] Refactor for clarity (one termination path for cap and wedge)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-094-stopped-review-leaves-a-record/features.json`):

```json
[
  {
    "id": "CLI-094-stopped-review-leaves-a-record-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
