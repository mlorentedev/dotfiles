---
tags: [spec, tasks]
created: "2026-08-27"
---

# Tasks - CLI-050-knowledge-crystallize-cutover

## Setup

- [x] Branch created from main: `feat/cli-050-crystallize-cutover`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC2] Repoint the SessionStart hook message (`cli/internal/mem/session_start_injectors.go`
      + its test) at `dotf vault crystallize`
- [x] [AC2] Repoint `scripts/vault-maintenance-weekly.sh` / `.ps1` at `dotf vault crystallize --all`
- [x] [AC1] [AC2] Remove `setup-linux.sh`'s chmod and `setup-windows.ps1`'s deploy block for the
      twin
- [x] [AC2] Repoint docs/comments: `.claude/CLAUDE.md`, `docs/runbooks/guide-knowledge-distillation.md`,
      `scripts/vault.sh`, `scripts/utils.ps1`, `.github/workflows/ci.yml` (drop the retired
      `-ps1.bats` from the PowerShell CI list), `docs/adr/adr-008-skills-ecosystem-overhaul.md`,
      `cli/internal/cmd/{vault,mem}.go` doc comments
- [x] [AC5] Verify equivalent Go-native coverage exists for each bats file before deleting it
      (`crystallize_test.go`'s `TestIsYAMLBlockScalar` + `TestProcessProjectRefusesYAMLAndLeavesFileIntact`
      cover what `-yaml-guard.bats` asserted; the go-parity suite's golden cases cover the rest)
- [x] [AC1] [AC3] Delete `scripts/knowledge-crystallize.{sh,ps1}`, the four bats files that
      invoked them directly or re-verified against them (`knowledge-crystallize.bats`,
      `-golden.bats`, `-ps1.bats`, `-yaml-guard.bats`), and `tests/golden/crystallize/capture.sh`
      (can no longer run — its oracle is gone)
- [x] Update `tests/golden/crystallize/{lib.sh,ORACLE}` and
      `tests/knowledge-crystallize-go-parity.bats` headers to describe the goldens as a frozen
      characterization contract, not a live shell comparison; drop the now-dead `shell` mode
      branch and `GC_ORACLE_SH` plumbing from `lib.sh`

## Post-review fixes

- [x] Harden `vault-maintenance-weekly.{sh,ps1}`'s PATH before calling bare `dotf` (cron/Task
      Scheduler don't reliably inherit `~/.local/bin`) — reviewed Major, REAL
- [x] Fix `features.json` f2's verification command to exclude the unrelated active spec
      `specs/HARNESS-063-...` it was falsely flagging — reviewed Minor, REAL
- [x] File TEST-002 (#1277) for the pre-existing `.ps1` test-coverage gap rather than build
      unverified Pester tests — reviewed Minor, THEORETICAL

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous
      verification command
- [x] Type checks pass (`go vet`, `GOOS=windows go vet`)
- [x] Lint passes (`golangci-lint`, `shellcheck`)
- [x] No unrelated changes in the diff (no scope creep) — two tangential doc-staleness findings
      ticketed as DOCS-014 (#1271) instead of fixed here
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder
