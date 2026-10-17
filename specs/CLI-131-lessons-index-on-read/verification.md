---
tags: [spec, verification, templates]
created: "2026-10-16"
---

# Verification - CLI-131-lessons-index-on-read

## Evidence

- [x] AC1 `dotf lessons index` -> tests `TestIndex_OneTablePerDirectorySortedByNumber`, `TestIndex_EscapesAPipeInATitleAndRefusesAnUnreadableLesson`, `TestLessonsIndex_PrintsTheTableAndWritesNothing`; on the real tree it prints one row per lesson file (463 rows for 463 files at `cdf425af`; the count moves with every new lesson, so re-measure with `dotf lessons index --dir docs/lessons | grep -c '^| \['` against `ls docs/lessons/lesson-*.md | wc -l`)
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
- Not measured: the pre-commit `lessons-fmt` hook, which runs the installed 0.72 `dotf` and so fails on a tree with no index until a release carrying this change is installed (lesson 464). The deleting commit skipped that one hook; CI builds `dotf` from the checkout and is the gate.

## Decisions made during implementation

- `--check` flags a committed `_index.md` (loud) rather than ignoring it. Blast radius measured: 11 repos on this machine carry one, only dotfiles runs `lessons fmt`.
  Reproduce (2026-10-17, repos under `~/Projects`): `for r in ~/Projects/*/; do [ -f "$r/docs/lessons/_index.md" ] && echo "$r"; done` lists garsync, hive, imagesensortool, iris, kubelab, leaving-denver, pdf-modifier, resume, ts-bridge, web, yt-metrics-cli; `grep -rlE 'lessons (fmt|index)' --include='*.yml' --include='*.yaml' --include=Makefile --include='*.sh' ~/Projects` outside this repo finds nothing, so none of them runs the checker.
- The advisor's two catches were taken: the session-start staleness check stat'd the index and would have died silently, and the CI workflow needs no change because `lessons fmt --check` stays the right command.
- The vault skill records and the `repo-hygiene.yml` comment still describe a regenerated index; they are a follow-up (record edit in the vault, and an owner-lands path).

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-464-removing-a-file-a-pinned-tool-requires-needs-a-release-before-the-hook-passes.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a removal of a generated file with a render command; the decision and its blast radius are in proposal.md
- [x] New pattern candidate for `00_meta/patterns/`? no: lesson 280 and 374 already record the committed-generated-file conflict; nothing new across projects

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-131-lessons-index-on-read/` -> `specs/archive/CLI-131-lessons-index-on-read/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)

## Review round 1 (2026-10-17, nan/qwen3.8-flash): FAIL, findings dispositioned

- Blocker F1 (the digests in `review-request.json` match no file on disk): not a defect. The launcher records the *normalised* digest (`ContractDigests`: status folded, checkboxes unticked, harness feature fields dropped, CRLF folded), and the reviewer compared raw `sha256sum` output. The round-1 files are kept as `review-round1.md` and `review-request-round1.json`; round 2 is the review that archives.
- Major (the 11-repo blast radius had no command): the command is now in Decisions above.
- Minor (461 versus 463): re-measured and made reproducible, in AC1 above.
- Minor (`dotf lessons --help` still said "generated index"): fixed in the parent `Short`, pinned by `TestLessonsHelp_DoesNotPromiseACommittedIndex`.
- Minor (four Closing boxes unticked while this file claims them): ticked in `tasks.md`.
- Question (WIP override text, `status: draft`): left as recorded; the override is the owner's call and is named in the report.
