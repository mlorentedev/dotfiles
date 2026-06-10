---
tags: [spec, tasks]
created: "2026-06-09"
---

# Tasks - REFACTOR-012-init-spec-issue-gate

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `refactor/init-spec-issue-gate` (worktree)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] Rewrite `tests/init-spec.bats` for the issue gate (stubbed `gh`): open-issue pass, missing/closed fail, bypass flags, no-11-tasks regression
- [x] Implement issue gate in `scripts/init-spec.sh` (`--issue`, `--force-no-gate`, `--force-no-vault` alias, remove `--task` + vault lookup)
- [x] Port to `scripts/init-spec.ps1` (`-Issue`, `-ForceNoGate`, `-ForceNoVault` alias; ASCII-only)
- [x] Update `AGENTS.md` SDD-workflow wording (work-gate = open issue; new flag names)
- [x] Update vault `/spec` skill SSOT + run `compile-harness --refresh` → refreshed `harness/skills/spec/SKILL.md` in this PR

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] shellcheck passes
- [x] Full bats suite passes (no regressions)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder and closing #304
