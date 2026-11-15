---
tags: [spec, verification, templates]
created: "2026-11-14"
id: "MEMORY-021-shared-thread-file-report"
type: spec-verification
status: active
owner: manu
---

# Verification - MEMORY-021-shared-thread-file-report

## Evidence

- [x] AC1 -> commit `e0e6da2e` / tests `TestArchiveParkedThreadLeavesAFileAnotherKeyHolds` (`Shared` holds the key, the file stays), `TestReportParkThreadsNamesASharedThreadFile` (the park and sweep text lines), `TestSweepReportRecordsThreadArchives` (`SharedThreads`), and `TestSweepReportNamesSharedThreadsOnTheWire` (the `shared_threads` JSON name, added after review). All three failed to compile before the change because the fields did not exist.
- [x] AC2 -> commit `e0e6da2e`. features.json f2 checks the 40-line limit mechanically. CC was counted by hand from decision points, because gocyclo excludes `internal/mem` and no CC tool is pinned in `versions.conf`. Counts: `takeThreads` 8, `parkTarget` 7, `archiveParked` 5, `moveThreadFile` 5, `moveThreadFiles` 3, `blockEnd` 3, `takesThread` 3. Every existing archive test passes unchanged.

## Test status

- Test suite: `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` -> all ok. `golangci-lint run ./...` on Linux and with `GOOS=windows` -> 0 issues.
- Manual smoke test: none. The change is a report line and a refactor, both covered by unit tests.
- No regressions in existing test suite: yes.

## Decisions made during implementation

- `Shared` is a separate list beside `Archived`, not a replacement for it. The entry is archived either way; `Shared` adds that its pointer leads to another thread's file.
- The collision is reported, not resolved. The file belongs to the other live thread, so moving or renaming it is out of scope (see proposal).

- Review (PASS-WITH-GAPS at a28674bf), Major on an untested `shared_threads` wire name: applied. `TestSweepReportNamesSharedThreadsOnTheWire` mirrors the `lane_skips` contract test. features.json is frozen by the review, so f1 does not name it.
- Review, Major that nothing guards the negative side of `Shared`: applied. `TestArchiveParkedThreadUsesTheDatedArchive` asserts that no own-file thread is reported shared, and it fails under the review's mutation (`shared` reported on every move).
- Review, Minor that no command emits `SweepReport` as JSON: declined. The struct tag is the contract the type's readers hold, as `archived_threads` and `lane_skips` already are. The proposal is frozen by the review.
- Review, Minor that two existing tests were edited, against AC2's "unchanged": declined. Both edits only add assertions. AC2 means that nothing was weakened or deleted, which the review confirmed.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: a review follow-up with no new failure mode; lesson 603 already covers MEMORY-020's
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a report line and a refactor change no decision
- [x] New pattern candidate for `00_meta/patterns/`? no: specific to this repository's memory store

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/MEMORY-021-shared-thread-file-report/` -> `specs/archive/MEMORY-021-shared-thread-file-report/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
