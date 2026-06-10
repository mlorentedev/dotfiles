---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - WIN-004-windows-ci-runner

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [ ] Branch created from main: `feat/WIN-004-windows-ci-runner`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] Write failing bats test: BUG-015 probe must SKIP (not FAIL) when Claude Code never ran (`tests/setup-linux.bats` "claude-mem hook probe n/a" parity assertion)
- [x] Gate the BUG-015 probe on `installed_plugins.json` in `scripts/healthcheck.ps1` (CI-clean-machine FAIL eliminated)
- [x] Mirror the gate in `scripts/healthcheck.sh` (cross-OS parity)
- [x] Add `test-windows` job to `.github/workflows/ci.yml`: choco age -> secrets/vault sandbox -> `setup-windows.ps1` under PS 5.1 (BUG-005 re-exec) -> core-tools fallback -> `healthcheck.ps1` -> Pester (`tests/*.Tests.ps1`) -> bats subset under Git Bash (pinned `BATS_VERSION`)
- [ ] Open PR; iterate on the live `windows-latest` run until green (cannot be executed locally — the PR run IS the execution verification)
- [ ] Measure wall-time on the green run; record in `verification.md` (AC6: <= 7 min increase)

## Post-merge

- [ ] Add `test-windows` to branch protection required checks for `main` (repo admin): `gh api -X PATCH repos/mlorentedev/dotfiles/branches/main/protection/required_status_checks -f 'contexts[]=test-windows'` (or via Settings UI)

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

Minimal `features.json` skeleton (drop into `<repo>/specs/WIN-004-windows-ci-runner/features.json`):

```json
[
  {
    "id": "WIN-004-windows-ci-runner-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
