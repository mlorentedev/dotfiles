---
tags: [spec, verification]
created: "2026-11-11"
id: "AI-055-budget-guard-readable-reserve"
type: spec-verification
status: active
owner: manu
---

# Verification - AI-055-budget-guard-readable-reserve

Repository commit: `af73427a` (origin/main) + this branch's working tree.
Toolchain: `go version go1.25.6 windows/amd64`. All commands run from `cli/`
unless the path says otherwise.

## AC1 - an unreadable answer cap is a GAP

```
go test ./internal/harness -run TestCheckReviewBudget_AnAnswerCapItCannotReadIsAGapNotAPass -count=1 -v
```

PASS. The step declaring `CONFIG__MAX_OUTPUT_TOKENS: "Three thousand"` produces
`the answer cap in <step> is "Three thousand", which this check cannot read as a
number, so nothing says how much room the prompt must leave`, and no
`prompt clip in …` line is emitted for that run.

## AC2 - a workflow that declares no cap says so

```
go test ./internal/harness -run TestCheckReviewBudget_AWorkflowThatNeverDeclaresTheAnswerCapSaysSo -count=1 -v
```

PASS. Message: `no review step declares CONFIG__MAX_OUTPUT_TOKENS, so the room
the prompt must leave for the answer is unknown`.

## AC3 - the reserve is still the largest cap

```
go test ./internal/harness -run TestCheckReviewBudget_TheReserveIsTheLargestCapNotTheFirst -count=1 -v
```

PASS. Restored from AI-054: two steps declaring 40000 and 60000 under
`max_model_tokens = 150000` report `answer cap 60000` as the reserve. This test
was absent from the retired branch's lineage and would have been lost by a
straight copy.

## AC4 - a streamed step is a lower bound

```
go test ./internal/harness -run TestCheckReviewBudget_AStreamedStepIsReportedAsALowerBound -count=1 -v
```

PASS, plus the real repository:

```
go run ./cmd/dotf harness check --repo-root .. | grep 'ceiling'
[check] OK -> PR-Agent on Anthropic, drawn first: 720 s of attempts in a 780 s ceiling
[check] OK -> PR-Agent: 360 s of attempts in a 720 s ceiling (streamed: ai_timeout bounds the gap between chunks, not the answer, so this is a lower bound on the step's wall clock)
[check] OK -> PR-Agent on Anthropic, after NaN published nothing: 720 s of attempts in a 780 s ceiling
```

The two non-streamed steps are unchanged; only the NaN step, whose call streams,
carries the caveat.

## AC5 - a step with no `timeout-minutes`

```
go test ./internal/harness -run TestCheckReviewBudget_AStepWithoutItsOwnTimeoutSaysSo -count=1 -v
```

PASS. Message names the missing `timeout-minutes` instead of reporting
`in a 0 s ceiling`.

## AC6 - the committed configuration still passes, AI-054 paths unchanged

```
go test ./internal/harness -run TestCheckReviewBudget -count=1 -v   # 19 tests, 19 PASS, 0 FAIL
go test ./internal/harness -count=1                                  # whole package, ok 13.151s
go vet ./internal/harness                                            # exit 0
go run ./cmd/dotf harness check --repo-root ..                       # exit 0
gofmt -d internal/harness/prbudget.go internal/harness/prbudget_test.go   # no output
```

`TheRepositoryPassesItsOwnGuard` covers the committed `.pr_agent.toml` and
workflow; `TheGuardLeavesTheJobCeilingToTheBatsTest`,
`AWorkflowWithoutAReviewStepFailsClosed`, `HalfAReviewerFailsClosed`,
`ACheckoutWithoutAReviewerSaysNothing`, `AFileWithoutAClipFailsClosed` and
`AnUnparseableFileFailsClosed` all still pass, so every fail-closed path from
AI-054 is intact.

## Mutation check - do the new tests munch?

Reverting **only** `cli/internal/harness/prbudget.go` to `origin/main` and
re-running the suite fails exactly the four tests that describe new behaviour:

```
--- FAIL: TestCheckReviewBudget_AStepWithoutItsOwnTimeoutSaysSo
--- FAIL: TestCheckReviewBudget_AnAnswerCapItCannotReadIsAGapNotAPass
--- FAIL: TestCheckReviewBudget_AWorkflowThatNeverDeclaresTheAnswerCapSaysSo
--- FAIL: TestCheckReviewBudget_AStreamedStepIsReportedAsALowerBound
FAIL
```

The other fifteen, including both restored AI-054 tests, stay green against the
old implementation, which is what separates "new behaviour" from "renamed
behaviour". The implementation was restored afterwards and the suite re-run
green.

## What this does not prove

- The streamed step's actual wall clock. The line now says it is a lower bound;
  measuring it is #3302.
- The workflow/job-level `CONFIG__MAX_MODEL_TOKENS` expression skip, still open
  on #3301.
- Anything about the bats job-ceiling test, which this PR does not touch.