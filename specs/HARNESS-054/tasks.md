---
tags: [spec, tasks, templates]
created: "2026-08-08"
---

# Tasks - HARNESS-054

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/doctrine-parity-agy-codex`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] Open questions in `proposal.md` are recorded as stated assumptions — neither agent is installed here, so marker tolerance is reasoned, not observed

## Implementation

> Replace these with the actual steps for this feature. Keep them small (one commit each) and in TDD order.
> The `[P]` / `[AC<n>]` markers are optional — see the legend above. Behaviors 1 and 2 below are independent, so their *first* test task carries `[P]`.

- [x] [P] [AC1] Failing test: `--deploy` creates the doctrine file for a surface that has none
- [x] [AC1] `deploy_doctrine` renders the compact payload (enforced rules + presence) and injects it
- [x] [P] [AC3] Failing test: injection preserves user content and is idempotent across re-deploys
- [x] [AC3] Replace-in-place when our region exists, append when it does not; never overwrite the file
- [x] [P] [AC4] Failing test: a file over the platform's documented cap warns
- [x] [AC4] Post-injection character count against the manifest's `char_cap`
- [x] [P] [AC5] Failing test: a shadow file that wins at read time warns
- [x] [AC5] `shadowed_by` check before injection
- [x] [AC2] Parity test: every surface the manifest declares carries a region
- [x] [AC6] Rationale + byte numbers + sources recorded in `manifest.json` next to each row
- [x] Curator persona drops its `targets[]` enumeration so presence reaches the new surfaces

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Lint passes (`shellcheck scripts/compile-harness.sh`, clean)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#819)

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-054/features.json`):

```json
[
  {
    "id": "HARNESS-054-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
