---
tags: [spec, tasks, templates]
created: "2026-11-16"
id: "BUG-173-converge-step-dependencies"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-173-converge-step-dependencies

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/3554-converge-step-dependencies`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] Failing tests: a failure holds back only its dependents; every failure is in the error (`needs_test.go`)
- [x] [AC1] [AC2] [AC3] `Needer`, `downSet`, `Unreached` (`needs.go`); `RunObserved` continues past a failure and joins the errors (`converge.go`)
- [x] [AC4] [AC5] Tests and default for an undeclared step, and for needs outside the run
- [x] [AC6] Tests and runner change: no revisit of a failed step; a failed revisit is not terminal
- [x] [AC3] Console: the not-reached line prints in place; goldens regenerated
- [x] [AC7] [AC8] Declarations for the 24 steps (`needs_registry.go`) and the graph tests (`needs_registry_test.go`)
- [x] [AC9] `dotf doctor` prints one line for a multi-line recorded error
- [x] ADR-064, runbook and `converge --help` text

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes (CI: the local checkout is CRLF, so the local linter reports gofmt noise); CI lint, lint-powershell and cli-lint green on #3560
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder: #3560

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/BUG-173-converge-step-dependencies/features.json`):

```json
[
  {
    "id": "BUG-173-converge-step-dependencies-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
