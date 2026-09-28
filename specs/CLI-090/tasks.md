---
tags: [spec, tasks, templates]
created: "2026-09-28"
---

# Tasks - CLI-090

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/bootstrap-recovery-update`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] Write a failing BATS test that pipes `install-dotf.sh` into Bash
  outside a checkout and installs a fixture release into an isolated `$HOME`.
- [x] [AC1] Make the POSIX installer self-contained for its raw-pipe mode while
  preserving its sourced-helper behavior.
- [x] [AC2] Write failing POSIX and PowerShell tests for resolving a latest
  release when neither an argument, environment pin, nor checkout pin exists.
- [x] [AC2] Implement validated latest-release resolution in both native
  installers without weakening explicit pin behavior.
- [x] [AC3] Write failing tests for malformed release metadata and failed
  verification preserving the pre-existing binary.
- [x] [AC3] Implement failure handling and atomicity required by those tests.
- [x] [AC4] Update the bootstrap and recovery documentation with the native
  commands, semantic distinction, and version assertion.
- [x] [AC1] [AC2] [AC3] [AC4] Run targeted installer tests, then the relevant
  shell, PowerShell, and Go validation.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [ ] Lint passes (ShellCheck is unavailable in the WSL test environment)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-090/features.json`):

```json
[
  {
    "id": "CLI-090-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
