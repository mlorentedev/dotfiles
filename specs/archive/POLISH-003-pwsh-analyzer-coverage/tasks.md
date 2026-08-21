---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - POLISH-003-pwsh-analyzer-coverage

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/polish-003-pwsh-analyzer-coverage`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] Write failing Bats test for dynamic discovery (`tests/pwsh-analyzer-coverage.bats`)
- [x] Update `.github/workflows/ci.yml` `lint-powershell` step to use `Get-ChildItem -Recurse -Include *.ps1, *.psm1`
- [x] Run Bats suite to verify test passes
- [x] Create PR to measure full repository PSScriptAnalyzer findings on CI runner

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in after CI measurement
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/POLISH-003-pwsh-analyzer-coverage/features.json`):

```json
[
  {
    "id": "POLISH-003-pwsh-analyzer-coverage-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
