---
spec: "MEMORY-020-one-archive-shape"
verdict: "PASS"
reviewed_sha: "aeaea027c191c9d144a01bcca79824894f00a8ef"
reviewer: "nan/deepseek-v4-flash"
date: "2026-11-14"
---

## Adversarial review

**Scope**: MEMORY-020-one-archive-shape / dotfiles issue #3487, PR #3518 (whole change, not the delta since round 1)
**Sources**: `specs/MEMORY-020-one-archive-shape/{proposal,tasks,verification}.md` + `features.json`; diff
`git diff 45cc40fa65c1532760a826997bf700c62e925ae5...HEAD` (26 files, two spec commits:
`682e1078` and `aeaea027`; `git log --first-parent --no-merges` lists only those two, so no other work
arrived through a merge). Contract digests checked with `dotf spec digest MEMORY-020-one-archive-shape`
against `review-request.json` — `features.json 0664c466…`, `proposal.md 1e28f3f4…`, `tasks.md e243c01b…`,
`spec.md` empty — **all match**. Nothing in the contract set was modified by this review.
Round 1 (`review-round1.md`, `nan/qwen3.8-flash`) returned FAIL on one REAL Blocker (F1, a `--finished`
write deleting another live thread's file) plus two REAL Majors (`verification.md` scaffold, ADR/runbook
SSOT drift); each is re-tested below at this head rather than taken on the author's word.

**Commands run fresh at this head** (evidence for every "green" claim below):
`go build ./...` → 0 · `go vet ./...` → 0 · `GOOS=windows go vet ./...` → 0 ·
`go test ./... -count=1 -timeout 6m` → **exit 0, no failing package** (whole suite, not a slice) ·
all four `features.json` verification commands → exit 0 each, and every test name they `-run` exists
(verified by `-v`: `=== RUN`/`--- PASS` for all five names in f1, and the rest) ·
`golangci-lint run ./internal/mem ./internal/cmd ./internal/worktree ./internal/lane ./internal/doctor`
→ `0 issues.` · two mutation runs (below) reverted, tree clean ·
a `dotf` built to a `mktemp -d` for an end-to-end check of the F1 fix · `grep`/`sed` reads of every
non-test call site. Probes lived under `/tmp`; `git status --porcelain` outside `specs/` is empty.

**UNVERIFIED** (stated rather than assumed): `golangci-lint run ./...` and the `GOOS=windows` lint
run over the whole module (I ran the five touched packages only); PR #3518's CI status (not queried —
this review judges an arbitrary head, not the forge); the archive gate's own implementation (out of
repo by instruction).

### Spec and task alignment

- **AC1 (one shape, one function)** — met, and met structurally. `ArchiveAndPruneThread` and
  `pruneThreadFromMemory` are deleted; `--finished` (`ArchiveThreadAs` → `takeThreads`) and a park
  (`ArchiveParkedThread` → `takeThreads`) share one extraction, and `archiveDir`/`archivedThreadFile`
  are one constant and one function. `archive/<key>.md` is asserted *absent* by
  `TestArchiveParkedThreadUsesTheDatedArchive`, so the retired shape cannot creep back silently.
  Entry points wired: `lane park` (cmd/lane.go:149 → `lane.ParkLane` → `worktree.Park`),
  `worktree done` (cmd/worktree.go:743 `VaultPath` → `Done` → `Park` for a lane),
  `worktree sweep` (cmd/worktree.go:435 → `sweepLane` → `archiveBranchThreads`),
  `lane apply` (lane_apply.go:68). All four non-test `SweepOptions`/`DoneOptions` construction sites
  carry `VaultPath` (worktree.go:433/738, lane_apply.go:68, lane/named.go:593) — checked by grep, 4 of 4.
- **AC2 (byte-identical entry, pointer resolves)** — met. The park side is now pinned by *equality*:
  `TestArchiveParkedThreadUsesTheDatedArchive` compares the whole dated file to a literal `wantArchive`
  (round 1's F4 Minor, closed). The pointer needs no rewrite because `archivedThreadFile` lands the
  file at `archive/threads/<name>.md`, which is what `threads/<name>.md` resolves to from the archive.
- **AC3 (forks archived; Awaiting-Manu block kept and named)** — met.
  `TestArchiveParkedThreadUsesTheDatedArchive` archives `feat-a` + `feat-a+pi`, keeps and names
  `feat-a+agy`, leaves `feat-c` alone, and asserts the kept files stay in `threads/`.
- **AC4 (failed archive still parks, ERROR/WHY/FIX, non-zero)** — met on all four entry points. The
  park result carries the report (`DoneResult.Threads/ThreadsErr`, `SweepReport.ArchivedThreads/
  KeptThreads/ArchiveFailures`), the command prints it and returns the error (`reportParkThreads`,
  `reportSweepThreads`), and the lane is written `PARKED` before the archive is attempted.
  Dry runs write nothing — verified by control flow, not assertion: `sweepLane` returns on
  `opts.DryRun` before `parkSingleNamedLane`, and `DoneOptions` has no dry-run field at all.
- **Round-1 F1 (Blocker: `--finished` deletes another live thread's file)** — **fixed and pinned**.
  `checkThreadFiles` now reads the live file the finished write removes last and requires its heading
  to be this key's; the park's `moveThreadFile` refuses a file whose heading belongs to another key.
  Reproduced end to end with a built binary: write `feat/x`, then `--thread feat-x --finished` →
  **exit 1**, `Error: thread "feat-x" would be stored in the file thread "feat/x" holds`, and
  `threads/feat-x.md` still holds `### thread: feat/x` untouched. Both guards are load-bearing:
  removing the live-file check turns `TestHandoffWriteFinishedRefusesALiveFileAnotherKeyHolds` red
  (`err = <nil>, want the refusal naming feat/x`), and removing the heading check in `moveThreadFile`
  turns `TestArchiveParkedThreadLeavesAFileAnotherKeyHolds` red (the file is moved). Both reverted,
  tree clean.
- **Round-1 Majors** — `verification.md` is filled: named tests per AC, mutation results, decisions on
  F1–F7, promotion candidates answered (lesson 603 written, ADR "no" with a reason, pattern "no").
  SSOT drift is closed: ADR-060's park step and `guide-cross-agent-memory.md` now teach
  `archive/superseded-handoffs-<date>.md` + `archive/threads/`, and the doctor warning + budget refusal
  name the same shape (grep for the retired "move the file to archive/" wording outside `specs/`
  returns nothing).
- **Tasks** — all Implementation and Closing boxes are ticked and each has diff evidence; the round-1
  review-fix rows (F1, F3/F4/F6) map onto `aeaea027`.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in the spec folder (grep: none), so the
  archive's tag pre-flight is satisfied.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Minor | REAL | quality thresholds | `ArchiveParkedThread` is 40 lines at roughly **CC 13**, and `takeThreads` 35 lines at roughly CC 12; AGENTS.md states <40 lines / CC <10. `.golangci.yml` sets gocyclo `min-complexity: 15` and excludes `internal/mem`, so lint cannot surface it. Context that keeps this Minor and not a gate: round 1 measured 52 / CC 17 on the same function, so this change already moved the number down (`appendArchiveFile` extracted), and the deleted `ArchiveAndPruneThread`/`pruneThreadFromMemory` were comparable. | `awk` line count + branch count over `internal/mem/thread_archive.go` at this head; `.golangci.yml:23-34` | n/a (measurement) | code: one more extraction (the empty-key / no-MEMORY guard, or the move loop) |
| Minor | REAL | report accuracy on a partial park | `ArchiveParkedThread` appends `t.key` to `res.Archived` **before** calling `moveThreadFile`, so when a move fails the command prints `archived   thread "X" to …` and then the ERROR, while X's file is still in `threads/` — the exact state `dotf doctor` reports as a lost handoff. The entry *is* archived at that point, so the statement is half-true rather than false; the mislabel is only in the per-file half. | Code read of `ArchiveParkedThread`'s move loop at this head; `TestParkReportsAFailedArchiveAndStillParks` forces the *dated-archive append* to fail, not the move | UNTESTED for a move failure (no test makes `moveThreadFile` fail) | code + tests: append after the move succeeds, or print "entry archived, file left in threads/" and add a named test |
| Minor | REAL | archive readability in the key-collision edge | A park of `feat-x` while `threads/feat-x.md` holds `feat/x` archives the entry but leaves the file, so the archived entry's unchanged `threads/feat-x.md` pointer resolves (from `archive/`) to nothing — the "a reader of the archive cannot follow a finished thread back" symptom this spec exists to remove, now confined to a state that was already ambiguous before the park (that pointer never named the parked thread's body). The park neither names the case in its report nor says the pointer is unusable. Not introduced by this change and strictly better than the alternative round 1 found (moving another thread's file). | `TestArchiveParkedThreadLeavesAFileAnotherKeyHolds` constructs exactly this state and asserts only that the file stays put; code read of `moveThreadFile`'s skip branch | `TestArchiveParkedThreadLeavesAFileAnotherKeyHolds` covers the file-stays half; the pointer-resolution half is UNTESTED | code (one report line) or spec wording in `verification.md`/ADR-060; **not** in the contract set, so it cannot be fixed in `proposal.md` under this verdict |
| Minor | REAL | output stream consistency | `worktree sweep` prints its archive report on **stdout** (worktree.go:488) while `lane park` and `worktree done` use **stderr**, and sweep's returned error then reaches cobra after the hand-built `ERROR: ` prefix. Round-1 F7, declined in `verification.md` with a stated reason (each command keeps the stream its other report lines already use). Recorded here so the decline is visible, not re-litigated. | Call sites at this head | `TestReportParkThreads*`, `TestReportSweepThreadsFailsOnAnArchiveFailure` (cover text, not stream) | declined — no action required |

Nothing found in the diff's security columns at this head: no shell built from a thread key (all git
calls are `exec.Command` argument vectors), no new path taken from user input unfiltered (both new
`os.ReadFile` sites carry `#nosec G304` with a stated reason and are confined beside `MEMORY.md`;
`threadFileName` refuses anything that could leave `threads/`), no secret handling, no new dependency,
`GOOS=windows go vet` clean, and both new archive paths take `LockHandoffMemory` (bounded) exactly as
the writer does. The round-1 destructive-remove path is now guarded on both sides.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | **A** | All four ACs verified with named tests, negative paths (collision, Awaiting-Manu, failed archive, repeated park) covered, both round-1 guards mutation-proven, whole `go test ./...` green, and the F1 scenario re-checked end to end with a real binary. |
| Verification | **A** | `verification.md` names a test per criterion with the exact commands, and each was re-run here green; the mutation claims in it reproduce; the two declines carry reasons. |
| Scope | **B** | Everything traces to a proposal bullet or the Definition of Done (ADR-060 amendment, runbook, lesson 603, doctor/budget wording, `VaultPath` threading); no test was deleted or weakened — but `releaseParkedLane`'s parameter reorder and the `SessionSkip` type reused for `ArchiveFailures` are incidental edits the proposal does not name. |
| Reliability | **B** | Write order honoured and tested, dry runs write nothing, a failed archive is reported with a non-zero exit; the move-failure mislabel and the stdout/stderr split are the gaps. |
| Maintainability | **B** | Lint 0 on every touched package, one shared extraction, honest comments; but `ArchiveParkedThread` is still above the repo's own line/length thresholds and `SessionSkip`'s doc no longer fits its second use. |
| Handoff-readiness | **A** | Lesson 603 written and linked, ADR-060 amended in place with the spec ref, runbook updated, `verification.md` decisions answer every round-1 question (including AC1's `worktree done` reading), tasks all ticked. |

Aggregation: no D, no C → the rubric path yields **PASS**. Severity path: no Blocker, no REAL Major,
Minors only → **PASS**. The two paths agree.

### Verdict
PASS

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is **closed** under this verdict — any
edit to it invalidates the digests above and this review. Everything below belongs to
`verification.md`, to code/tests, or to a follow-up ticket, and should be dispositioned there as
applied, ticketed, or declined with a reason:

- Record the four Minor dispositions in `verification.md` (applied / ticketed / declined + reason).
  The stdout/stderr one is already declined there; the other three are new and need a line each.
- Optional, code: one more extraction in `ArchiveParkedThread` to get under the CC 10 line, and report
  "entry archived, file left in `threads/`" when `moveThreadFile` fails — a named test that makes the
  move fail would pin it (`TestParkReportsAFailedArchiveAndStillParks` covers only the append).
- Optional, code: one line in the park report for the key-collision case so a reader is told the
  archived pointer is unusable, or a sentence in ADR-060 accepting the ambiguity.
- Vault/board: the spec's own archive checklist remains — flip `status: archived`, move the folder to
  `specs/archive/`, and close #3487 with the PR link (ADR-018).

`dotf spec archive MEMORY-020-one-archive-shape` is **advisable in the current state**: the verdict is
PASS, the digests are fresh, no `[AGENT-DRAFT]` tags remain, and `verification.md` answers every
promotion line. The Minors above are tracked work, not archive blockers.
