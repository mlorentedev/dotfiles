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

## Round 2 review - deletions audited

The reviewer found that the port had deleted AI-054 coverage rather than
adapting it. Every removal in `prbudget_test.go` was re-examined against
`origin/main` under a retention bar: a test may only go if (a) the failure it
detected cannot recur, and (b) the remaining suite still proves that failure is
caught. Neither held for any of them, so all were restored.

| Removed by the port | Failure it detected | Why it stayed | Status |
| --- | --- | --- | --- |
| `AFileWithoutAClipFailsClosed` | `max_model_tokens` absent from `.pr_agent.toml` → the reserve silently becomes 0 | The failure is exactly the class of bug this PR is about; condition (a) is false | Restored verbatim |
| `HalfAReviewerFailsClosed`, second half | workflow present, toml missing → the guard trusts half a configuration | A half-configured checkout is reachable at any time; `prbudget.go:130` had become dead | Restored verbatim |
| `AI-048` fixture comment | n/a (documentation) | It misnames the spec the fixtures came from | Corrected to `AI-054` |
| `// Five:` rationale on `TheCurrentConfigurationFails` | n/a (rationale) | The count is what the assertions key off; without it the next editor edits the number | Restored, with main's `t.Fatalf` wording |
| Equal-window negative assertion in `AStepThatLowersItsOwnWindowIsMeasuredAgainstIt` | a step window *equal* to the file's must change nothing | The rewrite kept only the positive case, so the test could no longer fail if `answerReserve` over-tightened | Restored, main's verbatim body |

The retention bar is also why `TheReserveIsTheLargestCapNotTheFirst` and
`AnUnparseableFileFailsClosed` were **not** counted as restorations: both
already existed on `main`, and the port had merely relocated them.

## Closing

- [x] Every acceptance criterion is covered by at least one test
- [x] Every acceptance criterion has a matching `features.json` entry with a non-vacuous verification command
- [x] Type checks pass (`go vet ./internal/harness`)
- [x] Lint passes (gofmt clean on LF-normalised copies; the CRLF working tree is a `core.autocrlf` artifact, not a formatting defect)
- [x] No unrelated changes in the diff (no `.pr_agent.toml`, pool, workflow or bats changes)
- [x] `verification.md` filled in
- [x] Round 2 review findings dispositioned (two Majors fixed, three Minors and one Question declined in scope with reasons in `verification.md`)
- [ ] PR opened referencing this spec folder