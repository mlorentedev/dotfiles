---
tags: [spec, tasks, templates]
created: "2026-08-12"
---

# Tasks - HARNESS-070-deploy-convergence

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `fix/harness-deploy-convergence`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (all three resolved by design decisions recorded there)

## Implementation

- [x] [P] [AC2] Write failing Go test for dotf-version-drift FAIL (checks_test.go)
- [x] [AC2] Change `checkOptionalTools`'s dotf-self branch from `rep.Warn` to `rep.Fail`
- [x] [P] [AC1] Write failing Go test for orphan-mirror-record detection + `--fix` prune
- [x] [AC1] Implement `checkHarnessMirrorOrphans` in `checks_deploy.go`; wired `opts.Fix` through `checkHarnessDrift` (doctor.go:95)
- [x] [AC5] Write failing Go test: symlink at an unmanaged name (no `harness/skills/` record) does not FAIL
- [x] [AC5] Narrow `checkDeployedSkillSymlinks` to only flag symlinks at managed record names; filed + fixed #943 (BUG-074) in the same change
- [x] [P] [AC3] Manually verify `--deploy`-only run updates all 4 instruction files (sandboxed `HOME=/tmp/fake-home`); extended `agents.presence[]` with `source`/`requires_command`
- [x] [AC3] Implement `deploy_instructions` in `do_deploy`, ordered first (before `deploy_skills`/`deploy_agents`); also fixed a pre-existing `deploy_agent_presence` bug found while smoke-testing (logged success even when injection was skipped)
- [x] [AC4] Write failing Go test for stale-instruction-file detection (region-stripped compare)
- [x] [AC4] Implement `checkInstructionDrift` + `stripHarnessRegions`, pinned against the bash marker constants via `TestHarnessMarkerConstants`
- [x] Refactor for clarity / dedupe once all four land

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] `features.json` entries added for each criterion
- [x] Type checks pass (`go build ./... && go vet ./...`)
- [x] Lint passes (`golangci-lint run` — 0 issues; `shellcheck` — clean)
- [x] No unrelated changes in the diff (no scope creep — `scripts/`, `sensitive/` orphan pruning from #802 explicitly deferred)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder, `Closes #843 #869 #828 #943`, `Refs #802`

## Machine-readable features

See sibling `features.json`.
