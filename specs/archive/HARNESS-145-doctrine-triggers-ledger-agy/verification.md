---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - HARNESS-145-doctrine-triggers-ledger-agy

## Evidence

- [x] AC1 -> S1: bats `doctrine: the committed records keep the capped agy surface under 8000 characters`
- [x] AC2 -> S2: bats `triggers: --refresh fails naming the trigger and pattern, and writes nothing`, `TestCheckTriggerTargets_DanglingTriggerFailsAndIsNamed`
- [x] AC3 -> S3: `TestSessionsWithNoIDShareNoLedger` (fails on the old code)
- [x] AC4 -> S4: `TestAgyAnswersInItsOwnProtocol`, `TestBindEmitsTheAgyGateIntoHooksJSONAndRetiresTheOldEntry`, `TestBindKeyHoldsOnlyFormatsAnOlderBinaryUnderstands`
- [x] AC5 -> operational tail: an `agy` decision record with a real conversation id

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
- `bats tests/*.bats`: 1597 run, 1 failure, the environmental oh-my-zsh snapshot (#1641), which is also red on plain main on this machine. That count predates S2 on this branch: the merged tree holds 1,599 tests, and the independent review's run there gave 1,599 run, 1 failure, the same snapshot.
- AC4: `TestAgyAnswersInItsOwnProtocol`, `TestBindEmitsTheAgyGateIntoHooksJSONAndRetiresTheOldEntry`, `TestBindKeyHoldsOnlyFormatsAnOlderBinaryUnderstands` and `TestLoadBindTargetsRefusesAFormatInTheWrongKey` pass.
- The manifest and binary skew was measured, not assumed: dotf 0.57.0, given the agy target under `agents.bind`, wrote a top-level `hooks` key with a `_managed` sidecar into a copy of the real `hooks.json`. Under `agents.bind_named` the same binary emits nothing.
- The agy payload is documentation-derived (workspace hooks never loaded headless), so AC5 stays open until a real call after `dotf harness bind`.
- Executable production lines: +245 / -28 across `harness_bind.go`, `harness_gate.go`, `bind.go` and `bind_manifest.go` (comments and blank lines excluded).

## Operational tail evidence (AC5)

Run on 2026-09-25 after all four slices and release 0.58.0 merged (`e1c5754`), from a detached worktree at that commit, with both dotfiles peer sessions told before and after. The full record is on #1682.

- `scripts/install-dotf.sh`: 0.57.0 converged to 0.58.0 (checksum-verified, atomic rename).
- `dotf harness mirror`: 9 updated, 67 unchanged; the deployed `triggers.json` and `manifest.json` are byte-identical to main. `compile-harness.sh --deploy`: exit 0, `GEMINI.md` 7,798 characters and 7,798 bytes.
- `dotf harness bind --harness agy`, rehearsed first on copies of the three files: Orca's `orca-status` hook value-identical (the file is re-rendered with sorted keys, so its bytes change and its JSON value does not), ours added (`PreToolUse`, matcher `*`), only our `BeforeTool` handler retired from `~/.gemini/settings.json`, modes kept. The live files equal the rehearsal, and a second run reports nothing to do.
- f5 is a human-run operational check, not a reproducible one: it reads this machine's gate journal, so it exits 1 anywhere else and cannot go red once green. The record's shape is pinned in-process by `TestAgyAnswersInItsOwnProtocol`.
- AC5: one headless `agy -p` call read `/etc/hostname` (exit 0) and left `{"harness":"agy","session":"8917b453-b620-4c9d-9d0e-ac1c21991bd9","tool":"view_file","outcome":"no-role","allowed":true}`.
- The first f5 verifier also accepted two synthetic records (`test-1`, `test-2`) that an earlier scratch-binary check wrote into the real journal, so it would have passed before the call. It now requires a UUID session; it exits 0 on the record above and 1 on four negative controls (synthetic session, no session, `payload-unrecognised`, another harness).
- `dotf doctor` against main: no trigger, instruction-drift, doctrine or version FAILs. What remains is outside this spec: the zombie specs (#1626), and deploy-dir drift in files only setup copies (`scripts/compile-harness.sh`, `versions.conf`, two `sensitive/dr` files).
- The Claude half of a full `dotf harness bind`, which adds a `_managed` marker to four existing handlers with no command change, was applied afterwards on Manu's word. A fresh headless Claude session loaded the marked file and its gate recorded the call.

## Independent review

`review.md`: PASS-WITH-GAPS from nan/deepseek-v4-flash on 2026-09-25, reviewed against `4e12d45` with base `5abc54a`. It asked for no contract-set edit. Each finding, dispositioned with Manu:

| Finding | Disposition |
|---|---|
| 1. `--refresh` reports every trigger pattern present when `jq` could not read the file, or it names none | ticketed: HARNESS-148 (#1695) |
| 2. `dotf doctor` says OK for zero triggers instead of SKIP | ticketed: HARNESS-148 (#1695) |
| 3. `_unparsed` is claimable as a session id, unlike `_unscoped` | ticketed: HARNESS-149 (#1696) |
| 4. The bind keeps an existing file mode on every target, not only agy's | kept, and recorded here: every target is a document another tool also writes (Claude Code and Orca write `~/.claude/settings.json`, Orca writes agy's `hooks.json`), so re-permissioning one is not the bind's call. The missing Claude-target test is in HARNESS-149 (#1696) |
| 5. `verification.md` overstated two claims | applied above: "value-identical" for Orca's hook; the S4 bats count predates S2 on the branch |
| 6. f5 is machine-bound | declared above as a human-run operational check |
| Rubric C: the gate's `RunE` closure grew from cyclomatic complexity 14 to 20 | ticketed: HARNESS-150 (#1697) |

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

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/HARNESS-145-doctrine-triggers-ledger-agy/` -> `specs/archive/HARNESS-145-doctrine-triggers-ledger-agy/`
- [x] Bitácora board ticket closed by the closing PR: the archive PR carries `Closes #1682`, and the board moves it to Done on merge (ADR-018)
- [x] Promotions above executed: lessons 292 and 293, the ADR-027 amendment and the ADR-010 row, all in S4 (#1688)
