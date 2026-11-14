---
tags: [spec, verification, templates]
created: "2026-11-14"
id: "MEMORY-020-one-archive-shape"
type: spec-verification
status: active
owner: manu
---

# Verification - MEMORY-020-one-archive-shape

## Evidence

- [x] AC1 (one shape, one function) -> `takeThreads` is the only extraction; tests `TestArchiveParkedThreadUsesTheDatedArchive`, `TestHandoffWriteFinishedArchivesTheThread`, `TestHandoffWriteFinishedMovesALiveThreadFile`, `TestParkArchivesTheBranchThread`, `TestLaneParkInPlaceAndTipVerification` (each asserts the dated file, `archive/threads/<key>.md`, and no `archive/<key>.md`)
- [x] AC2 (byte-identical entry, pointer resolves) -> `TestArchiveThreadDropsTheEntryAndReturnsTheBlockItWouldHavePlaced`, `TestHandoffWriteFinishedArchivesTheThread` and, after review F4, the exact archive text in `TestArchiveParkedThreadUsesTheDatedArchive`
- [x] AC3 (forks archived, Awaiting Manu kept and named) -> `TestArchiveParkedThreadUsesTheDatedArchive`, `TestReportParkThreadsNamesArchivedAndKept`, `TestSweepReportRecordsThreadArchives`
- [x] AC4 (failed archive still parks, ERROR/WHY/FIX, non-zero) -> `TestParkReportsAFailedArchiveAndStillParks`, `TestReportParkThreadsTurnsAFailedArchiveIntoTheError`, `TestReportSweepThreadsFailsOnAnArchiveFailure`, `TestSweepReportRecordsThreadArchives`
- [x] Review F1 (a colliding key must not take another thread's file) -> `TestHandoffWriteFinishedRefusesALiveFileAnotherKeyHolds`, `TestArchiveParkedThreadLeavesAFileAnotherKeyHolds`; each was red with its guard removed (mutation run in this session) and green with it

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./...` -> ok; `go test ./...` -> all ok; `golangci-lint run ./...` and `GOOS=windows golangci-lint run ./...` -> 0 issues (rerun after the review fixes, see the PR's triage comment)
- Manual smoke: the review reproduced F1 with a built binary (`--thread feat/x`, then `--thread feat-x --finished`); after the fix the same sequence is refused, pinned by the command-level test above
- No regressions: yes, the full suite passes

## Decisions made during implementation

- The thread file moves to `archive/threads/`, not stays in `threads/`: `dotf doctor`'s handoff-threads check reports a file without its entry as a lost handoff, which `--finished` as merged in #3488 tripped. Under `archive/threads/` the entry needs no pointer rewrite. The first build rewrote the pointer to `../threads/`; dropped (lesson 603).
- A park leaves a `threads/<name>.md` whose heading names another key where it is: `fix/x` and `fix-x` share a file name, and the file is the other live thread's. `--finished` refuses that case, as it did before this change.
- Review F5 (a park with no vault resolving prints nothing): declined. The behaviour predates this change, and `VaultPath`'s doc names it. A machine without a vault has no MEMORY.md to keep stale.
- Review F7 (sweep prints its park report on stdout, lane park and worktree done on stderr): declined. Each command keeps the stream its other report lines already use.
- Review question on AC1's `dotf worktree done`: intended. Only a named lane parks; a plain worktree's `done` removes it and archives nothing, as before. The list names the commands that can reach a park.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-603-nothing-reads-it-is-a-claim-about-every-package-that-imports-the-name.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the layout decision amends ADR-060's park step in place (docs/adr/adr-060-persistent-named-lanes-sticky-personas-and-meeseek-workers.md), and needs no new ADR
- [x] New pattern candidate for `00_meta/patterns/`? no: the archive layout is this repository's own memory store, used by no other project

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/MEMORY-020-one-archive-shape/` -> `specs/archive/MEMORY-020-one-archive-shape/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
