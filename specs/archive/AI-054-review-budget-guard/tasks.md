---
tags: [spec, tasks, templates]
created: "2026-11-10"
id: "AI-054-review-budget-guard"
type: spec-tasks
status: active
owner: manu
---

# Tasks - AI-054-review-budget-guard

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `feat/review-budget-guard`
- [x] Issue #3281 opened and linked by `dotf spec init`
- [x] `proposal.md` filled before any code

## Parse the configuration

- [x] `[AC4]` Red: a test builds a repo whose workflow parses but has no review step, and asserts the check reports a problem rather than passing.
- [x] `[AC4]` Red: a test builds a repo with a reviewer but an unparseable `.pr_agent.toml` / workflow, and asserts a problem.
- [x] `[AC4]` Red: a test builds a repo with neither file, and asserts no line is reported.
- [x] Green: `parseReviewToml` reads `max_model_tokens`, `custom_model_max_tokens` and `fallback_models` through `go-toml/v2`; `parseReviewWorkflow` reads each job's `timeout-minutes` and each step's `name`, `uses`, `timeout-minutes` and `env` through `go.yaml.in/yaml/v3`.
- [x] `[AC4]` A review step is one whose `uses` names the pr-agent image; the guard fails closed when none is found.

## Token clip

- [x] `[AC1]` Red: a test with `max_model_tokens` equal to `custom_model_max_tokens` reports a problem naming both numbers and the output cap.
- [x] `[AC1]` Green: `checkTokenClip` asserts `max_model_tokens + largestOutputCap < custom_model_max_tokens`.
- [x] `[AC1]` The largest output cap is the maximum `CONFIG__MAX_OUTPUT_TOKENS` across review steps, so a step that declares a bigger answer reserves more of the window.
- [x] `[AC1]` A review step that re-declares `CONFIG__MAX_MODEL_TOKENS` is checked too, because the workflow's env overrides the file — a fix that lowers only the toml clips nothing.

## Step ceilings

- [x] `[AC2]` Red: a test with a 480 s ceiling, chain 1, two attempts and `ai_timeout` 360 reports a problem showing `720 > 480`.
- [x] `[AC2]` Red: a test whose step declares `CONFIG__RETRY_SAME_MODEL_ON_TIMEOUT: "false"` uses one attempt, and the same numbers then pass.
- [x] `[AC2]` Red: a test whose step declares no `CONFIG__AI_TIMEOUT` reports a problem, because it runs an image default this repository never chose.
- [x] `[AC2]` Green: `chain` is the step's literal `CONFIG__FALLBACK_MODELS` list plus one, or the toml's `fallback_models` plus one when the step holds a GitHub expression.
- [x] `[AC3]` **The job ceiling is not checked in Go.** A first attempt (the two largest step ceilings plus setup) passed at 34 minutes on a job that needs 36 — a guard agreeing with the broken number, which is this spec's own defect class. `jobCeiling`, `prWorstPath`, `prSetupSeconds` and the test for them were deleted; `checkReviewBudget`'s comment records why, and a test pins the guard's silence so the weak duplicate cannot come back.

## Wire it in

- [x] `[AC1]` `checkReviewBudget` is called from `CheckDrift`, so `dotf harness check` runs it on every CI platform.
- [x] The guard is red on the pre-fix configuration: seven findings (four clips once a step is measured against its own window, two ceilings, one undeclared `ai_timeout`), and `harness check` exits non-zero.

## Fix the configuration

- [x] `[AC1]` `.pr_agent.toml`: `max_model_tokens` 200000 → 150000, with the arithmetic that justifies it in the comment.
- [x] `[AC1]` Both workflow `CONFIG__MAX_MODEL_TOKENS` overrides 200000 → 150000, so the env does not restore the inert clip the file just fixed.
- [x] `[AC2]` The two Anthropic review steps: `timeout-minutes` 8 → 13.
- [x] `[AC2]` The NaN review step: state `CONFIG__AI_TIMEOUT: "120"`, the value it already runs with.
- [x] `[AC3]` The job ceiling 34 → 38, with the sum `25 + 6 + 3 + 3 = 37` recorded in the comment; 38 is the smallest bound that exceeds it.

## Tests that pinned the old numbers

- [x] `tests/pr-agent-config.bats` routing test: the two Anthropic step ceilings 8 → 13.
- [x] `tests/pr-agent-config.bats` parity test: it asserted the prompt cap equalled the window, codifying the bug. It now asserts cap-for-cap across providers plus the room the answer needs, and points at `cli/internal/harness/prbudget.go` for the arithmetic.
- [x] `tests/pr-agent-config.bats` job test: red at `34 25 5 3` the moment the step ceiling grew, green at `37 25 5 3`. This test is what found the second defect.

## Documentation

- [x] `[AC5]` `docs/runbooks/guide-pr-agent-reviewer.md`: the Anthropic ceiling becomes 13 minutes, the NaN `ai_timeout` becomes stated rather than inherited, the clip and window are split into their own rows, and the job becomes 37 with its sum.
- [x] `docs/lessons/` gets the entry for the class: a setting compared against its own ceiling, a ceiling shorter than the budget it is supposed to cover, a value inherited from upstream, and a guard whose own bound was looser than the real one.

## Verification

- [x] `go test ./internal/harness/` green (full package, not only `-run Review`).
- [x] `harness check --repo-root ..` green on the fixed configuration and red on the pre-fix one (both captured in `verification.md`).
- [ ] `dotf spec review AI-054-review-budget-guard` passes before archive.
