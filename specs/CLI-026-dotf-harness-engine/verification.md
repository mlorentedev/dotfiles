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
  - [x] deploy -> `tests/harness-deploy-parity.bats`: the full committed manifest deployed by `compile-harness.sh --deploy` and `dotf harness deploy` into two HOMEs, byte-identical but for agy's prompts and the copilot file, which differs only by the catalog's bullet order; a second Go deploy writes nothing. `TestHarnessDeploy_*` cover convergence and a failed agent record stopping presence and doctrine
- [ ] AC5 check -> slice 6
  - [x] the Go verb -> `tests/harness-check-parity.bats` (5 tests): the same exit status and the same report lines as `compile-harness.sh --check` on the committed checkout and on each kind of drift (an edited, a missing and a doubly-marked region, an END line with a tail, a coverage gap, skill and agent records without their required frontmatter). `TestCheckDrift_*` (including a CRLF record that validates), `TestFirstDifference`
  - [ ] `ci.yml` and `dotf doctor` call it on every OS -> after a release carries the verb
- [ ] AC6 `--against-vault` -> slice 6
- [ ] AC7 refresh in Go -> slice 7
- [ ] AC8 callers rewired -> slices 3 and 5
- [ ] AC9 bash engine deleted -> slice 7

## Test status

- Slice 1, 2026-10-20, macOS arm64: `go test ./internal/harness/ -run 'Skill|LoadSkills'` ok; `DOTF_BIN=… bats tests/harness-skills-parity.bats` 5/5 ok.
- Slice 2, 2026-10-21, macOS arm64: `go test ./internal/harness/ ./internal/cmd/` ok; parity bats 6/6 ok under bash and zsh; `golangci-lint` (pinned 2.12.2) 0 issues; `GOOS=windows go vet` clean.
- Slice 4 (agents), 2026-10-21, macOS arm64: `go test ./internal/harness/ ./internal/cmd/` ok; `tests/harness-agents-parity.bats` 2/2 under bash and zsh; `golangci-lint` 0 issues; `GOOS=windows go vet` clean. Real records: 7 deployed, then `0 deployed, 7 current`.
- Slice 4 (doctrine), 2026-10-21, macOS arm64: `go test ./internal/harness/ ./internal/cmd/` ok; `tests/harness-doctrine-parity.bats` 4/4; `golangci-lint` 0 issues; `GOOS=windows go vet` clean.
- Slice 5 (deploy, Go half), 2026-10-21, macOS arm64: `go test ./internal/harness/ ./internal/cmd/` ok; `tests/harness-deploy-parity.bats` 3/3 under bash and zsh; `golangci-lint` 0 issues; `GOOS=windows go vet` clean. Real checkout into an empty HOME: 4 instruction files, catalog, 204 skill outputs, 7 agents, 4 presence regions, 2 doctrine files, `[deploy] OK`.
- Slice 6 (check, Go verb), 2026-10-21, macOS arm64: `go test ./internal/harness/ ./internal/cmd/` ok; `tests/harness-check-parity.bats` 5/5 under bash and zsh; `golangci-lint` 0 issues after a cache clean; `GOOS=windows go vet` clean. Real checkout: both engines exit 0 with the same 47 OK lines.
- Mutation check, slice 6: keeping the full-only marker lines fails 4 of the 5 oracle tests; closing the region on an END line with a tail fails 1; treating every id as opted out fails all 5; accepting a record without `---` on line 1 fails 1.
- Mutation check, slice 5: running skills before instructions fails 2 of the 3 oracle tests (the catalog finds no instructions file to write into on a fresh HOME); leaving presence out fails 2 of the 3.
- Mutation check, slice 4 (doctrine): changing the payload heading fails 3 of the 4 oracle tests.
- Mutation check, slice 4 (agents): keeping `kind:` in the render fails the oracle's byte diff.
- Mutation check, slice 2: accepting an END line with a tail fails `TestReplaceCatalogRegion_RefusesBadMarkers/END_with_a_tail`; hashing anything but the bullets fails `TestDeploySkills_WritesTheCatalogIntoItsRegion`.
- Mutation check, slice 1: restoring the shell's "drop every `---`" in the prompt render fails `TestRenderSkillRecord`; restoring substring `targets:` matching fails `TestSkillTargetsAgent/inline_list_names_copilot,_not_pi_(#1733)`.
- Manual oracle run, slice 1: 35 records, 240 files on each side with copilot and pi on PATH; `diff -r -x prompts` empty, file modes identical, rerun `0 deployed, 204 current, 0 pruned`.

- Slice 5 rewire, converge: `TestRecordsHarness_TheGoDeployConvergesTheRealCheckout` applies `records-harness` with `HarnessDeploy` on this checkout into an empty home; the probe then passes and a second plan reports 0 changes.
- Mutation check, slice 5 rewire: making `HarnessDeploy` return nil without deploying fails that test.
- `tests/harness-deploy-parity.bats` 3/3 after the composition moved into `harness.Deploy`: the verb's output did not change.

- Slice 5 rewire, Linux setup: `setup-linux.sh` calls `dotf harness deploy --repo-root "$CURRENT_DIR"`; the script went from 722 to 716 lines.
- `tests/setup-fails-loudly.bats` now finds the block by the `dotf harness deploy` line and still requires `exit 1` and no `log_warning` in it; the `compile-harness.bats` test that grepped for the script call is deleted with the call.
- Every bats file that references `setup-linux.sh` or `compile-harness` (48 files, 667 tests) passes but `harness-doctrine-records.bats`'s real-vault refresh, which fails the same way on main: the vault's crystallize skill is ahead of its committed record.
- The `integration` job is the behavioural check: its container builds dotf from the tree and runs setup-linux, then `verify-setup.bats` counts the deployed skills and commands.

## Decisions made during implementation

- Slice 5 rewire, Linux setup: `dotf` is called by name. Setup exports `~/.local/bin` onto PATH before `install_dotf`, and a failed install already stops setup, so no fallback lookup is needed at this line.
- Slice 3's shell delegation is not done: it would change `compile-harness.sh --deploy`, which no caller runs after this rewire.
- Slice 5 rewire, converge: the composition moved from the `deploy` command into `harness.Deploy`, which returns every surface's outcomes; the command prints from them and converge calls it in process.
- Slice 5 rewire, converge: in process rather than a `dotf harness deploy` child. Converge is the `dotf` binary, so a child would add a lookup of its own path and give nothing back: each surface's error already names the item that failed.
- Slice 5 rewire, converge: `records-harness` stays linux and darwin until `setup-windows.ps1` stops deploying the same files, or Windows would deploy twice.
- Slice 1: the copilot catalog moved to its own slice to keep slice 1 under the executable-line cap (about 293 lines).
- Slice 1: a record whose frontmatter is not valid YAML is now an error for that record (reported after the rest deploy); the shell never parsed it. Every committed record parses (`TestSkillTargetsAgent_EveryCommittedRecordParses`).
- Slice 1: an explicit empty or null `targets:` deploys to nobody, as the shell's substring match did.
- Slice 4 (agents): the model and capability maps are read in process, so the shell's "dotf absent or too old" degrade (a render with no `model:` line and a warning) has no Go form: a tier that cannot be resolved is an error for that record. The shell's whitespace-in-model-id refusal is kept.
- Slice 5: a failed agent record stops presence and doctrine, as the shell did (its `set -e` ended the run there); every other failure is collected and the run goes on. The exit code is 1 for any failure, where the shell used 2 for a failed instruction deploy; every caller treats non-zero alike. `--dry-run` previews every surface but presence, which has no preview mode, and says so.
- Slice 6: the check validates and does not render: the Go skill and agent renders are total functions, so the shell's separate "fails to render" step has nothing left to catch. A drifted region names its first differing line instead of printing a unified diff; the oracle compares every unindented line. Paths in the report have no doubled `/` (the shell's `*/` glob left one). The `ci.yml` and doctor wiring waits for a release that carries the verb (#1814 class), like slice 5's rewire.
- Slice 5: the rewire of `setup-linux.sh`, `setup-windows.ps1` and the `records-harness` converge step waited for a release that carries the verb (#1814 class); 0.77.0 does.
- Slice 4 (doctrine): the roster comes from `harness.RenderPresence` in process, so the shell's "dotf absent, payload carries no roster" degrade has no Go form. The warnings (shadowing file, non-ASCII left after the fold, breached cap in characters or bytes) are returned per file and printed on stderr, in the shell's words.
- Slice 4 (agents): the persona roster stays with `dotf harness presence`; `deploy` composes the two (slice 5).
- Slice 2: the catalog's record order differs from the oracle's. The shell's `*/` glob sorted `test-driven-development/` before `test/` even under `LC_ALL=C` (the `/` is in the sort key); Go uses byte order of the names, as declared. The oracle compares the bullets as a set (lesson 505), so the cutover rewrites the deployed catalog once, in a new order.
- Slice 2: an END line carrying anything after the marker is refused, by every GENERATED-region writer (the catalog skips the file as unmarked; doctrine fails that target, `TestInjectDoctrine_RefusesWhatWouldLoseTheUsersLines`). The shell counted it as an END for validation, then never matched it exactly and dropped the rest of the file.
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
