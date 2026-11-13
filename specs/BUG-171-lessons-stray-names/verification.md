---
tags: [spec, verification, templates]
created: "2026-11-12"
id: "BUG-171-lessons-stray-names"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-171-lessons-stray-names

## Evidence

- [x] AC1: `TestLessonsFmt_CheckNamesAFileInAnotherNamingScheme` puts `2026-12-01-stray.md` at the root and `ops/2026-12-02-also.md` in a category directory. `--check` exits non-zero, names both as `unrecognised name: <path>`, and prints the `lesson-NNN-<slug>.md` form in its fix line. Before `StrayNames` existed it printed `[OK] lessons are in their canonical form` and exited 0.
- [x] AC2: The same test holds `_format.md`, `README.md`, `ops/README.md` and `notes.txt`, and none is named. `go run ./cmd/dotf lessons fmt --check` on this repository prints `[OK]`.
- [x] AC3: The same test runs `fmt` without `--check`. It exits 0, prints `left alone: <path>` for the stray file, and the file is still in place.

## Test status

- `go test ./internal/cmd/ ./internal/lessons/` ok. `go vet ./...` and `GOOS=windows go vet ./...` are clean. `golangci-lint run` reports 0 issues, natively and with `GOOS=windows`.
- Smoke test: a scratch directory holding only `2026-12-01-stray.md` exits 1 with the file named and the fix line. `dotf lessons renumber --next` on that file refuses ("not a lesson-NNN-<slug>.md file"), so the fix line does not point at it.

## Decisions made during implementation

- The fix line says `git mv` to the next free number, then `dotf lessons fmt`. The first draft pointed at `renumber --next`, which refuses a file with no number (found in the smoke test).
- `StrayNames` reads the same directories `Plan` does (`lessonDirs`), not the whole tree the way `StrayIndexes` does. A markdown file deeper down is not read as a lesson either, so there is nothing for it to have skipped.
- `_index.md` is on the accepted list here because `StrayIndexes` already reports it under its own label, and naming it twice would give two fixes for one file.
- fmt's reporting became a table of findings (`lessonsLeftAlone`, `reportLessonsCheck`). The third kind of finding pushed `RunE` past gocyclo. The production diff grew past the 50-line spec-gate as a result, which is why this is a spec rather than a skip.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: a missing case in a checker, recorded on #3415 and in this spec
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: it applies the existing one-lesson-format rule (CLI-131) to file names
- [x] New pattern candidate for `00_meta/patterns/`? no: the general rule ("a check reports what it skips") is already the silent-failures guidance

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-171-lessons-stray-names/` -> `specs/archive/BUG-171-lessons-stray-names/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)

## Review dispositions

The review, by nan/qwen3.8-flash at `7f8ed464`, gave PASS.

1. Minor, REAL: nothing tested the `_index.md` exclusion in `lessonsDirDocs`, and deleting it left the suite green. Applied after the review: `TestLessonsFmt_CheckNamesAFileInAnotherNamingScheme` now holds `ops/_index.md` and asserts that it is named once, as `committed index`, never as `unrecognised name`. With the exclusion removed, the test fails (mutation-checked).
2. Minor, SPECULATIVE: an uppercase `NOTE.MD` passes in silence. Declined. The whole package reads only lowercase `.md` as markdown (`fileRE`, `readDir`), and no measured repository has such a name. A fleet case would come back here as an issue.
3. Question: the blast-radius measurement was outside the reviewer's worktree. Accepted as author evidence. The command used is in the PR body.
4. Minor, SPECULATIVE: no command prints the next free number. Declined: the proposal leaves the number as a human choice, and `dotf lessons index` lists every number in use.
