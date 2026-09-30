---
tags: [spec, tasks, templates]
created: "2026-09-29"
---

# Tasks - CLI-091e-dotf-as-a-product

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `docs/cli-help-scrub`
- [x] `proposal.md` is complete for the row in flight (E1)

## Implementation

Row E1:

- [x] [AC1] Failing test `TestHelpTextHasNoInternalReferences` in `cli/internal/cmd/help_text_test.go` (49 offending lines at the start)
- [x] [AC1] Rewrite the offending help text in `cli/internal/cmd/*.go` so it describes behaviour
- [x] [AC2] Rewrite `cli/README.md` for users

Later rows add their tasks here when they start.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test or check
- [ ] Every acceptance criterion has a matching entry in `features.json`
- [ ] Lint passes
- [ ] `verification.md` filled in

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-091e-dotf-as-a-product/features.json`):

```json
[
  {
    "id": "CLI-091e-dotf-as-a-product-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
