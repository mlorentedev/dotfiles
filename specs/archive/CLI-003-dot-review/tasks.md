---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - CLI-003-dot-review

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/dot-review`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (R1/R2 resolved by design; R3/R4 are documentation/test obligations)

## Implementation

- [x] Write failing table-driven tests for `dot review`: provider resolution (nan/openrouter/unknown), env validation errors naming the variable, empty stdin, `--max-bytes` exceeded, HTTP 5xx, timeout, happy path against an in-process `httptest` OpenAI-compatible mock (asserting auth header + default model)
- [x] Implement `cli/cmd/dot/review.go` (stdlib `net/http` only) + wire into root — make tests green
- [x] Update `cli/README.md`: review section (usage, providers, env vars, exit codes) + explicit privacy note (diffs leave the machine)
- [x] Manual QA: live NaN review of a real diff on Linux; inspect Windows CI output; file issues for findings

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-003-dot-review/features.json`):

```json
[
  {
    "id": "CLI-003-dot-review-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
