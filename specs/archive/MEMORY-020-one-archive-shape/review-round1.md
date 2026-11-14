---
spec: "MEMORY-020-one-archive-shape"
verdict: "FAIL"
reviewed_sha: "682e107846b8044e107047a638362196b62f791c"
reviewer: "nan/qwen3.8-flash"
date: "2026-11-14"
---

## Adversarial review

**Scope**: MEMORY-020-one-archive-shape / dotfiles PR #3518 (issue #3487)
**Sources**: `specs/MEMORY-020-one-archive-shape/{proposal,tasks,verification}.md` + `features.json`; diff
`git diff 45cc40fa65c...682e107846b8` (24 files, one spec commit, `git log --first-parent --no-merges` lists only `682e1078`);
PR body and `statusCheckRollup` for #3518. Contract digests checked with `dotf spec digest MEMORY-020-one-archive-shape`
against `review-request.json`: `features.json 0664c466…`, `proposal.md 1e28f3f4…`, `tasks.md e4a1d010…` — all three match,
`spec.md` empty. Nothing in the contract set was modified by this review.

**Commands run fresh at this head** (evidence for every "green" claim below):
`go build ./...` → 0; `go vet ./...` → 0; `GOOS=windows go vet ./...` → 0;
`go test ./internal/mem ./internal/cmd ./internal/worktree ./internal/doctor -count=1` → all `ok` (cmd 174s, worktree 130s);
the four `features.json` verification commands → all exit 0, and all 11 test names they `-run` exist (grep count 1 each, so none is vacuous);
`golangci-lint run ./internal/mem ./internal/cmd ./internal/worktree ./internal/doctor ./internal/lane` → `0 issues.`;
`gocyclo` via a throwaway config under `/tmp` for the complexity measurements;
the CLI itself (`dotf` built to `mktemp -d`) for the reproduction in F1, and the same two probes against a `git archive`
of the base commit `45cc40fa` to show the behaviour is a regression. Working tree left clean apart from this file.
Full `go test ./...` was **not** run (UNVERIFIED beyond the four touched packages and CI, whose matrix on this head is green).

### Spec and task alignment

- **AC1 (one shape, one function)** — met, and met structurally: `ArchiveAndPruneThread` and `pruneThreadFromMemory` are gone,
  both paths call `takeThreads`. Mutation proof that the shape is pinned, not just commented: changing
  `archivedThreadFile` to `archive/<name>.md` (the retired shape) turned **four** named tests red
  (`TestArchiveThreadDropsTheEntryAndReturnsTheBlockItWouldHavePlaced`, `TestArchiveParkedThreadUsesTheDatedArchive`,
  `TestHandoffWriteFinishedArchivesTheThread`, `TestHandoffWriteFinishedMovesALiveThreadFile`), reverted, tree clean.
- **AC2 (byte-identical, pointer resolves)** — met for `--finished`, under-pinned for the park (F4).
- **AC3 (park archives forks, keeps and names an Awaiting-Manu block)** — met.
  `TestArchiveParkedThreadUsesTheDatedArchive` asserts base + fork archived, `feat-a+agy` kept and named, `feat-c` untouched;
  mutation proof: neutering the `keep` predicate turns it red (verified, reverted).
- **AC4 (failed archive still parks, reports ERROR/WHY/FIX, exits non-zero)** — met on all four entry points.
  Every `DoneOptions`/`SweepOptions` construction site in non-test Go now carries `VaultPath` (4 of 4: `cmd/worktree.go:433`,
  `cmd/worktree.go:738`, `cmd/lane_apply.go:68`, `lane/named.go:593`), and `ParkLane`'s duplicate call is gone.
  Both dry-run paths return before any write or remove (`worktree.go:478` early-returns before `reportSweepThreads`;
  `mem_handoff.go:214` before `write()`), checked by reading, since a dry run writes nothing to disprove.
- **Write order** — the proposal's stated invariant (dated archive → MEMORY.md → file moves, "never an entry in neither")
  is what `ArchiveParkedThread` does, and `TestParkReportsAFailedArchiveAndStillParks` blocks the archive directory and
  asserts MEMORY.md still holds the thread and the lane is still `PARKED`.
- **Tasks** — the six Implementation boxes are ticked and each has diff evidence. **All seven Closing boxes are unticked**,
  including "verification.md filled in" and "Every acceptance criterion is covered by at least one test" (F2).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| **Blocker** | **REAL** | data loss / guard bypass | `--finished` checks `CheckThreadFile` against the **archived** path and then removes the **live** file unconditionally, so a finished write under a key that sanitizes to *another* live thread's file name deletes that thread's body and leaves its entry pointing at nothing. `mem.takeThreads`' key is the *sanitized* branch, so `feat/x` (heading) and `feat-x` (heading) share `threads/feat-x.md`: this is the exact collision `CheckThreadFile` exists for ("fix/x and fix-x name one file, and the file's own heading says whose it is"). The park path has the same shape through `moveThreadFile`'s `os.Rename` (code-traced, not reproduced). | Reproduced at this head with a built binary in `mktemp -d`: write `--thread feat/x`, then `--thread feat-x --finished` → **exit 0**, `archived thread "feat-x"`, `threads/feat-x.md` gone, `archive/threads/feat-x.md` holds the *dash* body, and MEMORY.md still carries `### thread: feat/x` with `[full thread](threads/feat-x.md)`. Same two commands against a `git archive` of the base `45cc40fa`: **exit 1**, `Error: thread "feat-x" would be stored in the file thread "feat/x" holds`, nothing written, live file intact. A regression, not a pre-existing hole: `mem_handoff.go:146` reads `prev` from `res.File`, which `ArchiveThreadAs` now retargets to `archive/threads/`, and `write()` ends with `os.Remove(archive.live)`. Nothing downstream reports it — `dotf doctor`'s check looks for a *file without an entry*, the opposite state. | **UNTESTED** — `TestCheckThreadFileRefusesAFileAnotherKeyHolds` exercises the pure function only; no command-level test puts a colliding live file in front of a `--finished` write | code + tests: check the live file (`CheckThreadFile(prevLive, res.Key)`) before removing it, and refuse `moveThreadFile` for a live file whose heading names another key; add a named case per path, e.g. `TestHandoffWriteFinishedRefusesALiveFileAnotherKeyHolds` |
| **Major** | **REAL** | verification artifact | `verification.md` is the untouched scaffold: the three Evidence lines still read `Criterion 1 -> commit '<hash>' / test '<name>'`, Test status is placeholders, and all three **Promotion candidates** lines are unanswered. `dotf spec archive --help` states the third pre-flight reads exactly those lines, refuses an unanswered one, and has **no flag that skips it** — so this spec cannot archive whatever the code does. Lesson-603 was written but is not named from the spec. | File read at this head; the archive CLI's own documented pre-flight | n/a (artifact, not behaviour) | verification.md + tasks.md (note: `tasks.md` is in the **contract set** — ticking it invalidates this review, so it belongs to the re-review round) |
| **Major** | **REAL** | docs / SSOT drift | The change retires a shape that two build/operate documents still teach. `docs/adr/adr-060-…md:34` records the park as "archive `memory/threads/<branch>.md` to `memory/archive/` while pruning `MEMORY.md`", and `docs/runbooks/guide-cross-agent-memory.md:98` tells the operator the finished remedy is "move the file to `archive/`". Following either now reproduces this spec's own defect (an entry whose `threads/<key>.md` pointer does not resolve from the archive), and the runbook omits the `--finished` verb the doctor warning in this very diff now recommends. PR #3518's `## Knowledge` declares "ADR: none" and "Runbook: none". | Both lines read at this head; the new shape is asserted by `TestArchiveParkedThreadUsesTheDatedArchive`'s `archive/<key>.md` non-existence check | n/a (docs) | docs: amend ADR-060's park step and the runbook remedy; correct the PR's `## Knowledge` lines |
| Minor | REAL | test traceability (AC2) | Byte-identity of the archived entry is pinned on the `--finished` path only. Mutating `takeThreads` to append `> archived by a park` to every taken entry turned `TestArchiveThreadDropsTheEntryAndReturnsTheBlockItWouldHavePlaced` and `TestHandoffWriteFinishedArchivesTheThread` red while `TestArchiveParkedThreadUsesTheDatedArchive`, `TestParkArchivesTheBranchThread` and `TestLaneParkInPlaceAndTipVerification` stayed **green** — the park tests assert `strings.Contains`, not equality. Reverted, tree clean. Low product risk because both paths now share one extraction; the criterion is still only half-proven. | Mutation run at this head (output above) | park-side byte-identity **UNTESTED**; fix by asserting the exact dated-archive text in `TestArchiveParkedThreadUsesTheDatedArchive` | tests |
| Minor | THEORETICAL | silent no-op (AC4 spirit) | With no vault resolving, `DoneOptions.VaultPath` is `""` and `archiveBranchThreads` returns an empty result with a nil error, so a park prints nothing at all and the lane's thread stays in MEMORY.md forever — the same "silently leaves a stale entry" cost the proposal's *Why* names. Pre-existing (the old `ParkLane` guarded on `vaultPath != ""` too), and the field's doc says "Empty archives nothing", so this is a named design choice rather than a bug; one line distinguishing "no vault" from "no threads" would close it. | `done.go:265-268` read; `lane park` / `worktree done` / both sweep callers all pass `vault.ResolveVault()`, which returns `""` on an unconfigured machine | UNTESTED (no test covers the empty-vault *report*, only `TestParkWithoutAVaultLeavesMemoryAlone`, which asserts silence) | code |
| Minor | REAL | quality thresholds | `ArchiveParkedThread` is 52 lines with cyclomatic complexity **17** (gocyclo-measured); `takeThreads` is 37 lines / CC 12. AGENTS.md states <40 lines and CC <10; the rubric's C band starts above CC 15. Context that keeps this Minor: the deleted `ArchiveAndPruneThread` was already 51 lines, the deleted `pruneThreadFromMemory` (31 lines) is gone, so the area's total complexity fell, and `.golangci.yml` deliberately exempts `internal/mem` from `gocyclo` ("older outliers, up to 32, that this change does not take on"). | `awk` line count and gocyclo output at this head | n/a | code: extract the dated-archive append loop out of `ArchiveParkedThread` |
| Minor | REAL | output surface | The park report goes to **stderr** for `lane park` and `worktree done` but to **stdout** for `worktree sweep` (`worktree.go:488`), and for sweep the returned error then reaches cobra on top of the hand-built `ERROR: ` prefix, so a failure prints `Error: ERROR: lane … `. Cosmetic; it is pinned by `TestReportSweepThreadsFailsOnAnArchiveFailure`, so any change re-pins that test. | call sites read at this head | `TestReportParkThreads*`, `TestReportSweepThreadsFailsOnAnArchiveFailure` (cover the text, not the stream) | code |
| Question | — | AC1 wording | AC1 and the *What* list `dotf worktree done` among the paths that go through the one function. That holds for a **lane** (`Done` delegates to `Park` when the lease names a lane, `done.go:59-61`); a non-lane `worktree done` tears the worktree down and archives no thread, exactly as before this change. Confirm the reading is intended rather than a fifth entry point being missed. | `done.go:53-101`, single non-test `worktree.Park` caller | n/a | spec (answer in `verification.md`, no contract edit needed) |

Nothing found in the diff's security columns: no shell built from a thread key (all git calls are `exec.Command` arg vectors),
no new path accepted from user input (both new `os.ReadFile` sites carry `#nosec G304` with a stated reason and are confined
beside `MEMORY.md`; `threadFileName` refuses anything that could leave `threads/`), no secret handling, no new dependency,
`GOOS=windows go vet` clean, and the memory lock (`LockHandoffMemory`, bounded 10s) is taken by the park as it is by the writer.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | **D** | One-arm implementation of the right design, but a demonstrated defect present: F1 destroys a live thread's file that base refused to touch. |
| Verification | **C** | `features.json`'s four commands are non-vacuous and green here, but `verification.md` holds no evidence at all and its promotion lines are unanswered, which the archive pre-flight refuses. |
| Scope | **B** | Diff matches the proposal; the side changes (`releaseParkedLane` argument order, doctor and budget wording, `VaultPath` threaded through callers) are all named in it, and no test was deleted or weakened. |
| Reliability | **C** | Interruption order stated, tested and honoured, dry runs write nothing, a failed archive is reported and non-zero; F1's unchecked destructive remove and F5's silence are the gaps. |
| Maintainability | **C** | Lint 0 issues, one shared extraction, honest comments; but a 52-line CC-17 function above the repo's own thresholds. |
| Handoff-readiness | **D** | Lesson 603 captured (good), but `verification.md` is scaffold, every Closing box is unticked, and ADR-060 plus the memory runbook still teach the retired shape. |

Two Ds, and a REAL Blocker: FAIL on both the aggregation rule and the severity path.

### Verdict

**FAIL**

### Recommended next steps

Ordered as the minimum set that flips this to PASS. F1 and F3 are outside the contract set and can be applied now;
F2's `tasks.md` half is **inside** it, so it lands with the re-review rather than before it.

1. **F1 (code + tests, blocking).** Make the archive refuse to remove or rename a `threads/<name>.md` whose heading names a
   different key — in `handoffWrite.write`/`run` for the `--finished` path and in `mem.moveThreadFile` for the park — and pin
   each with a named test. Reproduce first with the two-command probe in F1's evidence column; the base commit's refusal is
   the behaviour to restore.
2. **F3 (docs).** Amend ADR-060's park step and `docs/runbooks/guide-cross-agent-memory.md`'s "Finished" remedy to name
   `--finished`, `archive/superseded-handoffs-<date>.md` and `archive/threads/<key>.md`; fix the PR's `## Knowledge` lines.
3. **F2 (verification.md; tasks.md with the re-review).** Fill the Evidence map (AC1 → `TestArchiveParkedThreadUsesTheDatedArchive`
   …, AC2 → `TestArchiveThreadDropsTheEntry…` + the park-side assertion from step 4, AC3 → same park test +
   `TestReportParkThreadsNamesArchivedAndKept`, AC4 → `TestParkReportsAFailedArchiveAndStillParks` +
   `TestReportSweepThreadsFailsOnAnArchiveFailure`), record the commands run in this session's shape, and answer the three
   Promotion candidates — lesson 603 is a `yes: docs/lessons/lesson-603-…`, ADR and pattern need a reason either way.
   `dotf spec archive` refuses before that line is answered, no flag excepted.
4. **F4 (tests).** Assert the park's exact dated-archive text so AC2's byte-identity is proven on both paths.
5. **Then re-run `dotf spec review MEMORY-020-one-archive-shape`.** Step 3 touches `tasks.md`, a contract file, so this
   verdict cannot stand over it by design; the next round is the mechanism.
6. Tracked, non-blocking: F5 (name the empty-vault skip), F6 (split `ArchiveParkedThread`), F7 (one stream for the park
   report), and the AC1 wording question.

`dotf spec archive` is **not advisable** in this state: it would refuse on three independent counts (verdict FAIL,
unanswered promotion lines, and — once `tasks.md` is ticked — a stale contract).
