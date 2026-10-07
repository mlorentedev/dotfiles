---
tags: [spec, verification, templates]
created: "2026-10-06"
---

# Verification - DOCS-020-lessons-fmt

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 -> `TestNormalize_EveryShapeReachesTheCanonicalForm` (five shapes, idempotence on each), `TestNormalize_KeepsTheOtherFrontmatterKeysAndTheBody`, `TestNormalize_TheSeedTitleWinsOnlyWhileTheFileDeclaresNone`, `TestNormalize_RefusesWhatItCannotRead`
- [x] AC2 -> `TestRenderIndex_ReplacesTheHandKeptTableOnceThenItsMarkers`, `TestIndexTitles_ReadsTheHandKeptRows`
- [x] AC3 -> `TestPlan_RefusesANumberUsedTwice`, `TestPlan_RefusesLessonsWithoutAnIndex`, `TestPlan_RequiresTheRootIndexWhenOnlyACategoryHasLessons`
- [x] AC4 -> `TestLessonsFmt_CheckFailsThenFmtConvergesThenCheckPasses`; on the repository, `fmt --check` passes after the migration
- [x] AC5 -> `TestLessonsIndexTemplateIsWhatFmtRenders`
- [x] AC6 -> `.github/workflows/repo-hygiene.yml`

## Test status

- Migration on the repository (macOS arm64): `dotf lessons fmt --check` reported all 339 lessons and the index as `not formatted:`; `dotf lessons fmt` -> `formatted 340 file(s)`; a second `--check` -> `[OK]`; `bash scripts/check-lessons.sh` -> `OK (339 lessons)`
- Samples read in the diff: lesson 215 (no frontmatter, inline date), 026 (vault shape), 213 (H1 `213 — …`, with the curated title seeded from the index), 337 (title/date shape), and the head and tail of the index
- Measured before migrating: 0 lessons without a date once inline `**Date:**` is counted; 2 dates a day off the index (233, 234; the file wins); 77 titles differing from the index (the curated index title is seeded)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- The H1 pattern first ended in `\s*`, which under `(?m)` matched the newlines after the H1 and deleted one blank line per run, so `fmt` never converged. The command-level test caught it, and the unit fixtures gained the blank-line and bare-number shapes.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/DOCS-020-lessons-fmt/` -> `specs/archive/DOCS-020-lessons-fmt/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
