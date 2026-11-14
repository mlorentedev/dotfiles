---
tags: [spec, tasks, templates]
created: "2026-11-14"
id: "MEMORY-020-one-archive-shape"
type: spec-tasks
status: active
owner: manu
---

# Tasks - MEMORY-020-one-archive-shape

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/3487-one-archive-shape`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC3] Failing tests for the park archive in `internal/mem`: key and forks appended to the dated file, pruned from MEMORY.md; a fork whose thread file has Awaiting Manu items stays and is named
- [x] [AC1] Replace `ArchiveAndPruneThread` and `pruneThreadFromMemory` with one extraction (`takeThreads`) shared by `--finished` and the park
- [x] [AC4] Failing tests: `worktree.Park` whose archive cannot be written still parks and carries the error in its result
- [x] [AC4] Thread the vault path through `DoneOptions` and `SweepOptions`, carry the report in `DoneResult` and `SweepReport`, drop `ParkLane`'s duplicate call, print it and fail in `worktree done`, `lane park`, `worktree sweep` and `lane apply`
- [x] [AC1] [AC2] Doctor contract read: thread files move to `archive/threads/`, entries stay byte-identical; `--finished` moves a live file and removes it last
- [x] Budget refusal and doctor warning name the dated archive and `--finished`
- [x] Review F1: `--finished` refuses, and a park leaves, a live thread file whose heading names another key
- [x] Review F3, F4, F6: ADR-060 and the memory runbook name the shape; the park's archive text is asserted exactly; the dated-archive append is its own function

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/MEMORY-020-one-archive-shape/features.json`):

```json
[
  {
    "id": "MEMORY-020-one-archive-shape-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
