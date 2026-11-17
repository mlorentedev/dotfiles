---
tags: [spec, tasks, templates]
created: "2026-11-16"
id: "BUG-174-pi-native-launch-windows"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-174-pi-native-launch-windows

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/3569-pi-native-launch-windows`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] [AC3] [AC4] Failing tests for `lane.NativeTarget` (`exec_native_test.go`) and for the real `Exec` on Windows (`exec_native_windows_test.go`)
- [x] [AC1] [AC2] [AC3] [AC4] `lane.NativeTarget` and `lane.MiseNative` (`exec_native.go`); `Exec` on Windows starts what `NativeTarget` returns
- [x] [AC2] Accept only an `.exe` from mise: the first version accepted mise's answer for opencode and bats, an extensionless npm stub with a `.cmd` beside it (measured on this machine before the commit)
- [x] [AC5] Windows test through the real `runChild`, failing when the launch goes back to `argv[0]`
- [x] [AC5] `secrets run`: `safeChildTarget` is the check and, when it passes, the file `runChild` starts, so the launch resolves once (`childTarget` behind it); a nested command keeps `refuseBatchShimChild`

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes (CI's `lint` job; the local CRLF checkout fails `gofmt` on every file)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#3570)

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/BUG-174-pi-native-launch-windows/features.json`):

```json
[
  {
    "id": "BUG-174-pi-native-launch-windows-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
