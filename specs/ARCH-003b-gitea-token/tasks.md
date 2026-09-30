---
tags: [spec, tasks, templates]
created: "2026-09-29"
---

# Tasks - ARCH-003b-gitea-token

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/gitea-teledyne-token`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] Add the declarative `GITEA_TELEDYNE_TOKEN` registry mapping
- [x] [AC2] Run the secrets parser/command test packages
- [x] [AC2] Verify `dotf secrets ls` exposes only the ID and environment name
- [x] [AC3] Store the operator-created token through the hidden `set` prompt
- [x] [AC3] Verify resolution with `verify --require-all`

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by verification
- [x] Every acceptance criterion has a matching entry in `features.json`
- [x] Go tests pass
- [x] Registry parsing succeeds
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/ARCH-003b-gitea-token/features.json`):

```json
[
  {
    "id": "ARCH-003b-gitea-token-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
