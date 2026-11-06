---
tags: [spec, tasks, templates]
created: "2026-11-05"
id: "BUG-164-copilot-settings-guard"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-164-copilot-settings-guard

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from `origin/main`: `fix/3193-copilot-settings-guard`
- [x] Work-gate verified: #3193 is open and assigned
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC1] [AC2] Add `cli/internal/copilotsettings/policy_test.go` with a synthetic policy/settings fixture; run `go test ./internal/copilotsettings`; expected: FAIL because the package does not exist.
- [x] [AC2] Add `ai/copilot/retired-settings.json` and strict version-1 policy loading with exact dotted object paths.
- [x] [AC1] [AC4] [AC7] Implement read-only inspection plus leaf removal that preserves siblings and prunes only empty parents; malformed/non-object files fail.
- [x] [P] [AC3] [AC4] Add failing doctor tests for detect-only output, value redaction, fix/backup, malformed input, missing file, and second-run PASS.
- [x] [AC3] [AC4] Implement the Copilot settings doctor section through the shipped manifest entry and `deploy.EditJSON`.
- [x] [P] [AC5] [AC6] Add a failing optional repair-callback test to `cli/internal/deploy/run_test.go`; expected: deploy reports a repair as a change and the dry-run writes nothing.
- [x] [AC5] [AC6] Add the optional repair seam to `deploy.RunOptions` and preserve existing behavior when it is nil.
- [x] [P] [AC5] [AC6] Add failing command/converge tests proving bare and named deploy plus two converge runs use the same Copilot repair.
- [x] [AC5] [AC6] Wire the shared Copilot repair into `dotf deploy` and `configs-deploy`.
- [x] [AC7] Add unreadable/malformed policy and settings cases; assert no fallback creates or replaces the file.
- [x] [AC8] Add a static regression proving the new package does not execute `copilot config --list`.
- [x] [AC1] [AC2] [AC3] [AC4] [AC5] [AC6] [AC7] [AC8] Run exact `features.json` commands, focused `go vet`, formatting, and synthetic cross-OS tests.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [ ] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/BUG-164-copilot-settings-guard/features.json`):

```json
[
  {
    "id": "BUG-164-copilot-settings-guard-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
