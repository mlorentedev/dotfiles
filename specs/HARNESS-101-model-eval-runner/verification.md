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
- [ ] AC3 (slice 2, library half): `CheckoutReplay`. Tests: `TestReplayCheckoutFetchesAReviewedCommitFromItsRef` (no ref created in the clone, worktree removed and pruned), `TestReplayCheckoutFallsBackToTheFetchRef` (protocol v0 refuses the sha; dropping the fetch_ref branch fails it), `TestReplayCheckoutUsesALocalCommitWithoutFetching`, `TestReplayCheckoutNamesACaseItCannotResolve`, `TestReplayCheckoutRefusesARunDirectoryInsideTheRepository`. Commit `f696d224`. The non-zero exit is `dotf eval run`'s, in slice 3.
- [x] AC4 (slice 2): `PlanEvalLaunch`, `WriteEvalRequest`, `LoadCandidates`. Tests: `TestCheckReviewGate_RefusesEvalRecord` (RED was an archive with no pool: the pool check has no opinion there), `TestEvalSlotReadsNoPoolAndWritesNothingUnderSpecs` (a malformed pool in the tree; every file's digest unchanged), `TestEvalSlotRefusesALaunchItCannotKeepApart`, `TestLoadCandidatesRefusesWhatTheLauncherCannotRunSafely`, `TestRepositoryCandidatesLoad`. Mutation: building the eval prompt as a review prompt fails the slot test on "reviewer-pool.json" and "reviewer pool". Commits `9ef3195e`, `bbe037c3`, `0fbaffab`.
- [ ] AC6 (slice 3, in part): `dotf eval run --plan` prints the matrix and a per-candidate estimate (NaN credits, agy runs); a matrix above `--max-runs` is refused, by `--plan` and by a run alike, with the plan still printed. Tests: `TestPlanCountsEveryCaseCandidateAndEpoch`, `TestPlanEstimatesEachCandidateInItsRunnersUnit`, `TestPlanRefusesAMatrixAboveTheCap`, `TestPlanRefusesWhatItCannotPrice`, `TestRepositoryCandidatesPriceTheWholeCorpus`, `TestCandidateRatesMatchTheNaNReadme`, `TestEvalRunPlanPrintsTheMatrixAndEstimate`, `TestEvalRunRefusesAPlanAboveTheCap`, `TestPlanRenderTotalsEachUnitAndNamesTheEnvelope`, `TestPlanRefusesACandidateNamedTwice`, `TestSelectCasesNarrowsToTheNamedOnes`, `TestEvalRunPlanNarrowsToTheNamedCases` (`--case`). Mutations: dropping the cap check fails both cap tests; pricing at the input rate fails the estimate test; a JSON rate of 41 against the README's 42 fails the rate test. Open: the Anthropic estimate (no Anthropic candidate exists) and `measured` envelopes (need stored results).
- [ ] AC5, AC7, AC8: later slices.

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Row classes have a fixed order.** Noise, then decoys, then expected (one to one), then optional, then unadjudicated. A row asserting a known false claim is a false positive even if it also names a real symbol. A row below an entry's `severity_floor` does not satisfy it. ADR-063 §1 amended.
- **A replay source may match its own decoy.** ADR-063 §1 said every case must hit no decoy against its source. That made the most natural decoy, the source reviewer's own false claim (`harness-088-r2`'s `SessionEnd` stamps `claude`), impossible to encode. Now only a planted reference must be decoy-free, and a replay source may match a decoy only on the row its `sample` records. Each decoy carries a `sample` it must match. The first version relied on an over-broad decoy leaving an `expected` entry unmatched; a mutation showed that guard misses an unadjudicated row (widening the 088 decoy to `archive` took line 27, the `extractHandoffBlock` finding, and the check stayed green). `TestCheckRefusesAReplayDecoyThatTakesAnotherRowOfItsSource` now holds it. Owner confirmed this option (A) on 2026-11-14. ADR-063 §1 amended.
- **Planted cases carry no `reviewed_sha`.** `spec.ParseReview` refuses a review without one, as a gate should. `spec.FrontmatterVerdict` reads the verdict alone, with the gate's normalization.
- **Corpus.** Cases `ai-055-r2` (`nan/glm5.3-flash`), `harness-111-r1`, `harness-088-r2`, `harness-105` and `memory-017` (all `agy/gemini-3.1-pro-high`, retired). Each candidate in the first run authored at most one. HARNESS-106 (`nan/mimo-v2.5`) was dropped: its `reviewed_sha` `617cbafe` is gone from GitHub (lesson 601). HARNESS-063 round 2 was dropped: no finding of it was confirmed. Each `confirmed_by` cites the disposition or commit read for it.
- **The eval answer is written inside the replay tree.** ADR-063 §2 said the reviewer writes to the run directory. agy's `--add-dir` names only the tree, and the launcher's stray-write snapshot runs `git status` in the verdict's folder, so a path outside the tree defeats both. The answer goes to `<tree>/.dotf-eval/review.md`, outside `specs/`, and is kept in the run directory. ADR-063 §2 amended.
- **An eval record is refused before the pool check.** The pool check returns "no opinion" in a repository without a pool, so it could not stand for C37 there. The gate now refuses any signature, or launch request, whose reviewer carries `eval:`.
- **Fixture clones use `--no-local`.** A local clone hardlinks the whole object store, so the first replay test passed without fetching anything. Caught by dropping `fetch_ref` from the test and seeing it still pass.
- **Rates live in `candidates.json`, not the README.** ADR-063 §4 said the plan multiplies by the rates in `ai/nan/README.md`. Parsing a prose table at run time would make a doc edit change spend estimates. Each NaN candidate now carries `credits_per_mtok`, and `TestCandidateRatesMatchTheNaNReadme` fails when the two disagree. ADR-063 §4 amended; proposed by the implementer, put to the owner on #3520.
- **Envelopes are priced at the higher rate, and are not a bound.** #3476 measured totals (191,600 tokens for a Hydrogen-high AI-055 replay) with no input/output split. The plan prices each token at the higher of a model's input and output rates rather than invent a split. Only the rate is an upper bound: the envelope was measured on another model and case, so the plan says a run may spend more. No planted run was measured, so planted uses the replay figure. A deepseek-v4-flash plan over the corpus estimates 169 credits for 21 runs; the plan also totals each unit, because NaN bills every model from one pool.
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
