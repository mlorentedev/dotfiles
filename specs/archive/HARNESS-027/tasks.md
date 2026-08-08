---
tags: [spec, tasks, templates]
created: "2026-08-08"
---

# Tasks - HARNESS-027

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/pr-review-triage-skill`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] Risks resolved in the design: vendor coupling, status-notice-as-pass, ceremony

## Implementation

> Replace these with the actual steps for this feature. Keep them small (one commit each) and in TDD order.
> The `[P]` / `[AC<n>]` markers are optional — see the legend above. Behaviors 1 and 2 below are independent, so their *first* test task carries `[P]`.

- [x] [AC1] [AC2] Author the skill: watch the run once, report CI, name the two failure shapes worth recognising
- [x] [AC5] Reviewer step keyed to surface and content, with the "did it actually run" check and its three outcomes
- [x] [AC7] Strip every vendor name — the author earns a reply address, nothing else
- [x] [AC3] Disposition table: apply / defer / skip, one rationale each, scope and correctness left as explicit judgements
- [x] [AC4] Confirmation gate before acting; merge excluded unconditionally
- [x] [AC6] Early exit in one line when there is nothing to dispose of
- [x] Record refreshed from the vault SSOT

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by a check in `features.json`
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Lint passes (frontmatter validated by `compile-harness.sh --check`)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#822)

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-027/features.json`):

```json
[
  {
    "id": "HARNESS-027-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
