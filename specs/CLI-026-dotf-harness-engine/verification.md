---
tags: [spec, verification, templates]
created: "2026-06-21"
---

# Verification - CLI-026-dotf-harness-engine

## Evidence

- [x] AC1 (skills block, slice 1) -> `tests/harness-skills-parity.bats` (5 tests): every output but the prompts byte-identical to `compile-harness.sh --deploy`; prompts differ only by the body rules the shell dropped; a prompt keeps every body rule; executable bits match; a second Go run reports `0 deployed`
- [x] AC2 (skills) -> `TestRenderSkillRecord_CRLFRendersTheLFBytes`
- [x] AC3 copilot catalog (slice 2) -> `tests/harness-skills-parity.bats` "the catalog lists what the oracle lists, in byte order" over a seeded instructions file on both sides; `TestDeploySkills_WritesTheCatalogIntoItsRegion` pins the exact bytes and sha, `TestDeploySkills_SkipsAnAbsentOrUnmarkedCatalogFile`, `TestReplaceCatalogRegion_RefusesBadMarkers`, `TestCatalogBullets_ByteOrder`
- [ ] AC4 agents, doctrine, deploy -> slices 4–5
  - [x] agents -> `tests/harness-agents-parity.bats`: every agent file byte-identical to `compile-harness.sh --deploy` with the binary under test as the shell's resolver; a second run reports `0 deployed`. `TestDeployAgents_*` cover the resolved lines, a bad record failing alone with its previous output kept, the block-capabilities refusal, no-tier records and dry run
  - [x] doctrine -> `tests/harness-doctrine-parity.bats`: both rules files byte-identical to `compile-harness.sh --deploy` from three starting states (absent, a region among user lines with the legacy preamble, no region and no final newline); a second Go run reports both current. `TestCompactDoctrine_*`, `TestFoldToASCII`, `TestInjectDoctrine`, `TestMigratePreamble_ExactLineOnly`, `TestCapWarnings_NameWhoseContentBreached`, `TestDeployDoctrine_*`
  - [ ] deploy
- [ ] AC5 check -> slice 6
- [ ] AC6 `--against-vault` -> slice 6
- [ ] AC7 refresh in Go -> slice 7
- [ ] AC8 callers rewired -> slices 3 and 5
- [ ] AC9 bash engine deleted -> slice 7

## Test status

- Slice 1, 2026-10-20, macOS arm64: `go test ./internal/harness/ -run 'Skill|LoadSkills'` ok; `DOTF_BIN=… bats tests/harness-skills-parity.bats` 5/5 ok.
- Slice 2, 2026-10-21, macOS arm64: `go test ./internal/harness/ ./internal/cmd/` ok; parity bats 6/6 ok under bash and zsh; `golangci-lint` (pinned 2.12.2) 0 issues; `GOOS=windows go vet` clean.
- Slice 4 (agents), 2026-10-21, macOS arm64: `go test ./internal/harness/ ./internal/cmd/` ok; `tests/harness-agents-parity.bats` 2/2 under bash and zsh; `golangci-lint` 0 issues; `GOOS=windows go vet` clean. Real records: 7 deployed, then `0 deployed, 7 current`.
- Slice 4 (doctrine), 2026-10-21, macOS arm64: `go test ./internal/harness/ ./internal/cmd/` ok; `tests/harness-doctrine-parity.bats` 4/4; `golangci-lint` 0 issues; `GOOS=windows go vet` clean.
- Mutation check, slice 4 (doctrine): changing the payload heading fails 3 of the 4 oracle tests.
- Mutation check, slice 4 (agents): keeping `kind:` in the render fails the oracle's byte diff.
- Mutation check, slice 2: accepting an END line with a tail fails `TestReplaceCatalogRegion_RefusesBadMarkers/END_with_a_tail`; hashing anything but the bullets fails `TestDeploySkills_WritesTheCatalogIntoItsRegion`.
- Mutation check, slice 1: restoring the shell's "drop every `---`" in the prompt render fails `TestRenderSkillRecord`; restoring substring `targets:` matching fails `TestSkillTargetsAgent/inline_list_names_copilot,_not_pi_(#1733)`.
- Manual oracle run, slice 1: 35 records, 240 files on each side with copilot and pi on PATH; `diff -r -x prompts` empty, file modes identical, rerun `0 deployed, 204 current, 0 pruned`.

## Decisions made during implementation

- Slice 1: the copilot catalog moved to its own slice to keep slice 1 under the executable-line cap (about 293 lines).
- Slice 1: a record whose frontmatter is not valid YAML is now an error for that record (reported after the rest deploy); the shell never parsed it. Every committed record parses (`TestSkillTargetsAgent_EveryCommittedRecordParses`).
- Slice 1: an explicit empty or null `targets:` deploys to nobody, as the shell's substring match did.
- Slice 4 (agents): the model and capability maps are read in process, so the shell's "dotf absent or too old" degrade (a render with no `model:` line and a warning) has no Go form: a tier that cannot be resolved is an error for that record. The shell's whitespace-in-model-id refusal is kept.
- Slice 4 (doctrine): the roster comes from `harness.RenderPresence` in process, so the shell's "dotf absent, payload carries no roster" degrade has no Go form. The warnings (shadowing file, non-ASCII left after the fold, breached cap in characters or bytes) are returned per file and printed on stderr, in the shell's words.
- Slice 4 (agents): the persona roster stays with `dotf harness presence`; `deploy` composes the two (slice 5).
- Slice 2: the catalog's record order differs from the oracle's. The shell's `*/` glob sorted `test-driven-development/` before `test/` even under `LC_ALL=C` (the `/` is in the sort key); Go uses byte order of the names, as declared. The oracle compares the bullets as a set (lesson 505), so the cutover rewrites the deployed catalog once, in a new order.
- Slice 2: an END line carrying anything after the marker is refused, by every GENERATED-region writer (the catalog skips the file as unmarked; doctrine fails that target, `TestInjectDoctrine_RefusesAnENDLineWithATail`). The shell counted it as an END for validation, then never matched it exactly and dropped the rest of the file.
- Slice 2: an absent or unmarked catalog file is skipped with a warning and never created, as in the shell; `dotf harness instructions` owns the file and its slot.
- Slice 1: the mode comparison goes through `fsmode.Needs`. Windows reports 0666 for any writable file, so a full comparison rewrote every output on every run there; caught by `TestDeploySkills_FreshThenConverged` on the Windows CI leg.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? yes: `docs/lessons/lesson-504-a-fence-counter-that-never-stops-counting-eats-the-body.md`, `docs/lessons/lesson-505-a-directory-glob-sorts-by-the-slash-not-by-the-name.md`
- [ ] ADR-worthy decision? <to answer at archive>
- [ ] New pattern candidate for `00_meta/patterns/`? <to answer at archive>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-026-dotf-harness-engine/` -> `specs/archive/CLI-026-dotf-harness-engine/`
- [ ] Promotions above executed (if any)
