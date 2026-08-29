---
tags: [spec, tasks]
created: "2026-08-28"
---

# Tasks - HARNESS-092-harness-presence

## Setup

- [x] Branch: `feat/harness-presence` from `origin/main` (worktree `dotfiles-wt-presence`)
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left in `proposal.md`

## Implementation

- [x] [P] [AC1] Failing tests `TestBuildPresence_MatchesTheShellRendering`, `TestBuildPresence_RespectsTargetsAndSkipsAutonomous`, `TestBuildPresence_EmptyWhenNoPersonaTargetsTheHarness`
- [x] [AC1] `harness.BuildPresence`, `PresenceSHA` (LF form, 16 hex), `LoadPresence` (manifest `agents.presence[]` + `record_dir`)
- [x] [P] [AC2] Failing tests `TestDeployPresence_InjectsEveryTargetOnceAndKeepsTheRest`, `TestInjectPresence_ReplacesAStaleRegionInPlace`, `TestInjectPresence_HonoursCRLF`, `TestDeployPresence_AbsentTargetIsASkipAndEmptyRosterInjectsNothing`
- [x] [AC2] `harness.InjectPresence`, `DeployPresence`, `RenderPresence`, `PresenceStatus`; `dotf harness presence [--repo-root] [--home] [--render <agent>]` (`harness_presence.go` + cmd tests)
- [x] [P] [AC3] `compile-harness.sh`: `deploy_agent_presence` delegates, `build_agent_presence` wraps `--render`, `inject_agent_presence` deleted; absent/old dotf → ERROR, `--deploy` fails
- [x] [AC3] `setup-windows.ps1` calls the verb after `Deploy-SkillRecord`; `tests/setup-windows.bats` ordering + parity guard; `tests/compile-harness.bats` stub learns the verb (argv recorded, `--render` simulated), four scenario tests replaced by three contract tests
- [x] [P] [AC4] Failing test `TestCheckAgentPresence_ByStatus` (PASS / WARN missing / WARN stale / copilot gate / SKIP)
- [x] [AC4] `checkAgentPresence` registered after `checkDeployDrift`
- [x] [AC5] Box: 0 regions before → 4 injected → second run current → doctor section green → instruction-drift check still PASS (region stripped)
- [x] Mutation checks: autonomous listed / append-not-replace / sha depends on EOL / doctor calls stale current — each red, restored

## Verification

- [x] Go loop: build, vet (host, `GOOS=linux`), test, golangci-lint
- [x] bats: compile-harness, setup-windows; `bash -n`; `setup-windows.ps1` parse 0 errors, ASCII delta 0, CRLF intact
- [x] `verification.md` records the evidence; `features.json` per AC
