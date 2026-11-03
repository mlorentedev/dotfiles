---
tags: [spec, tasks, templates]
created: "2026-11-02"
id: "CLI-133-init-actionlint-every-stack"
type: spec-tasks
status: active
owner: manu
---

# Tasks - CLI-133-init-actionlint-every-stack

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/init-actionlint`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] [AC4] Write failing tests in `cli/internal/initrepo/lint_test.go` (build failed: `WriteLintOpts` undefined)
- [x] [AC2] Add `templates/lint.yml` from ci-workflows' `lint` job, naming the workflow directory for actionlint
- [x] [AC1] [AC2] Extract `writeWorkflow` from `WriteCIOpts`; add `WriteLintOpts`; add the `lint` row to `Init`
- [x] [AC3] Run actionlint and `check-actions-pinned.sh` on a scaffolded repository, with a negative control
- [x] [AC2] [AC4] Mutation checks: the directory substitution and the version pin each fail their test when broken
- [x] File #3093 (structure step) and #3094 (unpinned ci templates)

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

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-133-init-actionlint-every-stack/features.json`):

```json
[
  {
    "id": "CLI-133-init-actionlint-every-stack-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
