---
tags: [spec, tasks, templates]
created: "2026-11-14"
id: "MEMORY-021-shared-thread-file-report"
type: spec-tasks
status: active
owner: manu
---

# Tasks - MEMORY-021-shared-thread-file-report

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/3521-archive-followups`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] Failing tests: `TestArchiveParkedThreadLeavesAFileAnotherKeyHolds` asserts `Shared`, `TestReportParkThreadsNamesASharedThreadFile` asserts the park and sweep lines, `TestSweepReportRecordsThreadArchives` asserts `SharedThreads`
- [x] [AC1] `moveThreadFile` returns `shared`; `ParkArchive.Shared`, `SweepReport.SharedThreads`, and the report lines in `park_threads.go`
- [x] [AC2] Split `ArchiveParkedThread` into `parkTarget` and `archiveParked`; move `takeThreads`'s match and scan into `takesThread` and `blockEnd`

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

Minimal `features.json` skeleton (drop into `<repo>/specs/MEMORY-021-shared-thread-file-report/features.json`):

```json
[
  {
    "id": "MEMORY-021-shared-thread-file-report-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
