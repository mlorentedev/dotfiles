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
- [x] AC7 -> `TestPlan_RefusesAWikilinkThatNamesNoLesson`; `.pre-commit-config.yaml` hook `lessons-fmt`; the three files deleted. On the real tree, an appended `[[lesson-999-nope]]` made `fmt --check` exit 1 naming the file and link, and the 13 existing wikilinks all resolve

## Test status

- Migration on the repository (macOS arm64): `dotf lessons fmt --check` reported all 339 lessons and the index as `not formatted:`; `dotf lessons fmt` -> `formatted 340 file(s)`; a second `--check` -> `[OK]`; `bash scripts/check-lessons.sh` -> `OK (339 lessons)`
- Samples read in the diff: lesson 215 (no frontmatter, inline date), 026 (vault shape), 213 (H1 `213 — …`, with the curated title seeded from the index), 337 (title/date shape), and the head and tail of the index
- Measured before migrating: 0 lessons without a date once inline `**Date:**` is counted; 2 dates a day off the index (233, 234; the file wins); 77 titles differing from the index (the curated index title is seeded)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- The H1 pattern first ended in `\s*`, which under `(?m)` matched the newlines after the H1 and deleted one blank line per run, so `fmt` never converged. The command-level test caught it, and the unit fixtures gained the blank-line and bare-number shapes.

- Review round (pr-agent, 00728cc): the H1 prefix rule accepted a hyphen glued to digits, so `3-2-1 backup rule` lost `3-` and `12-factor apps` lost `12-`. It now strips only the measured forms: `Lesson NNN` with `:`/`—`/`–`/`-`, or a bare `NNN` followed by a spaced em or en dash (`TestNormalize_StripsOnlyTheMeasuredNumberingPrefixes`).
- Review round: a directory with no lessons left (all moved into categories) kept a stale generated table, because `planDir` returned early. It now regenerates that table empty when the index carries the markers (`TestPlan_RegeneratesAnIndexWhoseDirectoryHasNoLessonsLeft`).
- Merged origin/main instead of rebasing. The index conflict was resolved by re-running `fmt`, which also normalised lesson 339, added on main in the old shape. This is the resolution the PR body describes.
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
