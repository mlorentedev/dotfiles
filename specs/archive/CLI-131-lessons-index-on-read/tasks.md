---
tags: [spec, tasks, templates]
created: "2026-10-16"
---

# Tasks - CLI-131-lessons-index-on-read

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch from main in its own worktree (feat/cli-131-lessons-index)
- [x] `proposal.md` is complete and acceptance criteria are testable

## Implementation

One PR: the change is a removal plus a render command, and splitting it would leave `fmt --check` red on main in between.

- [x] [AC1] `lessons.Index` and `dotf lessons index`, tests first
- [x] [AC2] [AC4] `Plan` and `danglingLinks` stop reading and requiring an index; the title-seed path (`IndexTitles`, `seedTitle`) goes, measured at 0 untitled lessons
- [x] [AC3] `StrayIndexes`; `--check` reports a committed index and the fix, `fmt` leaves it alone
- [x] [AC5] `dotf init` stops scaffolding the index
- [x] [AC6] The session-start staleness warning reads the newest lesson
- [x] [AC7] `git rm docs/lessons/_index.md`; reword the stale pointers (docs README, legacy stub, lane prompt, pre-commit name and comment)
- [ ] Follow-up, ticketed as #2659, not in this PR: the vault skill records that still say `dotf lessons fmt` regenerates `_index.md` (architecture-session, autopilot, crystallize, handoff, insights, place-knowledge, spec) and the comment in `.github/workflows/repo-hygiene.yml` (owner-lands path)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder
