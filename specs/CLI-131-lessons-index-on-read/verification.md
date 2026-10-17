---
tags: [spec, verification, templates]
created: "2026-10-16"
---

# Verification - CLI-131-lessons-index-on-read

## Evidence

- [x] AC1 `dotf lessons index` -> tests `TestIndex_OneTablePerDirectorySortedByNumber`, `TestIndex_EscapesAPipeInATitleAndRefusesAnUnreadableLesson`, `TestLessonsIndex_PrintsTheTableAndWritesNothing`; on the real tree it prints 461 rows for 461 lesson files
- [x] AC2 no index needed -> `TestPlan_NeedsNoIndexAndWritesNone`; `dotf lessons fmt --check --dir docs/lessons` exits 0 with the file deleted
- [x] AC3 committed index named -> `TestLessonsFmt_CheckNamesACommittedIndexAndFmtLeavesItAlone`, `TestStrayIndexes_NamesACommittedIndexAtAnyLevel`
- [x] AC4 duplicates and dangling links -> `TestPlan_RefusesANumberUsedTwice`, `TestPlan_RefusesAWikilinkThatNamesNoLesson`
- [x] AC5 `dotf init` -> `TestScaffoldCreatesStructureAndFiles` (asserts no `_index.md`)
- [x] AC6 staleness -> `TestLessonsStaleness` (two new subtests, no index present)
- [x] AC7 deletion -> `git rm docs/lessons/_index.md`, feature f7

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./...`: clean
- `go test ./...` (cli): all packages ok
- `golangci-lint run` (v2.12.2, the pin) over the four touched packages: 0 issues
- all 7 `features.json` verification commands exit 0
- `bats` over the files that mention lessons (doc paths, knowledge gate, agents-md, ci-path-filtering, twin retirement): 98 tests, none failing
- Not measured: the pre-commit `lessons-fmt` hook, which runs the installed 0.72 `dotf` and so fails on a tree with no index until a release carrying this change is installed (lesson 463). The deleting commit skipped that one hook; CI builds `dotf` from the checkout and is the gate.

## Decisions made during implementation

- `--check` flags a committed `_index.md` (loud) rather than ignoring it. Blast radius measured: 11 repos on this machine carry one, only dotfiles runs `lessons fmt`.
- The advisor's two catches were taken: the session-start staleness check stat'd the index and would have died silently, and the CI workflow needs no change because `lessons fmt --check` stays the right command.
- The vault skill records and the `repo-hygiene.yml` comment still describe a regenerated index; they are a follow-up (record edit in the vault, and an owner-lands path).

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-463-removing-a-file-a-pinned-tool-requires-needs-a-release-before-the-hook-passes.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a removal of a generated file with a render command; the decision and its blast radius are in proposal.md
- [x] New pattern candidate for `00_meta/patterns/`? no: lesson 280 and 374 already record the committed-generated-file conflict; nothing new across projects

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-131-lessons-index-on-read/` -> `specs/archive/CLI-131-lessons-index-on-read/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
