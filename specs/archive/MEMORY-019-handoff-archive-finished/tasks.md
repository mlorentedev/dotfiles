---
tags: [spec, tasks, templates]
created: "2026-11-13"
id: "MEMORY-019-handoff-archive-finished"
type: spec-tasks
status: active
owner: manu
---

# Tasks - MEMORY-019-handoff-archive-finished

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/3393-handoff-archive-finished`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] Failing test in `internal/mem`: `ArchiveThreadAs` on a finished body returns MEMORY.md without that thread (others byte-identical) and the block to append, equal to what `StoreThreadAs` would have placed
- [x] [AC1] Implement `ArchiveThreadAs` on top of `StoreThreadAs` and `pruneThreadFromMemory` with an exact-key mode (forks kept)
- [x] [AC1] Failing test for the archive file text: a new dated file opens with `# Superseded handoffs <date>`, an existing one gets the block after one blank line
- [x] [AC1] Implement the append, and wire `--finished` into `handoff-write`: archive, then thread file, then MEMORY.md, under the existing lock; stderr names the move
- [x] [AC2] Failing tests: `--finished` with an Awaiting Manu item is refused and writes nothing; a "None until X, then Y" thread without the flag stays
- [x] [AC2] Implement the refusal
- [x] [AC3] Failing test: agent B's `--finished` under agent A's key archives `<key>+B` only, and A's thread stays in MEMORY.md
- [x] [AC4] Failing tests: a repeated `--finished` write appends nothing and says `unchanged`; `--dry-run --finished` writes none of the three files and names the archive file on stderr
- [x] [AC4] Implement idempotence and the dry run
- [x] Help text of `handoff-write` names `--finished`

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

Minimal `features.json` skeleton (drop into `<repo>/specs/MEMORY-019-handoff-archive-finished/features.json`):

```json
[
  {
    "id": "MEMORY-019-handoff-archive-finished-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
