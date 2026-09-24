---
tags: [spec, tasks, templates]
created: "2026-09-23"
---

# Tasks - GUARD-017-protection-as-code

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/protection-as-code` (worktree `dotfiles-wt-protection-as-code`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [ ] Owner confirms the declaration path (`forge/`), the one open question in `proposal.md`

## Implementation: PR-A (declare and detect, read-only)

- [x] [P] [AC4] Failing tests: schema rejects a repo with neither object nor state, and a state without a reason → `TestProtectionSchema*`
- [x] [AC4] `forge/branch-protection.schema.json` + loader in `cli/internal/forge`
- [x] [P] [AC1] Failing tests: GET-shape normalisation (`enabled` wrappers, `checks[]` with `app_id`) and field-by-field diff → `TestProtectionNormalise*`, `TestProtectionDiff*`
- [x] [AC1] Normalise + diff
- [x] [AC3] `TestProtectionApprovalsRationalePinned` + policy block
- [x] [AC1] `dotf forge protection check`, exit contract tested → `TestForgeProtectionCheckExit*`
- [x] [AC2] Failing test + `branch-protection` doctor section → `TestCheckBranchProtection`
- [x] [AC1] `forge/branch-protection.json` declaring today's live state; `check` exits 0 live

## Implementation: PR-B (apply)

- [ ] [AC5] Failing tests against a fake forge: complete PUT body, re-read assert, `changed=0` on the second run → `TestProtectionApply*`
- [ ] [AC5] Reported-context preflight → `TestProtectionApplyRefusesUnreportedContext`
- [ ] [AC6] Declare `spec-gate` on dotfiles `main`; the owner runs `apply`

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

Minimal `features.json` skeleton (drop into `<repo>/specs/GUARD-017-protection-as-code/features.json`):

```json
[
  {
    "id": "GUARD-017-protection-as-code-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
