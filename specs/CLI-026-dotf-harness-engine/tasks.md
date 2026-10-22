---
tags: [spec, tasks, templates]
created: "2026-06-21"
---

# Tasks - CLI-026-dotf-harness-engine

> Re-planned 2026-10-20 against `origin/main` `856f49fc`. One slice = one PR under the ~300 executable-line cap. TDD order inside each slice.

## Setup

- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] Open questions resolved (line endings, cutover, sequencing 2026-06-21; verb shape, oracle, declared divergences 2026-10-20)
- [x] Branch: `chore/spec-triage-cli-026` (slice 1)
- [x] `dotf harness` noun wired in `cli/internal/cmd/harness.go` (landed before this re-plan)

## Slice 1 — `dotf harness skills` (AC1, AC2)

- [x] Parity oracle: `tests/harness-skills-parity.bats` deploys with bash and Go into two HOMEs over the committed records and diffs them
- [x] Tests: render per kind, awk line semantics, CRLF → LF bytes, `targets:` forms, deploy/converge/dry-run/symlink/prune/unmanaged/bad record
- [x] `cli/internal/harness/skills.go` (manifest, targets, render) and `skills_deploy.go` (deploy, aux files, prune)
- [x] `cli/internal/cmd/harness_skills.go`
- [x] Lesson 504: the prompt render's fence counter

## Slice 2 — the copilot skill catalog (AC3)

- [x] Failing test: catalog region rendered and replaced in place, sha over the bullets, byte-order record walk
- [x] Implement in `skills` (`skills_catalog.go`, the `regions.go` marker constants); extend the oracle with a seeded `.copilot/copilot-instructions.md` on both sides
- [x] Lesson 505: a directory glob sorts by the slash

## Slice 3 — rewire skills (part of AC8; needs a release carrying slices 1–2)

- [x] ~~`deploy_skills`/`deploy_prune`/`build_skill_catalog` in `compile-harness.sh` delegate to `dotf harness skills`~~ superseded: no caller runs `compile-harness.sh --deploy` once the slice 5 rewire lands, and slice 7 deletes it
- [x] `setup-windows.ps1` calls `dotf harness deploy` (slice 5), which runs the skills surface; `Deploy-SkillRecord`, `Test-SkillTargetsAgent`, `Get-SkillField`, `Test-SkillFrontmatterKeyKept`, `Convert-SkillRecord` and their bats parity tests deleted; the copilot smoke test runs `dotf harness skills`
- [ ] Note the skills half fixed on #1733
- [ ] `doctor`'s `recordTargets` (`checks_agent_tiers.go`) mirrors the shell's substring match on purpose (`TestRecordTargetsAgreesWithTheRender`); switch it and its test to `harness.SkillTargetsAgent` in the slice that moves the agent render to Go (slice 4), so the check keeps predicting the render

## Slice 4 — agents and doctrine (AC4)

- [x] `dotf harness agents`: agent-md render with resolved model and capability lines; parity oracle (`tests/harness-agents-parity.bats`)
- [x] `dotf harness doctrine`: compact payload, ASCII fold, char cap, shadow warning; parity oracle (`tests/harness-doctrine-parity.bats`)
- [x] Windows gains `agents.deploy` (#1387), through `dotf harness deploy` in `setup-windows.ps1`; live check pending (slice 5)

## Slice 5 — `dotf harness deploy` (AC4, AC8)

- [x] Compose instructions → skills → agents → presence → doctrine, failure collected like the shell (`dotf harness deploy`, `tests/harness-deploy-parity.bats`)
- [x] Rewire `setup-linux.sh`, `setup-windows.ps1` and `converge/harness.go` (`records-harness`) to it
  - [x] `records-harness` calls `harness.Deploy` in process (`converge.HarnessDeploy`); `CompileHarnessDeploy` deleted
  - [x] `setup-linux.sh` calls `dotf harness deploy` (supersedes slice 3's shell delegation line)
  - [x] `setup-windows.ps1` calls `dotf harness deploy`; `Deploy-SkillRecord` and its helpers deleted; `records-harness` gains `windows`
  - [ ] Live acceptance on the work-lab Windows box after a release carries it: setup, then `dotf converge` reports `records-harness` changed=0, and `~/.claude/agents/` holds the seven personas (#1387)
  - [x] doctor's `instructionsRemedy` (`checks_deploy.go`) is gone: every OS is told `run: dotf harness deploy`

## Slice 6 — `dotf harness check` (AC5, AC6)

- [x] Offline drift check in Go (`dotf harness check`, `tests/harness-check-parity.bats`)
- [ ] `ci.yml` and `dotf doctor`'s harness drift check call it on every OS (0.77.0 carries the verb)
  - [x] `dotf doctor` runs `harness.CheckDrift` in process on every OS (`checkRecordDrift`); its remedies name `dotf harness deploy` and `dotf harness refresh`
  - [ ] `ci.yml` (a `.github/**` change: prepared here, landed by the owner)
- [ ] `--against-vault` with a regression test (#1545)

## Slice 7 — `refresh` in Go and deletion (AC7, AC9)

- [ ] Render half of `--refresh` in Go behind the existing gate
- [ ] `git rm scripts/compile-harness.sh tests/compile-harness*.bats tests/harness-skills-parity.bats`; guard-grep clean

## Closing

- [ ] Every acceptance criterion is covered by at least one test
- [ ] `go build`, `go vet`, `go test`, `golangci-lint`, `GOOS=windows go vet` pass
- [ ] `verification.md` filled in
- [ ] Independent review (`dotf spec review CLI-026-dotf-harness-engine`) before archive
