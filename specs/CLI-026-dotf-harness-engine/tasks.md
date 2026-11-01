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
- [x] `doctor`'s agent-tier check reads targets and tier with the Go render's own predicates (`harness.SkillTargetsAgent`, `harness.RecordModelTier`); `recordTargets`, `readAgentFrontmatter` and the two awk-agreement tests are gone, because the shell render they followed no longer deploys

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
- [x] `ci.yml` and `dotf doctor`'s harness drift check call it on every OS (0.77.0 carries the verb)
  - [x] `dotf doctor` runs `harness.CheckDrift` in process on every OS (`checkRecordDrift`); its remedies name `dotf harness deploy` and `dotf harness refresh`
  - [x] `ci.yml` (#2923, a `.github/**` change: prepared here, landed by the owner)
- [x] `--against-vault` with a regression test (#1545): `harness.CompareWithVault`, shared with doctor's WARN; `TestHarnessCheck_AgainstVaultFlagsAnUnrefreshedVaultEdit`

## Slice 7 — `refresh` in Go and deletion (AC7, AC9)

> Split 2026-11-01 into three PRs, because the render, the vault compare and the deletion are three reviews:
> 7a renders in Go with the script kept as its oracle, 7b makes `--against-vault` compare renders, 7c deletes.
> 7a is over the executable-line cap with no functional seam (declared on its PR). 7c touches `ci.yml` and the
> harness schemas, so the owner lands it.

- [x] Render half of `--refresh` in Go behind the existing gate (`harness.Refresh`, `tests/harness-refresh-parity.bats`); every check runs before the first write, and the dirty-paths guard reads the manifest's targets (lesson 529)
  - [x] With the render in Go, `--against-vault` also renders each vault source and compares it with the committed record, so a record edited without its vault source is caught before a refresh reverts it (the other half of #1545; the stamp compare sees only the vault side)
    - [x] Skill and agent records whose sha matches are rendered from the source dir and compared file by file
      (`VaultDrift.Edited`, `[EDITED]`); a file the vault does not have, or one missing, is an edit too
    - [x] Enforced records, which carry no stamp, are held to the vault section they are cut from; a section
      that is gone is stale
    - [x] Line endings compare as LF, so a CRLF vault checkout is not an edit
    - [x] doctor's vault check warns on an edited record and names the remedy (carry the edit to the vault)
    - [x] Real vault at origin/master: 50 records level; a one-line edit to `skills/helm` and to an enforced
      record each fails the check
- [ ] `git rm scripts/compile-harness.sh tests/compile-harness*.bats tests/harness-skills-parity.bats`; guard-grep clean
  - [ ] The five `harness-*-parity.bats` oracles go with the script; `harness-doctrine-records.bats` and `harness-refresh-announce.bats` move to `dotf harness refresh --vault`
  - [ ] Before closing #2884 with it, each of its three cases has a Go-side test, or gets one here
  - [ ] Close #1733 with it: the shell render is the last reader of `targets:` by substring, and the deploy and doctor already use `harness.SkillTargetsAgent`

## Closing

- [ ] Every acceptance criterion is covered by at least one test
- [ ] `go build`, `go vet`, `go test`, `golangci-lint`, `GOOS=windows go vet` pass
- [ ] `verification.md` filled in
- [ ] Independent review (`dotf spec review CLI-026-dotf-harness-engine`) before archive
