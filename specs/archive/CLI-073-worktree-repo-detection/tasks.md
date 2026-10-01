---
tags: [spec, tasks, templates]
created: "2026-09-29"
---

# Tasks - CLI-073-worktree-repo-detection

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/worktree-repo-detection`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> Replace these with the actual steps for this feature. Keep them small (one commit each) and in TDD order.
> The `[P]` / `[AC<n>]` markers are optional — see the legend above. Behaviors 1 and 2 below are independent, so their *first* test task carries `[P]`.

- [x] [AC1] Add a failing worktree row to
  `TestCheckRepoDirResolves`.
- [x] [AC1] Pass the injected `System` into `checkRepoDirResolves` and resolve
  Git's top-level checkout path.
- [x] [AC2] Re-run the normal-checkout row.
- [x] [AC3] Re-run the missing and non-checkout rows.
- [x] [AC4] Add a failing checkout-subdirectory row and compare the configured
  path with `git rev-parse --show-toplevel`.
- [x] [AC5] Add a failing session-start regression from a linked worktree root
  and subdirectory.
- [x] [AC5] Resolve checkout root and main project identity from the `.git`
  pointer before emitting hive/specs/lessons/triage context.
- [x] Add a failing doctor regression for inherited `GIT_DIR` /
  `GIT_WORK_TREE`, then run the Git probe with repository-local variables
  removed from its subprocess environment.
- [x] Add a failing session-start regression for a symlink to a checkout
  subdirectory, then resolve the physical path before walking ancestors.
- [x] Add a failing `--separate-git-dir` identity regression, then keep the
  checkout directory name for generic pointer layouts.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder: #1835

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-073-worktree-repo-detection/features.json`):

```json
[
  {
    "id": "CLI-073-worktree-repo-detection-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
