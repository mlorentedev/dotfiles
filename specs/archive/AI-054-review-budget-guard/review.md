---
spec: "AI-054-review-budget-guard"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "8e1248802ce4481b510ac85c50e731a4afdc3e28"
reviewer: "nan/glm5.3-flash"
date: "2026-11-10"
---

## Adversarial review

**Scope**: AI-054-review-budget-guard — whole change, `git diff ef14013b05592e1a061c663851674ccbf6f1dfdb...HEAD` (8 first-parent commits, no merges in range; all 8 belong to this spec's work: 6449133, 9bf05c5, dc2d07b8, f4cc882, 2eea97b, 6b1ff96, 937f92a, 8e12488).
**Sources**: `specs/AI-054-review-budget-guard/{proposal,tasks,verification}.md`, `features.json`, `review-request.json`, `review.md` (round 1, `937f92aa`); `cli/internal/harness/prbudget.go` + `prbudget_test.go`, `check.go`, `.pr_agent.toml`, `.github/workflows/pr-agent.yml`, `tests/pr-agent-config.bats`, `docs/runbooks/guide-pr-agent-reviewer.md`, `docs/lessons/lesson-573-*.md`.

### Gate pre-flight

- Contract digests match `review-request.json` exactly via `dotf spec digest AI-054-review-budget-guard` (features `584f13f0…`, proposal `edb297ea…`, tasks `8c422b4d…`). Not stale at review time.
- `reviewer: nan/glm5.3-flash` recorded as drawn. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec folder. `reviewed_sha` = `git rev-parse HEAD` = 8e12488.
- `proposal.md` declares no `risk:` → single signature suffices.

### Spec and task alignment

All six ACs map to named tests, every one re-run in this session:

| Claim | Command | Result |
|---|---|---|
| Package green | `go test ./internal/harness/ -timeout 5m` | ok, exit 0 |
| Review tests | `go test ./internal/harness/ -run Review -v` | **16/16** pass, matching verification.md |
| AC1/AC2/AC4 feature set | `test -run '…TheCurrentConfigurationFails\|TheFixedConfigurationPasses\|TheRepositoryPassesItsOwnGuard$'`, `test -run '…AWorkflow…\|HalfAReviewer\|AFileWithoutAClip\|AnUnparseable\|ACheckout…$'` | pass (4 + 1 ok) |
| Guard green on this repo | `go run ./cmd/dotf harness check --repo-root ..` | exit 0, 4 clip-OK + 3 ceiling-OK lines, `no harness drift` |
| AC3 job ceiling | `bats tests/pr-agent-config.bats --filter 'the job outlives both bounded Actions and the probes'` | ok (sum `38 > 25+6+3+3`) |
| AC6 routing + parity | bats filters `at most two bounded attempts…`, `the Anthropic attempts run the routed model…` | both ok (`13 13 True…`, `True True True`) |
| AC5 runbook | grep for `31-minute` / 8-minute / 200000-as-clip | only window-side 200000 remains (intended) |
| Wire-in | `grep checkReviewBudget cli/internal/harness/check.go` | called from `CheckDrift` |
| Red guard exits non-zero | mutation: clip 200000 beside workflow `mktemp` copy, `harness check --repo-root $TMP` | `[GAP]` printed, **exit 1** |
| Lint/build | `go build ./...`, `go vet ./internal/harness/`, `gofmt -l`, `golangci-lint run ./internal/harness/...` | clean, 0 issues |

Mutation battery on copies of the tree in a `mktemp -d` checkout of the two files + `harness/` (all reverted, repo tree clean):

| Mutation | Observed |
|---|---|
| toml `max_model_tokens` 150000 → 200000 | `[GAP] the prompt clip in .pr_agent.toml never fires: 200000 + 40000 = 240000, not under 200000` |
| drawn-first step's `CONFIG__CUSTOM_MODEL_MAX_TOKENS` 200000 → 160000 | `[GAP] … drawn first … 150000 + 40000 = 190000, not under 160000` (the audited fix `f4cc882` bites) |
| NaN step's `CONFIG__AI_TIMEOUT: "120"` deleted | `[GAP] PR-Agent does not declare CONFIG__AI_TIMEOUT` |
| first Anthropic `timeout-minutes: 13 → 8` | `[GAP] … 1 models x 2 attempts x 360 s = 720 s in a 480 s ceiling` |

Round-1 finding dispositions verified against the tree: `TestCheckReviewBudget_AnUnparseableFileFailsClosed` exists and pins the garbage-in-either-file case (round-1 Major, applied); proposal says "AI-054's AC7" nowhere and cites "AI-045's AC7"; the fix cited on the branch is `f4cc882` (no dangling `da794f6`); `features.json` carries three features whose verification commands I ran, all green.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | spec artifacts (contract) | `proposal.md` Risks still carries stale arithmetic: "*12 + 13 = 25* of the 34 minutes available; the probes, setup and publication calls take *the other 11*" — 34−25 is 9, the post-audit other-part is 12, and the job is now 38; it contradicts the same proposal's own `25 + 6 + 3 + 3 = 37` two paragraphs earlier | read of `proposal.md` line 60; no repro needed, it is a written contradiction | UNTESTED (prose) | spec (proposal.md is contract set — disposition in `verification.md`, fix next contract revision; editing it now would invalidate this review) |
| Minor | THEORETICAL | guard coverage | A token override at the job or workflow-level `env:` is invisible to the clip check (fail-open class) | ticket **#3299**, OPEN | UNTESTED at job level; step-level override covered by `TestCheckReviewBudget_AStepThatDeclaresItsOwnClipIsCheckedToo` | already ticketed (#3299) — tracked |
| Minor | THEORETICAL | guard reserve | Reserve is 0 when no review step declares an output cap, and a step declaring one as an expression reads as 0; a step at exactly its ceiling passes (`need <= have`) | ticket **#3301**, OPEN | boundary `< vs <=` covered by `TestCheckReviewBudget_TheClipMustLeaveSlackNotJustFit`; the missing-cap case is UNTESTED | already ticketed (#3301) — tracked |
| Minor | THEORETICAL | guard marker | A review step that loses the `pragent/pr-agent` marker (`uses:` rename, `run: docker run`, reusable workflow) drops out silently only when *no* step has it; a partial marker loss inside a job that still has one step is not detected | ticket **#3300**, OPEN | `TestCheckReviewBudget_AWorkflowWithoutAReviewStepFailsClosed` covers the all-lost case; partial loss UNTESTED | already ticketed (#3300) — tracked |
| Minor | THEORETICAL | timeout semantics | The streamed NaN step's `ai_timeout` bounds the gap between chunks, not the whole answer, and upstream `num_retries` is unset | ticket **#3302**, OPEN; also `runbook` names the streaming difference | UNTESTED (needs measurement) | already ticketed (#3302) — tracked |
| Minor | THEORETICAL | guard chain bound | `literalList` accepts only JSON; a YAML flow-list literal (`[a, b]`, unquoted) falls back to the toml pool count, which can over-estimate chain — a false positive, the safe direction | code read of `prbudget.go` `literalList` | UNTESTED | code (follow-up ticket candidate; no action needed this round) |

No Blockers. No REAL Majors. The four tickets above (#3299–#3302) exist, are OPEN, and are dispositioned in `verification.md` with reproduce-first evidence — they are tracked, not open claims.

### Test deletions — retention field check

The change deletes the first-attempt Go job-ceiling check (`jobCeiling`, `prWorstPath`, `prSetupSeconds`) together with its test. Every retention field is recorded: (1) the deleted test guarded a bound that passed at 34 on a job needing 37; (2) the stronger proof that remains is shown — `tests/pr-agent-config.bats:1189` computes the sum from the workflow and the pool and was demonstrated red at 37 (`False 37 25 6 3`) before the job moved to 38; (3) no production caller remains; (4) history is this change itself; (5) the focused command is the bats filter, run green here. The silence against a weaker duplicate is pinned by `TestCheckReviewBudget_TheGuardLeavesTheJobCeilingToTheBatsTest`. Compliant.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All six ACs verified by named tests re-run here plus an independent mutation battery; residual gaps are narrow, ticketed blind spots (job-level env, marker loss) |
| Verification       | A | Every claim in verification.md reproduced in this session from the stated command; digests, exit codes and lint all checked |
| Scope              | A | Diff is exactly the guard + the two config files it fixes + runbook/lesson/spec; no creep |
| Reliability        | A | Fail-closed at every parse/half-file/no-step path, each with a named test; red exit verified |
| Maintainability    | A | Small functions, low complexity, table-driven fixtures, comments state why (34→37 incident recorded in both code and spec) |
| Handoff-readiness  | A | Lesson-573 promoted, runbook refreshed, round-1 dispositions and four open tickets recorded with reproduce-first evidence |

### Verdict

**PASS WITH GAPS** — no Blocker, no REAL Major; five Minors, three REAL/THEORETICAL already ticketed (#3299–#3302) and one spec-text inconsistency (proposal Risks arithmetic) to disposition. Rubric all B or above.

`dotf spec archive` is **advisable** in the current state: the contract digests match, the review is fresh at the reviewed sha, and every round-1 Major was applied with a named test. The tracked minors ride in their own tickets and do not gate the archive.

### Recommended next steps

Route by set: the contract set is closed under this verdict — these are for the implementer to disposition in `verification.md` (applied / ticketed / declined with reason) or to carry into the existing tickets.

- Confirm the dispositions on #3299, #3300, #3301 and #3302 stay recorded here and land with their own tests when the tickets are picked up (no contract edit).
- Disposition the proposal Risks stale arithmetic ("34 minutes available", "the other 11") in `verification.md`; correct it whenever the contract is next legitimately revised.
- Optional follow-up ticket: `literalList` rejecting unquoted YAML flow lists over-estimates chain via the toml pool — safe direction, worth one sentence in #3299's class rather than code now.
- When #3301/#3299 are fixed, extend `dotf pr reproduce` coverage to the job-level env override so the blind spot cannot re-open silently.
