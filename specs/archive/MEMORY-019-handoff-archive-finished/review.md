---
spec: "MEMORY-019-handoff-archive-finished"
verdict: "PASS"
reviewed_sha: "0b4e2510ae04bc097fe488e1549012761f307215"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-13"
---

## Adversarial review

**Scope**: MEMORY-019-handoff-archive-finished
**Sources**: specs/MEMORY-019-handoff-archive-finished/{proposal,tasks,verification}.md, git diff 9553caa3e12a3dd52cda883cff29ee79dcac7a5c...HEAD

### Spec and task alignment
- **AC1** (archive move, byte-identical threads, thread file write, stderr reporting): Implemented in `ArchiveThreadAs`, `AppendArchive`, and `publish`. Tested in `TestArchiveThreadDropsTheEntryAndReturnsTheBlockItWouldHavePlaced`, `TestArchiveThreadRemovesAnExistingEntryOnly`, `TestAppendArchive`, and `TestHandoffWriteFinishedArchivesTheThread`.
- **AC2** (Awaiting Manu refusal, non-marker "None until"): Implemented via `strings.TrimSpace(ParseThread(body).All("Awaiting Manu")) != ""` check in `ArchiveThreadAs` and explicit `--finished` boolean flag check in `store`. Tested in `TestArchiveThreadRefusesAThreadAwaitingManu` and `TestHandoffWriteFinishedRefusesAwaitingAndNoneIsNotAMarker`.
- **AC3** (agent fork isolation): Handled by delegating ownership to `StoreThreadAs` with `pruneThreadFromMemory(..., forks=false)`. Tested in `TestArchiveThreadByAnotherAgentArchivesOnlyItsFork`, `TestArchiveThreadByItsWriterKeepsAnotherAgentsFork`, and `TestPruneExactKeepsForks`.
- **AC4** (idempotence and dry run): Verified via `AppendArchive` substring guard and dry-run conditional in `publish`. Tested in `TestHandoffWriteFinishedIsIdempotentAndDryRuns`.
- **Tasks**: All 10 implementation tasks in `tasks.md` are marked `[x]` and corroborated in the diff. Closing items `[ ]` are cleanly deferred to the closing/archive workflow.
- **Features**: `features.json` specifies 4 executable test commands for AC1-AC4, all passing with exit code 0; lifecycle states correctly stay `pending` pending harness gate verification.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | line-endings | `AppendArchive` trims `\n` via `strings.TrimRight(existing, "\n")`; if an archive file has Windows CRLF (`\r\n`), a dangling `\r` remains before `\n\n`. | Code inspection of `cli/internal/mem/thread_archive.go:65` | UNTESTED | code (`cli/internal/mem/thread_archive.go`) |
| Minor | THEORETICAL | test-coverage | `TestHandoffWriteFinishedArchivesTheThread` writes a thread key absent from initial `finishedMemory`. While unit tests cover pruning an existing entry, an end-to-end cmd test verifying on-disk MEMORY.md removal of a pre-existing thread is not explicit. | `cli/internal/cmd/mem_handoff_finished_test.go:28` | `TestArchiveThreadRemovesAnExistingEntryOnly` | tests (`cli/internal/cmd/mem_handoff_finished_test.go`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All acceptance criteria satisfied, edge cases and refusals verified |
| Verification       | A | Reproducible tests cover all criteria, negative paths, and fork mutations |
| Scope              | A | Exact implementation of #3393 without creep or unrelated touches |
| Reliability        | A | Safe write ordering (archive -> thread file -> memory), locking, and idempotence |
| Maintainability    | A | Clean Go idioms, modular pure mem package functions, CC <= 10, functions <= 40 lines |
| Handoff-readiness  | A | Spec complete, promotion candidates addressed, no agent-draft tags |

### Verdict
PASS

### Recommended next steps
- The implementer may optionally address the minor CRLF trimming defensive enhancement in `AppendArchive` (`strings.TrimRight(existing, "\r\n")`) and note disposition in `verification.md`.
- `dotf spec archive MEMORY-019-handoff-archive-finished` is **advisable** in the current state.
