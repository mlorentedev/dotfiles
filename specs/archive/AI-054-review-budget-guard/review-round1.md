---
spec: "AI-054-review-budget-guard"
verdict: "FAIL"
reviewed_sha: "2eea97b3a77cc2a9992a904381257bc7eec5827f"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: AI-054-review-budget-guard (whole change, `git diff ef14013b05592e1a061c663851674ccbf6f1dfdb...HEAD`; 5 first-parent commits, no merges in range: 6449133, 9bf05c5, dc2d07b8, f4cc882, 2eea97b)

**Sources**: `specs/AI-054-review-budget-guard/{proposal,tasks,verification,review}.md`, `features.json`, `review-request.json`; `cli/internal/harness/prbudget.go` + `prbudget_test.go`, `cli/internal/harness/check.go`, `.pr_agent.toml`, `.github/workflows/pr-agent.yml`, `tests/pr-agent-config.bats`, `docs/runbooks/guide-pr-agent-reviewer.md`, `docs/lessons/lesson-573-*.md`.

### Gate pre-flight

- Contract digests match `review-request.json` exactly via `dotf spec digest AI-054-review-budget-guard` (features 2d9da26b…, proposal 05c8987b…, tasks 9b987d0e…). Not stale at review time.
- `reviewer: nan/mimo-v2.6-flash` is a pool member (`harness/reviewer-pool.json:87`). No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the folder. `reviewed_sha` = `git rev-parse HEAD` = 2eea97b3.
- `proposal.md` declares no `risk:` → single signature required.

### Spec and task alignment

Reproduced in this session (not taken on trust):

| Claim | Command | Result |
|---|---|---|
| Package green | `go test ./internal/harness/ -timeout 5m` | ok, exit 0 |
| Review tests | `go test ./internal/harness/ -run Review -v` | **15/15** pass (verification.md says 12/12 — stale) |
| Guard green on this repo | `go run ./cmd/dotf harness check --repo-root ..` | exit 0; 4 clip OK lines + 3 ceiling OK lines |
| Guard red on pre-fix config | base `.pr_agent.toml` + `pr-agent.yml` (ef14013b) extracted to `mktemp -d`, same command | exit 1; **7** budget findings (4 clips + 2 ceilings + 1 undeclared `ai_timeout`) — tasks.md says six |
| AC4 behaviour (unparseable) | garbage toml in `mktemp -d` root | `[ERROR] .pr_agent.toml: not valid TOML…` — behaviour holds at runtime, but **no committed test** (finding 1) |
| AC3 SSOT | `bats tests/pr-agent-config.bats` | 72/72 pass |
| Build/vet/lint | `go build ./...`, `go vet` (+`GOOS=windows`), `golangci-lint run ./internal/harness/...`, `gofmt -l` | all clean (lint: 0 issues) |

Mutation battery (all in `mktemp -d` git worktrees, all reverted, worktrees removed; working tree clean apart from this review):

1. `clipLine`: `<` → `<=` → `TestCheckReviewBudget_TheClipMustLeaveSlackNotJustFit` **red** (AC1 boundary bites).
2. workflow job `timeout-minutes: 38` → `37` → `@test "pr-agent: the job outlives both bounded Actions and the probes before them"` **red** at `False 37 25 6 3` (AC3 SSOT bites, matching verification.md's claim).
3. `.pr_agent.toml` clip `150000` → `200000` → `dotf harness check` **red** with the clip GAP on the file and on the NaN step, Anthropic steps still OK (AC1 end-to-end bites).
4. Same toml mutation → `@test "pr-agent: the Anthropic attempts run the routed model, allowlisted, with bounded input and output"` (the parity test) **red** at line 1306 (AC6 bites).

Tasks: all boxes ticked except the final review task. One ticked task has no corresponding artifact (finding 1). The late audit fix (f4cc882) updated code, workflow, runbook and the proposal's *What*, but left several recorded counts and the proposal's AC5/Risks stale (findings 2–4). Dispositions for the PR-agent review and the independent audit are recorded in `verification.md` with tickets #3299–#3302 — good practice, two citation defects in them (findings 3–4).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | spec/tests (AC4) | AC4's "either file is present but unparseable" clause has no test, yet tasks.md ticks `Red: a test builds a repo with a reviewer but an unparseable .pr_agent.toml / workflow` — no such test exists anywhere in the package, and verification.md's AC4 evidence cites only the no-step, missing-file and neither-file tests | `prbudget_test.go` has 15 test funcs; grep for `unparseable\|not valid\|invalid` matches nothing; only `prbudget_test.go` references the guard's functions | **UNTESTED** — the finding *is* the absent regression test (behaviour itself observed working: `[ERROR] .pr_agent.toml: not valid TOML…` on a garbage-toml root) | tests (+ verification.md to name it once added) |
| Major | REAL | spec (contract, AC5) | AC5 requires the runbook to state "the 37-minute job"; the runbook states **38** (correctly — the audit raised it), the proposal's *What* says 38, but proposal Risks still says "the job ceiling (37)" and AC5 says 37: AC5 cannot be honestly ticked as written. verification.md ticks AC5 while misquoting the file as "the step/job rows are `13` and `37`" | proposal.md:43 (38) vs :55 ("(37)") vs :69 (AC5 "37-minute job"); runbook job row `38`; f4cc882 amended only the What (2-line proposal change) | The number is pinned by `@test "pr-agent: the job outlives both bounded Actions and the probes before them"` — green at 38, red at 37 (reproduced); the defect is spec text | spec (proposal.md = contract → digest change → re-review; verification.md line free) |
| Minor | REAL | spec + verification (counts) | The late fix's sweep missed recorded counts: tasks.md "six findings" (pre-fix run now reports **7** — f4cc882's step-window measurement added a fourth clip line); verification.md "asserts **four** findings" (the test asserts **5**), "**12/12**" (now 15), "**six** findings" (7) | my pre-fix run (7 findings); `TestCheckReviewBudget_TheCurrentConfigurationFails` asserts `n != 5`; `-run Review -v` = 15 | `TestCheckReviewBudget_TheCurrentConfigurationFails` (named) | tasks.md (contract) + verification.md |
| Minor | REAL | verification (citations) | The disposition table credits `da794f16` for both applied fixes; that sha is an ancestor of no branch (dangling, content differs: job 37, sum 36) and cannot be resolved on a fresh clone. The in-range commit is `f4cc882` | `git merge-base --is-ancestor da794f16 HEAD` → not ancestor; `git branch -a --contains da794f16` → empty; `git range-diff` da794f16 vs f4cc882 shows the 37-vs-38 content difference | n/a | verification.md (outside contract) |
| Minor | REAL | spec (contract, features.json) | `features.json` is untouched template scaffolding: `behavior: "Outcome 1"`, `verification: "echo 'not implemented' && exit 1"`, `state: "pending"` — contradicts the ticked implementation. Its digest is in `review-request.json`, so it must be decided while the contract is open | file contents; `dotf spec digest` | n/a — whether the archive gate consumes `features.json` state is **UNVERIFIED** (gate lives outside this repo; not run, it would mutate) | spec (contract) — owner decides: fill or drop |
| Minor | THEORETICAL | code (fail-open edge) | A step-level *expression*-valued `CONFIG__MAX_MODEL_TOKENS` / `CONFIG__CUSTOM_MODEL_MAX_TOKENS` / `CONFIG__MAX_OUTPUT_TOKENS` reads as 0 and is silently skipped in `checkTokenClip` (fail-open), while an expression `ai_timeout` fails closed — inconsistent direction. Tickets #3299 (job/workflow-level env) and #3301 (reserve) do not cover a step's clip/window expression | prbudget.go: `if s.maxTokens <= 0 && s.window <= 0 { continue }`; `envInt` returns 0 on `Atoi` failure; both committed clip overrides are literals (workflow:459, :690) | UNTESTED; closest is `TestCheckReviewBudget_AnExpressionChainUsesTheTomlPoolAsItsBound` (fallbacks only) | code or ticket (#3299-adjacent) |
| Minor | SPECULATIVE | spec (citation) | proposal.md cites a bare "AC7" — this spec has only AC1–AC6; it is AI-045's AC7 (`specs/archive/AI-045-nan-catalog-alignment/proposal.md:98`, which indeed requires the job bound to exceed the sum). The workflow/bats bare "(AC7)" predate this change | proposal.md:43 vs :65–70; AI-045 archive | n/a | spec (optional; bundle with the FAIL edits) |

Strengths, where they mitigate a documented risk: the deleted weak Go job-ceiling bound is pinned against return by `TestCheckReviewBudget_TheGuardLeavesTheJobCeilingToTheBatsTest` (reproduced green on a 1-minute job), which addresses exactly the "guard agrees with a wrong ceiling" risk this spec exists for.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | All runtime behaviour I could exercise is correct, but AC5 is not met as written and AC4's unparseable clause has no committed test — criteria partially met |
| Verification       | C | Evidence is extensive and I reproduced it, but a cluster of recorded claims do not hold as written (four/five, 12/15, six/seven, 37/38, unreachable da794f16) |
| Scope              | A | Exactly the spec's work: 5 in-range commits, no merges, no creep; the audit amendments are documented in-commit |
| Reliability        | B | Fail-closed paths observed end-to-end (half reviewer, no step, garbage toml, missing clip); residual expression-level fail-open edges are theoretical and ticketed |
| Maintainability    | A | golangci-lint 0 issues, gofmt/vet clean (incl. GOOS=windows), functions <40 lines, WHY-comments throughout, 15 named fixture tests |
| Handoff-readiness  | C | Lesson-573, runbook and dispositions exist, but the contract text contradicts delivered reality in three places (AC5, Risks "(37)", task tick) |

### Verdict

**FAIL.** Two **REAL Majors** force it regardless of rubric: (1) an acceptance-criterion clause (AC4 unparseable) with **UNTESTED** coverage whose implementing task is ticked against a test that does not exist — by the gate's own rule an UNTESTED Major needs a named regression test before PASS; (2) AC5 as written cannot be honestly ticked against the delivered runbook, and its verification tick misquotes the artifact — a contract-set defect, and the contract set must be edited before this can archive. The rubric independently floors at PASS-WITH-GAPS (three C's); severity × reality governs and is the more severe path.

### Recommended next steps

Contract set (`proposal.md`, `tasks.md`, `features.json`) — these are the point of a FAIL; a re-review follows them:

1. `proposal.md` AC5: "the 37-minute job" → "the 38-minute job"; Risks: "the job ceiling (37)" → "(38)". (Optional: qualify "AC7" as "AI-045's AC7".)
2. `tasks.md`: "six findings" → "seven findings" (the pre-fix run's actual count under the current guard).
3. `features.json`: fill the placeholder Outcome/verification or state in it why the placeholder stands for this spec.

Tests (outside the contract set):

4. Add the AC4 unparseable regression test (invalid toml and invalid workflow fixtures, each asserting a problem) — this is the Major's fix and makes the existing tick true.

`verification.md` (outside the contract set, editable any time):

5. Correct the counts ("four"→"five", "12/12"→"15/15", "six"→"seven"), the AC5 evidence line ("13 and 37"→"13 and 38"), replace the unreachable `da794f16` citation with `f4cc882`, and name the new unparseable test under AC4.

Then re-run `dotf spec review AI-054-review-budget-guard` — the contract edits change the digests, so this verdict is superseded by the next round by design.

**UNVERIFIED**: archive-gate internals and its handling of `features.json` (implementation is outside this repository and was not run, since archiving would mutate); verification.md's Windows-side bats run (I ran the full file on macOS: 72/72); the origin of dangling `da794f16` beyond "no branch contains it".
