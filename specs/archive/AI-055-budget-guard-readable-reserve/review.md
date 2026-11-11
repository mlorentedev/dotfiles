---
spec: "AI-055-budget-guard-readable-reserve"
verdict: "PASS"
reviewed_sha: "5247b12ece5f4605d9fb388330b25102589d3a7a"
reviewer: "nan/glm5.3-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: AI-055-budget-guard-readable-reserve (round 4; round 2 FAIL by nan/glm5.3-flash, round 3 left a PENDING review by agy/gemini-3.8-flash-high that signed nothing)
**Sources**: `specs/AI-055-budget-guard-readable-reserve/{proposal,tasks,verification}.md`, `features.json` + `git diff 2aa3ad1693bd95b19de67fd37aea1df8e798b6f1...5247b12ece5f4605d9fb388330b25102589d3a7a` (`7d0c2b33` fix, `06f3100b` spec docs, `5247b12e` coverage restore)

### Spec and task alignment

- All six acceptance criteria map to named tests and non-vacuous `features.json` verification commands; the diff scope matches the proposal (prbudget.go, prbudget_test.go, one lesson, one runbook section, spec artifacts — no `.pr_agent.toml`, pool, workflow or bats change).
- Both round-2 Majors verified fixed in this session, not trusted from the artifact:
  - The deleted fail-closed tests are back in the tree (`AFileWithoutAClipFailsClosed`, the toml-missing second half of `HalfAReviewerFailsClosed`, `TheReserveIsTheLargestCapNotTheFirst`), and block-level coverage shows the two previously-zero blocks now executing: `prbudget.go:259.29,262.3` (max_model_tokens absent) count 1, `prbudget.go:130.20,132.3` (error naming the toml) count 1. `checkTokenClip`, `answerReserve`, `stepCeiling` and `declaresStreaming` are all at 100% function coverage.
  - `verification.md` AC6 now names only tests that exist in the tree; my fresh run reproduces its counts exactly (20/20 guard tests, whole package `ok 24.4s`).
- The deletion audit in `tasks.md` covers every removal against the retention bar with the failure each deleted test detected and why it stayed; the dropped equal-window negative assertion, the `// Five:` rationale and the `AI-054` fixture comment are all restored verbatim from `origin/main`.
- The mutation check was re-executed by this reviewer, not trusted: with `prbudget.go` reverted to the pre-fix base `2aa3ad16` (`answerReserve` absent), exactly the four new-behaviour tests fail and the other sixteen stay green; the file was restored byte-identical afterwards.
- The contract digests in `review-request.json` match the tree; no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | code | `declaresStreaming` matches key presence only, so `LITELLM__FORCE_STREAMING: "false"` also earns the caveat. Advisory annotation only; under-reporting a stream is the safe direction | Round-2 finding, still present in `declaresStreaming`; dispositioned in `verification.md`, tracked #3302 | UNTESTED (no falsy-value case) | none in this change — tracked on #3302 |
| Minor | THEORETICAL | code (message truth) | A cap of `"0"` or `"-5"` is a number but is reported as "cannot read as a number". Fail-closed either way; `checkTokenClip` rejects the toml one line earlier | Round-2 finding, still present in `answerReserve`; dispositioned, tracked #3302 | UNTESTED for the non-positive literal | none in this change — tracked on #3302 |
| Minor | THEORETICAL | code (diagnostics) | A step missing both `CONFIG__AI_TIMEOUT` and `timeout-minutes` reports only the ai_timeout gap (branch order in `stepCeiling`). Fail-closed; the second gap surfaces once the first is fixed | Round-2 finding, dispositioned | UNTESTED for the both-missing combination | none in this change — tracked on #3302 |
| Minor | THEORETICAL | spec artifacts (stale evidence) | `features.json` f6 evidence says "19/19 guard tests PASS"; the restored coverage makes it 20/20 (fresh count in this session). Accurate when written at `06f3100b`, stale after `5247b12e` | This session: 20 `--- PASS` lines under `-run TestCheckReviewBudget`; `verification.md` already records 20 | n/a — prose staleness, no behaviour | spec artifacts — but `features.json` is in the contract set, so do NOT edit it now; record the disposition in `verification.md` and correct the count in the archive's next touch of that file (if any), or leave as-is with the disposition |
| Minor | SPECULATIVE | housekeeping | `review-round3.md` in the spec folder is a PENDING review by `agy/gemini-3.8-flash-high` that signed nothing (verdict PENDING, "Initial evaluation in progress"), untracked in git. It is gate-inert (the archive reads `review.md`) but a future session could mistake it for a live review | File read: verdict PENDING, single placeholder finding row | n/a | spec artifacts — human decision: delete the stale round-3 file or fold its existence into the provenance note; do not commit it as-is |
| Question | — | spec (scope, resolved) | Round 2 asked whether a cap declared in `.pr_agent.toml` is invisible to AC2. Answered in `verification.md`: AC2 covers the workflow's own declaration; the toml cap is `checkTokenClip`'s bound and is required (`AFileWithoutAClipFailsClosed`). Different ceilings, deliberately | `verification.md` round-2 disposition | `TestCheckReviewBudget_AFileWithoutAClipFailsClosed` | none — answered |

One additional probe came back clean: a cap declared with surrounding whitespace (`" 40000 "`) parses identically in `envInt` (which trims) and `outputCapRaw` (trimmed in `newReviewStep`), so the GAP message can never display a trimmed number the check in fact read.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All six criteria verified with named tests; negative paths covered and the mutation battery re-run by the reviewer fails exactly the four new-behaviour tests |
| Verification       | A | Every evidence command re-run fresh in this session with matching output, including the mutation check and block-level coverage of the restored branches |
| Scope              | A | Diff matches the proposal exactly; every test deletion audited in tasks.md under the retention bar |
| Reliability        | A | Every fail-closed path probed still closes; both previously-uncovered blocks now execute; the repository passes its own stricter guard (exit 0, caveat on the NaN step only) |
| Maintainability    | B | Small functions, low CC, messages name their files and numbers; `largestOutputCap`→`answerReserve` consolidation is cleaner than its parts; CC not formally measured this round |
| Handoff-readiness  | B | Lesson + runbook added and "What this does not prove" is honest; blemishes are the stale f6 count and the stray PENDING round-3 file |

### Verdict
PASS

No Blockers; both round-2 REAL Majors verified fixed; the open Minors are all THEORETICAL/SPECULATIVE and already dispositioned in `verification.md` (tracked on #3302) or housekeeping. Rubric all A/B.

### Recommended next steps

On a PASS the contract set is closed — none of these block the archive; disposition each in `verification.md` or carry into a follow-up ticket:

- Resolve the stray `review-round3.md` (PENDING, signed nothing) before archiving: delete it or record its provenance, so a later session does not read it as a live review. Human call — the file is untracked.
- Record in `verification.md` that the `features.json` f6 evidence count ("19/19") predates the coverage restore and the current count is 20/20; correct the number only if `features.json` is touched for another reason.
- The three code Minors (streaming value-blindness, non-positive-literal wording, both-missing diagnostics) stay tracked on #3302 as dispositioned.
- After merge, fold the resolved Question (toml-declared cap invisible to AC2) into #3301's follow-ups if it ever earns its own ticket.

Session evidence produced in this review: suite 20/20 guard tests PASS, whole package `ok 24.4s`; `go vet ./internal/harness` exit 0; `go run ./cmd/dotf harness check --repo-root ..` exit 0 with the `(streamed: …)` caveat on the NaN step only; mutation check re-run against base `2aa3ad16` — exactly the four new-behaviour tests red, sixteen green, working tree restored byte-identical (`cmp` clean); gofmt clean on LF-normalised copies; block coverage count 1 on both previously-zero fail-closed blocks.
