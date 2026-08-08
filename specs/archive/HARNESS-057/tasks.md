---
tags: [spec, tasks, templates]
created: "2026-08-08"
---

# Tasks - HARNESS-057

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/skill-frontmatter-contract`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] Risks resolved before implementing: the guard was proved non-decorative, dates derived not stamped

## Implementation

> Replace these with the actual steps for this feature. Keep them small (one commit each) and in TDD order.
> The `[P]` / `[AC<n>]` markers are optional — see the legend above. Behaviors 1 and 2 below are independent, so their *first* test task carries `[P]`.

- [x] [AC1] Backfill the missing keys across the store, `created` from each file's first commit
- [x] [AC1] Normalise key order so the seven read the same way in every file
- [x] [P] [AC2] Prove the guard bites: strip a key, `--check` must fail (red) and pass restored (green)
- [x] [AC2] Tighten `required[]` to the seven and document each property in the schema
- [x] [AC2] Correct the stale validator comment — it still claimed a two-key minimum
- [x] [AC3] Record the vendored-provenance rule in the pattern, not the schema
- [x] [P] [AC4] Test: the schema requires the seven, every record satisfies them, `--check` rejects a drop, a vendored skill has provenance and an attribution row

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Lint passes (`shellcheck scripts/compile-harness.sh`, clean)
- [x] No unrelated changes in the diff — the 34 record edits are `--refresh` output, and the one untracked record belongs to another PR
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-057/features.json`):

```json
[
  {
    "id": "HARNESS-057-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
