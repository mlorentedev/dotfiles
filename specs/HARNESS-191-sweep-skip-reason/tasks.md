---
tags: [spec, tasks, templates]
created: "2026-11-11"
id: "HARNESS-191-sweep-skip-reason"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-191-sweep-skip-reason

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `fix/3263-sweep-skip-reason` (from `4e0281f6`, rebased onto main before review)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] [AC3] Red: the four `TestParkSingleNamedLane_Outcomes` subtests read a reason from `parkSingleNamedLane` and assert it (dirty tree, forge error, refused detach, parked lane with no reason); `TestSweepReportsALaneItDidNotPark` asserts `LaneSkips` through `SweepWithRunner` on a real repository with a held index lock
- [x] [AC1] [AC2] [AC3] [AC4] Green: `parkSingleNamedLane` returns `(parked, why)`; `laneLanded` carries the landing decision; `gitFailure` keeps git's first line; `SweepReport.LaneSkips`; the loop appends on a non-park; `cmd/worktree.go` prints `LaneSkips` with the other skips
- [x] [AC4] `TestWorktreeSweepPrintsWhyALaneWasNotParked`: the printed line through the real command, on a lane landed by ancestry with a held index lock
- [x] Lesson 584: the push gate counts insertions plus deletions; judge the WIP limit before the work exists

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass (`go vet`, both GOOS)
- [x] Lint passes (pinned golangci-lint, both GOOS)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

`features.json` beside this file maps AC1..AC4 to the tests above; `state: verified` is written only with the harness's captured evidence.
