---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - HARNESS-145-doctrine-triggers-ledger-agy

## Evidence

- [ ] AC1 -> S1: bats `doctrine: the committed records keep the capped agy surface under 8000 characters`
- [ ] AC2 -> S2: bats `triggers: --refresh fails naming the trigger and pattern, and writes nothing`, `TestCheckTriggerTargets_DanglingTriggerFailsAndIsNamed`
- [ ] AC3 -> S3: `TestSessionsWithNoIDShareNoLedger` (fails on the old code)
- [ ] AC4 -> S4: `TestAgyAnswersInItsOwnProtocol`, `TestBindEmitsTheAgyGateIntoHooksJSONAndRetiresTheOldEntry`, `TestBindKeyHoldsOnlyFormatsAnOlderBinaryUnderstands`
- [ ] AC5 -> operational tail: an `agy` decision record with a real conversation id

Each slice PR fills its own section below and nothing else.

## S1 evidence

- `go build`, `go vet` (linux and windows), `go test -count=1 ./...` (25 packages), `golangci-lint`, `shellcheck --severity=error`, `compile-harness.sh --check`, `check-bats-names.sh` and `check-doc-paths.sh`: all pass.
- `bats tests/*.bats`, serial: 1597 run, 1 failure, the environmental oh-my-zsh snapshot (#1641), which is also red on plain main on this machine.
- AC1: `doctrine: the committed records keep the capped agy surface under 8000 characters` passes. The deployed `~/.gemini/GEMINI.md` is 7,798 characters and 7,798 bytes, down from 11,977 and 11,986.
- Mutation: with `migrate_legacy_preamble` disabled, `doctrine: the old em-dash preamble is migrated ...` fails; restored, it passes.
- `guard-bats-negation` caught a bare `!` assertion in this slice's first version of that test; it now uses `refute_grep_fixed`.
- Executable production lines: `scripts/compile-harness.sh` +48 / -15 (comments and blank lines excluded).

## S2 evidence

- `go build`, `go vet` (linux and windows), `go test -count=1 ./...` (25 packages), `golangci-lint`, `shellcheck --severity=error`, `compile-harness.sh --check`, `check-bats-names.sh` and `check-doc-paths.sh`: all pass, on top of the merged S1.
- `bats tests/*.bats`: 1599 run, 1 failure, the environmental oh-my-zsh snapshot (#1641), which is also red on plain main on this machine.
- AC2: `triggers: --refresh fails naming the trigger and pattern, and writes nothing` and `TestCheckTriggerTargets_DanglingTriggerFailsAndIsNamed` pass. `TestSuggestSharedPatternDoesNotCrossLinkSkills` and `TestMatchPathsNeverReportsAnEmptyPattern` pin the two linking defects, and `TestCheckTriggerTargets_NoVaultSkipsRatherThanPasses` keeps the doctor check honest on a machine without the vault.
- Expected transitional effect: on a machine whose deploy dir has not mirrored the repaired `harness/triggers.json`, the new doctor check FAILs until `dotf harness mirror` runs. That is the check doing its job.
- Executable production lines: +87 / -10 across `checks_trigger_targets.go`, `checks_deploy.go`, `triggers.go` and `compile-harness.sh` (comments and blank lines excluded).

## S3 evidence

- `go build`, `go vet` (linux and windows), `go test -count=1 ./...` (25 packages), `golangci-lint`, `shellcheck --severity=error`, `compile-harness.sh --check`, `check-bats-names.sh` and `check-doc-paths.sh`: all pass.
- `bats tests/*.bats`: 1597 run, 1 failure, the environmental oh-my-zsh snapshot (#1641), which is also red on plain main on this machine.
- AC3: `TestSessionsWithNoIDShareNoLedger` drives two payload shapes end to end (a `--role` override and an agent-id-only payload) and fails on the old code; restoring the digest-of-nothing key fails both subtests.
- `TestAnEmptyScopeHasNoStateFile`, `TestDecisionPathNeverFallsBackToTheDigestOfNothing` and `TestConsumptionScopeNeedsASession` pin the pieces the end-to-end test stands on.
- `TestTheReservedJournalNameIsNotASession` (two payload shapes) fails on the previous head, where a payload claiming `_unscoped` was gated as a real session, and passes with the reserved name read as no session; `TestConsumptionScopeNeedsASession` gained the two matching cases.
- Executable production lines: +45 / -7 across `gate.go`, `dispatch.go`, `decision.go` and `harness_gate.go` (comments and blank lines excluded).

## S4 evidence

- `go build`, `go vet` (linux and windows), `go test -count=1 ./...` (25 packages), `golangci-lint`, `shellcheck --severity=error`, `compile-harness.sh --check`, `check-bats-names.sh` and `check-doc-paths.sh`: all pass, on top of the merged S3.
- `bats tests/*.bats`: 1597 run, 1 failure, the environmental oh-my-zsh snapshot (#1641), which is also red on plain main on this machine.
- AC4: `TestAgyAnswersInItsOwnProtocol`, `TestBindEmitsTheAgyGateIntoHooksJSONAndRetiresTheOldEntry`, `TestBindKeyHoldsOnlyFormatsAnOlderBinaryUnderstands` and `TestLoadBindTargetsRefusesAFormatInTheWrongKey` pass.
- The manifest and binary skew was measured, not assumed: dotf 0.57.0, given the agy target under `agents.bind`, wrote a top-level `hooks` key with a `_managed` sidecar into a copy of the real `hooks.json`. Under `agents.bind_named` the same binary emits nothing.
- The agy payload is documentation-derived (workspace hooks never loaded headless), so AC5 stays open until a real call after `dotf harness bind`.
- Executable production lines: +245 / -28 across `harness_bind.go`, `harness_gate.go`, `bind.go` and `bind_manifest.go` (comments and blank lines excluded).

## Decisions made during implementation

- The slim records are derived from the vault section (the SSOT); the original wording stays beside the dense text in the vault so nothing is lost. The cost is duplicated rules, accepted and recorded as a risk in `proposal.md`.
- A format an older binary cannot emit lives under a manifest key that binary never reads (`agents.bind_named`), because an older binary treats every target in `agents.bind` as Claude's shape.
- An empty session key means "no consumption or dispatch state": the call is allowed, and journaled as `session-unscoped` when it would have enforced a persona or recorded a skill consumption, rather than mapped to a shared bucket. The journal still stores the record, under one reserved name that a real session id cannot take.
- agy answers `ask`, never `allow`: `allow` approves without asking, which would make measurement plumbing a permission grant.
- The eight dangling triggers were kept and remapped, not pruned: the rules route prompts to a persona.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`: yes, lessons 292 and 293 (S4)
- [x] ADR-worthy decision for the repo's `docs/adr/`: amendment to ADR-027 and the ADR-010 hooks row (S4)
- [ ] New pattern candidate for `00_meta/patterns/`: no

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-145-doctrine-triggers-ledger-agy/` -> `specs/archive/HARNESS-145-doctrine-triggers-ledger-agy/`
- [ ] Bitácora board ticket moved to Done / closed with the closing PR (ADR-018)
- [ ] Promotions above executed
