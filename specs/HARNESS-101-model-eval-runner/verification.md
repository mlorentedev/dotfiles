---
tags: [spec, verification, templates]
created: "2026-11-13"
id: "HARNESS-101-model-eval-runner"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-101-model-eval-runner

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (slice 1): `harness/evals/case.schema.json`, embedded with an identity test. Tests: `TestValidateCaseRejectsMalformedCases` (19 mutations, each one property), `TestEmbeddedCaseSchemaEqualsTheRepoSchema`, `TestRepositoryCorpusChecksClean` (5 replay, 2 planted). Commits `361321c1`, `86e76cd9`.
- [x] AC2 (slice 1): `dotf eval check`. Tests: `TestCheckReportsEachBrokenProperty`, `TestCheckLetsAReplaySourceCarryItsOwnDecoy`, `TestCheckHoldsAPlantedReferenceToTheIdealAnswer`, `TestGradeAssignsRowsOneToOneWhenFindingsShareASymbol`, `TestEvalCheckFailsAndNamesABrokenCase`. CI: `TestRepositoryCorpusChecksClean` runs in `cli.yml`'s test job, which has no path filter. Mutation: renaming the `latch` anchor in `harness-105.json` fails it with "expected wrapped-latch-prefix matches no row of source_review".
- [ ] AC3-AC8: later slices.

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Row classes have a fixed order.** Noise, then decoys, then expected (one to one), then optional, then unadjudicated. A row asserting a known false claim is a false positive even if it also names a real symbol. A row below an entry's `severity_floor` does not satisfy it. ADR-063 §1 amended.
- **A replay source may match its own decoy.** ADR-063 §1 said every case must hit no decoy against its source. That made the most natural decoy, the source reviewer's own false claim (`harness-088-r2`'s `SessionEnd` stamps `claude`), impossible to encode. Now only a planted reference must be decoy-free. Each decoy carries a `sample` it must match, and an over-broad decoy still fails because it leaves an `expected` entry unmatched. ADR-063 §1 amended.
- **Planted cases carry no `reviewed_sha`.** `spec.ParseReview` refuses a review without one, as a gate should. `spec.FrontmatterVerdict` reads the verdict alone, with the gate's normalization.
- **Corpus.** Cases `ai-055-r2` (`nan/glm5.3-flash`), `harness-111-r1`, `harness-088-r2`, `harness-105` and `memory-017` (all `agy/gemini-3.1-pro-high`, retired). Each candidate in the first run authored at most one. HARNESS-106 (`nan/mimo-v2.5`) was dropped: its `reviewed_sha` `617cbafe` is gone from GitHub (lesson 601). HARNESS-063 round 2 was dropped: no finding of it was confirmed. Each `confirmed_by` cites the disposition or commit read for it.
- **The planted bash defect is bash ≥4.1 only.** macOS `/bin/bash` 3.2.57 exits 0 on it, while bash 5.3 exits 1 (measured 2026-11-14). The reference answer says so, and lesson 004 is amended.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-101-model-eval-runner/` -> `specs/archive/HARNESS-101-model-eval-runner/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
