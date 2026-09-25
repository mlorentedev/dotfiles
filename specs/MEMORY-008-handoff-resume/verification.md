---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - MEMORY-008-handoff-resume

## Evidence

- [ ] AC1 -> slice 3: `TestResumeIsAPureRenderOfTheThreadFile`
- [ ] AC2 -> slices 1 and 3: `TestResumeRendersVerifyAndAwaiting`, `TestResumeListsCandidatesWhenTheThreadIsAmbiguous`
- [ ] AC3 -> slice 4: `TestSessionStartResumeBySource`
- [ ] AC4 -> slice 2: `TestTheRealDotfilesMemoryStaysUnderBudgetWithTheNewFields`, `TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget`
- [ ] AC5 -> slice 5: the `handoff` and `catchup` records
- [ ] AC6 -> slice 5: the compiled doctrine, `--check` and the 8,000-character budget test

Each slice PR fills its own section below and nothing else.

## Slice 0 evidence (prerequisite bugs, each on its own issue)

- #1620, `session-end` replaced an authored journal: PR #1701. `TestSessionEndLeavesAnAuthoredJournalByteIdentical` fails on main and passes with the fix.
- #1606, `handoff-write` keyed a thread from another repository: branch `fix/handoff-thread-from-project`. `TestHandoffThreadRefusesAKeyFromAnotherRepository` and `TestMemHandoffWriteFromAnotherRepositoryTouchesNoThread` fail on main (the first cannot build there, the second finds the other session's thread replaced) and pass with the fix.
- #1651, the legacy block stays in front: pending.

## Slice 1 evidence

## Slice 2 evidence

## Slice 3 evidence

## Slice 4 evidence

## Slice 5 evidence

## Decisions made during implementation

-

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`?
- [ ] ADR-worthy decision for the repo's `docs/adr/`? (the per-thread store changes the layout of every project's `memory/`)
- [ ] New pattern candidate for `00_meta/patterns/`?

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/MEMORY-008-handoff-resume/` -> `specs/archive/MEMORY-008-handoff-resume/`
- [ ] Bitácora board ticket moved to Done / closed with the closing PR (ADR-018)
- [ ] Promotions above executed
