---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - WIN-005-windows-defaults-script

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/win-005-windows-defaults`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (R1-R5 all carry their resolution inline)

## Implementation

- [x] Write failing structural bats tests (`tests/windows-defaults.bats`): script exists, doc block, StrictMode, HKCU-only invariant (no `HKLM` token anywhere), no `explorer.exe` restart, named-constants table (R2), PSScriptAnalyzer + ParseFile via `tests/winpath.bash` (red: 18/19 fail before implementation)
- [x] Write failing setup-integration bats tests (same file): `setup-windows.ps1` declares `-WithDefaults` switch, invocation gated on the flag (default OFF), script deployed to ScriptsDir, README documents the flag
- [x] Write failing Pester behavior tests (`tests/windows-defaults.Tests.ps1`, Windows-only via discovery-time `-Skip`): sandboxed `-Root HKCU:\Software\dotfiles-win005-test` run applies all defaults; second run reports 0 changes (idempotency, R4); non-HKCU root is rejected
- [x] Implement `scripts/windows-defaults.ps1`: defaults table as named constants (15 entries), injectable `-Root` (HKCU-validated), read-before-write logging, Win10/Win11 branch on build 22000 (R3), explorer-restart note in output (R1)
- [x] Wire `setup-windows.ps1`: `-WithDefaults` switch + gated invocation + deploy to ScriptsDir + explicit flag forwarding through the BUG-005 re-exec
- [x] Document the opt-in flag in README
- [x] Run full local suite (bats subset + Pester) green (bats 20/20, Pester 4/4)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass (n/a -- PowerShell/bats only)
- [x] Lint passes (lint-powershell + bats analyzer tests green on PR #331)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (PR #331, merged)

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/WIN-005-windows-defaults-script/features.json`):

```json
[
  {
    "id": "WIN-005-windows-defaults-script-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
