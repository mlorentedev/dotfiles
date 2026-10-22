---
tags: [spec, verification, templates]
created: "2026-08-18"
---

# Verification - TOOL-017-deterministic-review-triage

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> superseded by `dotf pr triage-queue` (#1057, #1101); owner accepted it on #1062, 2026-10-10
- [x] Criterion 2 -> `cli/internal/prtriage/prtriage.go` reads `harness/review-attestation.json`; test `TestTheRealRegistryServesBothConsumers`
- [ ] Criterion 3 -> declaration and reader landed (`Classify`, tests `TestClassify`, `TestTheRealRegistryDeclaresTheClassificationMarker`, bats `pr-agent: the reviewer is asked for every class marker the registry declares`); the consumer landed with tests `TestUnclassifiedFindings`, `TestFetchReportsUnclassifiedInlineFindings`, `TestFetchReadsNoInlineFindingsWithoutADeclaration`, `TestFetchMarksUnreadInlineFindings`, `TestPrTriageQueue_NamesUnclassifiedFindings`; it reports only once the registry declares `classification.reviewers`

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- 2026-10-20, re-baseline: the spec was resumed rather than abandoned (owner). AC1 and AC2 were met outside the spec folder and are ticked with their evidence; AC7 is partly met by `dotf pr land` at PR granularity.
- The class marker is declared in `harness/review-attestation.json` (`classification`), not in Go, so a class change is a registry edit (AC2's rule applied to AC3). A bats contract test ties `.pr_agent.toml` to it.
- Measured before choosing the marker (2026-10-20): PR-Agent wrote the gradation as `**Header (REAL)**` (#2833), as the body's first word `REAL.` (#2832), or not at all (#2839). Hence a whole line `Class: <CLASS>`, matched exactly; two lines that disagree are a conflict and stay unclassified.
- 2026-10-21, consumer: an unclassified finding is REPORTED by `dotf pr triage-queue` and does not make a PR pending. Making it pending would change the land gate before the marker has a measured adoption window (the proposal's escalation risk). Live: the first review after #2845 (#2849, claude-haiku) emitted the marker on 2 of 2 inline findings.
- The findings are read from `pulls/N/comments`: PR-Agent's COMMENTED review has an empty body, so the reviews endpoint the verdict reads carries no finding text (lesson 448, applied to the classifier). They are read only for a PR the queue reports, and only when the registry names a classifying reviewer, so the session-start probe pays nothing extra by default.
- Which reviewers are asked for the marker is declared in the registry (`classification.reviewers`), not inferred: CodeRabbit's inline findings carry its own severity format and would all read as unclassified.
- 2026-10-21, owner decisions on #1062: AC4-AC6 option A (a reproduction is a committed failing test or a recorded non-zero exit, never a model's judgement); AC7 option B (the PR-level gate stands). The registry now names `github-actions` as the one classifying reviewer.
- 2026-10-21, AC5 (`dotf pr reproduce`): a check that never ran as designed (start failure, 126/127, a signal, a timeout) is `could-not-run`, a state of its own (owner), because its non-zero exit is not evidence of a defect. The command exits 1 for a row without a verdict (`could-not-run`, `did-not-land`), which keeps main.go's one exit contract.
- `--probe` is a mutation, not an uncommitted test: under option A a test that names the defect is committed and run without a probe. The patch lives outside the tree, since an untracked file is a dirty tree and is refused.
- The probe is applied, hashed and reverted from the repository root, because a patch names files from there. The first version hashed cwd-relative paths and reported a probe that had landed from `cli/` as `did-not-land` (lesson 512). `WaitDelay` bounds `Wait` after a timeout kill, because a child holding the output pipe outlives the kill.
- The command applies nothing and decides nothing: a human pastes its row into `## Review triage`. Gating `apply` on it is AC4, the next slice.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons.md`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/TOOL-017-deterministic-review-triage/` -> `specs/archive/TOOL-017-deterministic-review-triage/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
