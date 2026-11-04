---
tags: [spec, tasks, templates]
created: "2026-11-04"
id: "CLI-129-initrepo-lane-contract"
type: spec-tasks
status: active
owner: manu
---

# Tasks - CLI-129-initrepo-lane-contract

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/CLI-129-initrepo-lane-contract`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> New code lives in `cli/internal/initrepo/lane.go`, tests in `lane_test.go`, templates in `cli/internal/initrepo/templates/`. Follow `agents.go` for the section and `scaffold.go` for file writes.

- [x] [AC1] Failing test: `BootstrapLane` on an empty dir writes `.dotf/lane.json` that `lane.LoadConfig` loads (empty queue, defaults)
- [x] [AC1] Embed `templates/lane.json` and implement `BootstrapLane` for the config file
- [x] [AC2] Failing test: the scaffolded `.dotf/lane-prompt.md` renders with `lane.PromptData`
- [x] [AC2] Embed `templates/lane-prompt.md` (generic, only `.Number .Title .Branch .Remaining`) and write it
- [x] [AC3] Failing test: the `## Lanes` section is created, appended after prior content, and anchored by its heading
- [x] [AC3] Embed `templates/agents-lanes-section.md` and implement the append-if-absent section
- [x] [AC4] [AC5] Failing tests: second run is byte-identical and reports `unchanged`; existing files are left alone
- [x] [AC6] Failing test: `--force` replaces the prompt and the section in place and never touches `lane.json`
- [x] [AC4] [AC5] [AC6] Implement the unchanged/skip/replace outcomes (`LaneResult`)
- [x] [AC7] Failing test: `Init` runs the lane step after AGENTS.md; `--dry-run` writes nothing
- [x] [AC7] Wire the step into `orchestrator.go` and add `dotf init lane [--repo] [--dry-run] [--force]` in `cmd/init.go`
- [x] Refactor for clarity (functions under 40 lines, nesting under 4)
- [x] [AC8] File the kubelab rollout ticket and link it from `verification.md`

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

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-129-initrepo-lane-contract/features.json`):

```json
[
  {
    "id": "CLI-129-initrepo-lane-contract-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
