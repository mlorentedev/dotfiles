---
tags: [spec, tasks, templates]
created: "2026-11-16"
id: "2026-11-16-sysupgrade-windows-elevation"
type: spec-tasks
status: active
owner: manu
---

# Tasks - 2026-11-16-sysupgrade-windows-elevation

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/3553-sysupgrade-windows-elevation`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] [AC3] Write failing test: non-elevated Windows run with a failing id, over the three statuses
      observed on 2026-11-16
- [x] [AC4] Pin the existing AC5 test to an elevated run
- [x] [P] [AC5] Write failing case in `TestSystemUpgradeSkipReadsOnlyTheStepsSkip`
- [x] [AC1] [AC2] Leave failed ids waiting in `systemUpgrade.windows` when not elevated
- [x] [AC1] [AC2] Write failing tests: a machine-wide id is never attempted (apply, plan, unknown scope), after
      a real run raised an administrator prompt per package
- [x] [AC1] [AC2] `splitMachineWide`: read the scope from `winget list --scope machine` before the upgrade loop
- [x] [AC3] Probe tolerates what is still behind on a non-elevated Windows run
- [x] [AC6] `processElevated`: token on Windows, euid elsewhere, behind the existing `root` seam
- [x] [AC5] `systemUpgradeSkip` reads the waiting marker
- [x] Document: `dotf update` help, self-deploy runbook, lesson 609

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

Minimal `features.json` skeleton (drop into `<repo>/specs/2026-11-16-sysupgrade-windows-elevation/features.json`):

```json
[
  {
    "id": "2026-11-16-sysupgrade-windows-elevation-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
