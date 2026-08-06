---
tags: [spec, tasks]
created: "2026-06-10"
---

# Tasks - AI-022-pi-harness-slot

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `feat/pi-harness-slot` (worktree)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] Extend `tests/skills-pipeline.bats`: pi gets /spec as a native skill (regular copy); Claude-only skill NOT exposed to pi; symlink sweep covers pi dir
- [x] Add pi entry to `harness/manifest.json` `skills.deploy[]`
- [x] Comment in `scripts/healthcheck.sh` documenting the deliberate pi exclusion from the strict symlink sweep

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] shellcheck passes
- [x] Full bats suite passes (no regressions)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder and closing #161
