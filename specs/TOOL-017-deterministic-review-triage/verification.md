---
tags: [spec, verification, templates]
created: "2026-08-18"
---

# Verification - TOOL-017-deterministic-review-triage

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> superseded by `dotf pr triage-queue` (#1057, #1101); owner accepted it on #1062, 2026-10-10
- [x] Criterion 2 -> `cli/internal/prtriage/prtriage.go` reads `harness/review-attestation.json`; test `TestTheRealRegistryServesBothConsumers`
- [ ] Criterion 3 -> declaration and reader landed (`Classify`, tests `TestClassify`, `TestTheRealRegistryDeclaresTheClassificationMarker`, bats `pr-agent: the reviewer is asked for every class marker the registry declares`); a consumer that escalates unclassified findings is still open

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- 2026-10-20, re-baseline: the spec was resumed rather than abandoned (owner). AC1 and AC2 were met outside the spec folder and are ticked with their evidence; AC7 is partly met by `dotf pr land` at PR granularity.
- The class marker is declared in `harness/review-attestation.json` (`classification`), not in Go, so a class change is a registry edit (AC2's rule applied to AC3). A bats contract test ties `.pr_agent.toml` to it.
- Measured before choosing the marker (2026-10-20): PR-Agent wrote the gradation as `**Header (REAL)**` (#2833), as the body's first word `REAL.` (#2832), or not at all (#2839). Hence a whole line `Class: <CLASS>`, matched exactly; two lines that disagree are a conflict and stay unclassified.

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
