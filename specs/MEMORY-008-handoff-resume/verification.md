---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - MEMORY-008-handoff-resume

## Evidence

- [x] AC1 -> slice 3: `TestResumeIsAPureRenderOfTheThreadFile`
- [x] AC2 -> slices 1 and 3: `TestResumeRendersVerifyAndAwaiting`, `TestResumeListsCandidatesWhenTheThreadIsAmbiguous`
- [ ] AC3 -> slice 4: `TestSessionStartResumeBySource`
- [x] AC4 -> slice 2: `TestAMemoryShapedLikeDotfilesStaysUnderBudgetWithTheNewFields`, `TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget`
- [ ] AC5 -> slice 5: the `handoff` and `catchup` records
- [ ] AC6 -> slice 5: the compiled doctrine, `--check` and the 8,000-character budget test

Each slice PR fills its own section below and nothing else.

## Slice 0 evidence (prerequisite bugs, each on its own issue)

- #1620, `session-end` replaced an authored journal: PR #1701. `TestSessionEndLeavesAnAuthoredJournalByteIdentical` fails on main and passes with the fix.
- #1606, `handoff-write` keyed a thread from another repository: PR #1702. `TestHandoffThreadRefusesAKeyFromAnotherRepository` and `TestMemHandoffWriteFromAnotherRepositoryTouchesNoThread` fail on main (the first cannot build there, the second finds the other session's thread replaced) and pass with the fix.
- #1651, the legacy block stays in front: PR #1703. `TestWriteThreadMovesTheUnthreadedBlockIntoALegacyThread`, `TestWriteThreadMigratesEvenWhenItsOwnThreadIsUnchanged`, `TestWriteThreadNamesAnUndatedLegacyBlockAndNeverCollides` and `TestMemHandoffWriteDryRunShowsTheLegacyMigration`.

## Slice 1 evidence

- `ParseThread` and `ThreadLabels` in `cli/internal/mem/thread.go`.
  - `TestParseThreadReadsEveryCanonicalLabel`: all eight labels are read.
  - `TestParseThreadReadsTheImprovisedLabelsAsTheirFields`: `Verify at start` is read as `Verify`, and `Judgment calls left open` as `Awaiting Manu`.
  - `TestParseThreadKeepsAnUnknownLabel`: an unknown label is kept. A qualified canonical label counts as unknown, because its qualifier is content.
  - `TestParseThreadDropsNoLine`.
  - `TestParseThreadReadsEitherFormOfTheDateAndJournal`.
- `handoff-write` names a missing or empty `Next action`, and every unknown label, on stderr. It still writes the body. Tests: `TestThreadWarningsNameOnlyWhatIsMissingOrUnknown`, `TestMemHandoffWriteNamesWhatTheBodyLacksAndStillWrites`.
- **Mutation checks**, each reverted:

  | Mutation | Test that went red |
  |---|---|
  | aliases ignored | the improvised-labels test |
  | continuation lines dropped | two tests |
  | an empty `Next action` accepted | the warnings test |

- **Found along the way: `handoff-write` narrowed the mode of the file it replaced.** `os.CreateTemp` creates the temp file 0600, and the rename carried that mode onto MEMORY.md. On 2026-09-25, 9 of the vault's 21 MEMORY.md files were 0600, the ones this command had written; the rest were 0664.
  - Fix: the temp file takes the replaced file's mode before the rename, as `harness bind` does for settings files. Test: `TestMemHandoffWriteKeepsTheFileMode`, which failed at 600 before the fix.
  - The write now lives in `replaceMemoryFile`. That brings `newMemHandoffWriteCmd` from 17 (after #1711 and this slice's warnings) down to 14.
- Rebased onto #1711. Its `TestMemHandoffWriteWithoutAgentSaysWhatItAlwaysSaid` expected an empty stderr with a body that has no `Next action`, which this slice now warns about. The test now passes a body with one, so it still checks only what `--agent` could add.
- **Probe over every thread in the vault's `10_projects/*/memory/MEMORY.md`** (a throwaway test, not committed):
  - 34 threads parsed, with 0 lines lost.
  - 1 thread has no `Next action`.
  - 9 fields carry one of 7 labels outside the set. The most frequent are `Trap` and `Decisions (Manu, 2026-09-24)`, twice each. These are what the new warning will name.

## Slice 2 evidence

- The store is in `cli/internal/mem/thread_store.go`, and the design is in ADR-056.
  - `StoreThreadAs` writes the body to `memory/threads/<key>.md` and puts the derived entry (Updated, the first line of the Next action, the Awaiting Manu count, Journal, pointer) under the heading in `MEMORY.md`. Tests: `TestStoreThreadWritesTheBodyToItsFileAndADerivedEntry`, `TestTheDerivedEntryCutsALongNextAction` (entry at most 350 B), `TestTheDerivedEntryCapsEveryQuotedField` (every quoted field at its cap: at most 450 B).
  - Ownership and forks are shared with `WriteThreadAs` through `threadOwner`. Test: `TestStoreThreadForksAroundAnotherAgentsEntry`.
  - Migration on the next write: `TestStoreThreadMovesAnInlineBlockAndLeavesTheOthersByteIdentical`.
  - Key to file: `TestThreadFileNames`. The one live key with a `/` (ts-bridge `fix/ux355-init-headscale-control-plane`, 74 thread headings checked across the vault) maps to `fix-ux355-init-headscale-control-plane.md`. Collisions: `TestCheckThreadFileRefusesAFileAnotherKeyHolds`. A pointer in a body is refused: `TestStoreThreadRefusesAPointerInTheBody`.
- `handoff-write` writes the thread file first, then `MEMORY.md`, under the existing lock. A new file takes `MEMORY.md`'s mode. Tests: `TestMemHandoffWriteStoresTheThreadInItsOwnFile`, `TestMemHandoffWriteDryRunStoresNothing`.
- **Budget (#1477):** `CheckMemoryBudget` refuses only a write that grows the file past 25,000 B, and names the three oldest threads. Tests: `TestCheckMemoryBudget`, `TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget`.
- **Found along the way: `session-end` reads the thread too.** Its fallback journal archives the block from `MEMORY.md`, so after the store it would have archived the entry. It now reads through `threadText`. Test: `TestSessionEndArchivesTheStoredThreadNotItsEntry`. Lesson 507.
- **Probe on a copy of the real dotfiles `MEMORY.md`** (2026-10-20, a throwaway test, not committed). Each of its 8 threads was written again with a `Verify` line and two `Awaiting Manu` items:

  | | Bytes |
  |---|---|
  | as it is | 23,423 |
  | inline, with the fields | 24,488 |
  | stored, with the fields | 14,516 |
  | budget | 25,000 |

- **Mutation checks**, each reverted:

  | Mutation | Tests that went red |
  |---|---|
  | budget check off | `TestCheckMemoryBudget`, `TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget` |
  | over-budget writes that shrink also refused | `TestCheckMemoryBudget` |
  | collision check off | `TestCheckThreadFileRefusesAFileAnotherKeyHolds` |
  | Next action not cut | `TestTheDerivedEntryCutsALongNextAction` |
  | resolver ignores the pointer | `TestThreadTextFollowsThePointer`, `TestSessionEndArchivesTheStoredThreadNotItsEntry` |

## Slice 3 evidence

- `dotf mem resume [--thread K] [--agent A] [--format prompt|context] [--memory PATH]` in `cli/internal/cmd/mem_resume.go`. The render is in `cli/internal/mem/resume.go`.
  - `RenderResume` is pure: `TestResumeIsAPureRenderOfTheThreadFile` renders a fixture twice per format and compares both renders with `testdata/resume/{context,prompt}.golden`. An unknown format is refused.
  - Next action, Verify and Awaiting Manu render first. Verify commands are listed one per line, and one outside the read-only allowlist is marked unverified (`TestResumeRendersVerifyAndAwaiting`). The allowlist matches on a word boundary, and any chaining, pipe, redirection or substitution fails it (`TestReadOnlyCommand`).
  - `SelectThread` takes the agent's fork of the key, then the key unless another agent's stamp is on it. Otherwise it picks none and returns every thread key (`TestResumeListsCandidatesWhenTheThreadIsAmbiguous`). The command lists those keys, each with its `--thread` flag (`TestMemResumeListsCandidates`).
  - `ThreadText` reads the stored file when the entry points at one, and the inline block otherwise. It also returns the source path (`TestThreadTextForKey`). Round trip through the command: `TestMemResumeRendersWhatHandoffWriteStored`.
- `--memory` defaults to the checkout project's vault `MEMORY.md`, through `mem.ProjectMemory`. That helper was lifted out of `session-end`, which resolved the same path inline, so the layout is derived in one place.
- The read-only allowlist leaves out generic file readers (`cat`, `grep`, `head` and the like). They take any path, so `cat <secrets file>` would render as safe.
- **Live run** on a copy of the real dotfiles `MEMORY.md` (inline blocks, 2026-10-21). `--thread feat-plat003-close@macmini --agent claude` rendered that thread's Next action. `--thread nope` listed all 8 threads.
- **Mutation checks**, each reverted:

  | Mutation | Test that went red |
  |---|---|
  | chaining allowed | `TestReadOnlyCommand` |
  | prefix without a word boundary | `TestReadOnlyCommand` |
  | another agent's thread chosen | `TestResumeListsCandidatesWhenTheThreadIsAmbiguous` |
  | this agent's fork ignored | `TestResumeListsCandidatesWhenTheThreadIsAmbiguous` |
  | Verify not rendered first | `TestResumeIsAPureRenderOfTheThreadFile` |

  The first attempt at the "another agent's thread" mutation did not compile. The harness printed no failing test, which read as a surviving mutant (lesson 508).

## Slice 4 evidence

## Slice 5 evidence

## Decisions made during implementation

- Slice 2: the AC4 fixture is synthetic, shaped like the real file. Copying the vault's `MEMORY.md` into a public repository would publish it. The real file was probed instead, and the probe's figures are recorded above.
- Slice 2: a write that keeps or shrinks a file already over the budget is allowed, so that file can still be brought back under (ADR-056 §7).

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`?
- [ ] ADR-worthy decision for the repo's `docs/adr/`? (the per-thread store changes the layout of every project's `memory/`)
- [ ] New pattern candidate for `00_meta/patterns/`?

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/MEMORY-008-handoff-resume/` -> `specs/archive/MEMORY-008-handoff-resume/`
- [ ] Bitácora board ticket moved to Done / closed with the closing PR (ADR-018)
- [ ] Promotions above executed
