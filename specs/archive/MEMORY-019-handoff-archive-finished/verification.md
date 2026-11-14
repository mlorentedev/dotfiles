---
tags: [spec, verification, templates]
created: "2026-11-13"
id: "MEMORY-019-handoff-archive-finished"
type: spec-verification
status: active
owner: manu
---

# Verification - MEMORY-019-handoff-archive-finished

## Evidence

- [x] AC1 -> commit `b64c11c1` / `TestArchiveThreadDropsTheEntryAndReturnsTheBlockItWouldHavePlaced`, `TestArchiveThreadRemovesAnExistingEntryOnly`, `TestAppendArchive` (cli/internal/mem/thread_archive_test.go), `TestHandoffWriteFinishedArchivesTheThread` (cli/internal/cmd/mem_handoff_finished_test.go)
- [x] AC2 -> commit `b64c11c1` / `TestArchiveThreadRefusesAThreadAwaitingManu`, `TestHandoffWriteFinishedRefusesAwaitingAndNoneIsNotAMarker`
- [x] AC3 -> commit `b64c11c1` / `TestArchiveThreadByAnotherAgentArchivesOnlyItsFork`, `TestArchiveThreadByItsWriterKeepsAnotherAgentsFork`, `TestPruneExactKeepsForks`
- [x] AC4 -> commit `b64c11c1` / `TestHandoffWriteFinishedIsIdempotentAndDryRuns`

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./...`: clean. `go test ./... -count=1`: every package ok.
- `golangci-lint run` and `GOOS=windows golangci-lint run` on `internal/mem` and `internal/cmd`: 0 issues.
- Mutation: pruning forks in `ArchiveThreadAs` (`true` for `false`) turns `TestArchiveThreadByItsWriterKeepsAnotherAgentsFork` red. The fork-side test alone did not catch it, which is why the writer-side test exists.
- Manual smoke test on a copy of the dotfiles vault's MEMORY.md (2026-11-14): `handoff-write --finished --thread feat-worktree-one-root --agent claude` removed exactly that thread's seven lines, every other line unchanged, and wrote `archive/superseded-handoffs-<date>.md` opening with its heading and holding the new entry. Run for a key absent from MEMORY.md, the file came out byte-identical.
- No regressions in the existing suite: yes. `ArchiveAndPruneThread` keeps its fork-pruning behaviour through the new `forks` parameter.

## Decisions made during implementation

- The dated archive file, not `archive/<key>.md`: every archive in the vault has the dated shape, and the lane-park path's shape has no instance. Converging the two is #3487.
- The prune is the lane-park's `pruneThreadFromMemory` with an exact-key mode, not a new loop.

- Review (PASS, two minors) dispositioned, both applied: `AppendArchive` trims a CRLF ending (`TestAppendArchiveTrimsACRLFEnding`), and an end-to-end test removes a thread already on disk (`TestHandoffWriteFinishedRemovesAnExistingThreadOnDisk`).

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? no: nothing went wrong; the mutation gap is recorded above
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: an additive flag inside ADR-056's store; the open shape question is #3487
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: specific to dotf mem

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/MEMORY-019-handoff-archive-finished/` -> `specs/archive/MEMORY-019-handoff-archive-finished/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
