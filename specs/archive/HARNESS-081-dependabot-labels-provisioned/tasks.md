---
tags: [spec, tasks, templates]
created: "2026-11-02"
id: "HARNESS-081-dependabot-labels-provisioned"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-081-dependabot-labels-provisioned

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/init-github-labels`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC1] Failing tests: missing labels created once, present ones (case-insensitive) left alone, second run creates nothing (`labels_test.go`)
- [x] [AC1] `ApplyDependabotLabels` in `cli/internal/initrepo/labels.go`
- [x] [AC2] Failing test, then dry-run path: lists, creates none
- [x] [AC3] Failing test, then no-declaration path: no `label` call at all
- [x] [AC4] Failing test, then a rejected create returns an error naming the label and repo; `dotf init github` returns it (non-zero exit)
- [x] Wire into `dotf init github` (`cli/internal/cmd/init.go`) and the `github` step of `dotf init` (`orchestrator.go`)
- [x] [AC5] Failing bats tests, then step 3b `apply_github_defaults` in `scripts/bitacora-rollout.sh`

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-081-dependabot-labels-provisioned/features.json`):

```json
[
  {
    "id": "HARNESS-081-dependabot-labels-provisioned-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
