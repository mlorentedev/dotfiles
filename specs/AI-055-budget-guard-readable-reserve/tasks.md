---
tags: [spec, tasks]
created: "2026-11-11"
id: "AI-055-budget-guard-readable-reserve"
type: spec-tasks
status: active
owner: manu
---

# Tasks - AI-055-budget-guard-readable-reserve

> TDD order. Tests were written against the pre-fix implementation and observed
> red before the port landed; the mutation check in `verification.md` re-runs
> that observation against `main`'s implementation.

## Setup

- [x] Branch created from main: `fix/budget-guard-readable-reserve` (worktree, issue lease on #3301)
- [x] `proposal.md` complete and acceptance criteria testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC1] Failing test `AnAnswerCapItCannotReadIsAGapNotAPass` — a cap of `Three thousand` reports a GAP naming the step and the raw value, and no clip line prints
- [x] [AC1] Implement `answerReserve(cfg) (int, []CheckLine)`: a declared-but-unreadable cap is a problem, not a zero
- [x] [P] [AC2] Failing test `AWorkflowThatNeverDeclaresTheAnswerCapSaysSo` — no step declaring the cap is a GAP naming the variable
- [x] [AC2] `answerReserve` tracks whether any step declared a cap at all and reports when none did
- [x] [AC3] Keep the largest-cap property: `TheReserveIsTheLargestCapNotTheFirst` (restored from AI-054) still passes on `answerReserve`
- [x] [P] [AC4] Failing test `AStreamedStepIsReportedAsALowerBound` — a step with `LITELLM__FORCE_STREAMING: "true"` gets the `(streamed: …)` suffix, the non-streamed step does not
- [x] [AC4] Implement `declaresStreaming(env)` / `streams` on `reviewStep` and the OK-line suffix in `stepCeiling`
- [x] [P] [AC5] Failing test `AStepWithoutItsOwnTimeoutSaysSo` — a step with no `timeout-minutes` reports that, not `in a 0 s ceiling`
- [x] [AC5] Add the `s.minutes <= 0` branch ahead of the ceiling comparison
- [x] Reconcile with main: `orZero` renamed to main's `orInt`, `(AI-048)` comment corrected to `(AI-054)`, and the two AI-054 tests absent from the old lineage restored
- [x] Refactor: `largestOutputCap` deleted, its contract absorbed by `answerReserve`

## Closing

- [x] Every acceptance criterion is covered by at least one test
- [x] Every acceptance criterion has a matching `features.json` entry with a non-vacuous verification command
- [x] Type checks pass (`go vet ./internal/harness`)
- [x] Lint passes (`gofmt -d` clean on both touched files)
- [x] No unrelated changes in the diff (no `.pr_agent.toml`, pool, workflow or bats changes)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder