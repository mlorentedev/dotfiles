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
go test ./internal/harness -run TestCheckReviewBudget -count=1 -v   # 20 tests, 20 PASS, 0 FAIL
go test ./internal/harness -count=1                                  # whole package, ok 14.998s
go vet ./internal/harness                                            # exit 0
go run ./cmd/dotf harness check --repo-root ..                       # exit 0 (no harness drift)
```

`TheRepositoryPassesItsOwnGuard` covers the committed `.pr_agent.toml` and
workflow; `TheGuardLeavesTheJobCeilingToTheBatsTest`,
`AWorkflowWithoutAReviewStepFailsClosed`, `HalfAReviewerFailsClosed` (both
halves), `ACheckoutWithoutAReviewerSaysNothing`, `AFileWithoutAClipFailsClosed`
and `AnUnparseableFileFailsClosed` all still pass, so every fail-closed path
from AI-054 is intact.

Formatting: this checkout runs with `core.autocrlf=true`, so the working tree is
CRLF and `gofmt -l` names both files unconditionally. Normalising to LF first is
the check that means something:

```
gofmt -l <LF copies of prbudget.go and prbudget_test.go>   # no output
```

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

The other sixteen, including both restored AI-054 tests, stay green against the
old implementation, which is what separates "new behaviour" from "renamed
behaviour". The implementation was restored afterwards and the suite re-run
green.

## Round 2 review disposition

Reviewer `nan/glm5.3-flash` (verdict FAIL, against `06f3100b`). Both Majors were
real regressions introduced by the port, not stale claims.

| Finding | Severity | Disposition |
| --- | --- | --- |
| `7d0c2b33` deleted `AFileWithoutAClipFailsClosed` and the toml-missing half of `HalfAReviewerFailsClosed`, leaving `prbudget.go:259` and `:130` executed 0 times | Major | Fixed. Both restored verbatim from `origin/main`; the coverage they carry is back. |
| AC6 named a test that no longer existed | Major | Fixed. AC6 above names only tests in the tree, with this round's counts. |
| `declaresStreaming` matches key presence, so `"false"` and `'[]'` also earn the caveat | Minor | Declined in scope. The caveat is a lower bound either way; a wrong-value key means the call may not stream, and under-reporting there is the safe direction. Tracked as #3302. |
| A cap of `"0"` or `"-5"` is a number but is reported as "cannot read as a number" | Minor | Declined in scope. The sentence is imprecise for a non-positive literal, the outcome (GAP, not pass) is right, and `checkTokenClip` already rejects the config one line earlier. Tracked as #3302. |
| A step missing both `CONFIG__AI_TIMEOUT` and `timeout-minutes` reports only the timeout gap | Minor | Declined in scope. Reporting one gap per step is the existing AI-054 shape; the second surfaces once the first is fixed. |
| A cap declared in `.pr_agent.toml` is invisible to AC2 | Question | Answered: AC2 covers the workflow's own declaration; the toml cap is what `checkTokenClip` bounds, and it is required (`AFileWithoutAClipFailsClosed`). They are different ceilings, deliberately. |

Also restored, because the port had silently reverted them rather than removed
them on purpose: the `AI-054` fixture comment, the `// Five:` rationale on
`TheCurrentConfigurationFails`, and main's equal-window negative assertion in
`AStepThatLowersItsOwnWindowIsMeasuredAgainstIt`. See `tasks.md` for the
retention bar applied to every deletion in this PR.

## What this does not prove

- The streamed step's actual wall clock. The line now says it is a lower bound;
  measuring it is #3302.
- The workflow/job-level `CONFIG__MAX_MODEL_TOKENS` expression skip, still open
  on #3301.
- Anything about the bats job-ceiling test, which this PR does not touch.