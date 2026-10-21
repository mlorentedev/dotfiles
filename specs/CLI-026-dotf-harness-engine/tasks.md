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

- [ ] `deploy_skills`/`deploy_prune`/`build_skill_catalog` in `compile-harness.sh` delegate to `dotf harness skills`
- [ ] `setup-windows.ps1` calls `dotf harness skills`; delete `Deploy-SkillRecord`, `Test-SkillTargetsAgent`, `Get-SkillField`, `Test-SkillFrontmatterKeyKept`, `Convert-SkillRecord` and their Pester/bats parity tests
- [ ] Note the skills half fixed on #1733
- [ ] `doctor`'s `recordTargets` (`checks_agent_tiers.go`) mirrors the shell's substring match on purpose (`TestRecordTargetsAgreesWithTheRender`); switch it and its test to `harness.SkillTargetsAgent` in the slice that moves the agent render to Go (slice 4), so the check keeps predicting the render

## Slice 4 — agents and doctrine (AC4)

- [x] `dotf harness agents`: agent-md render with resolved model and capability lines; parity oracle (`tests/harness-agents-parity.bats`)
- [ ] `dotf harness doctrine`: compact payload, ASCII fold, char cap, shadow warning; parity oracle
- [ ] Windows gains `agents.deploy` (#1387)

## Slice 5 — `dotf harness deploy` (AC4, AC8)

- [ ] Compose instructions → skills → agents → presence → doctrine, failure collected like the shell
- [ ] Rewire `setup-linux.sh`, `setup-windows.ps1` and `converge/harness.go` (`records-harness`) to it

## Slice 6 — `dotf harness check` (AC5, AC6)

- [ ] Offline drift check in Go; `ci.yml` and `dotf doctor`'s harness drift check call it on every OS
- [ ] `--against-vault` with a regression test (#1545)

## Slice 7 — `refresh` in Go and deletion (AC7, AC9)

- [ ] Render half of `--refresh` in Go behind the existing gate
- [ ] `git rm scripts/compile-harness.sh tests/compile-harness*.bats tests/harness-skills-parity.bats`; guard-grep clean

## Closing

- [ ] Every acceptance criterion is covered by at least one test
- [ ] `go build`, `go vet`, `go test`, `golangci-lint`, `GOOS=windows go vet` pass
- [ ] `verification.md` filled in
- [ ] Independent review (`dotf spec review CLI-026-dotf-harness-engine`) before archive
