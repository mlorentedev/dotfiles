---
spec: "AI-055-budget-guard-readable-reserve"
verdict: "FAIL"
reviewed_sha: "06f3100b7efafdfdf77d250553b23994eefc7bd2"
reviewer: "nan/glm5.3-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: AI-055-budget-guard-readable-reserve (round 2; round 1, `agy/gemini-3.8-flash-high`, left a PENDING review that signed nothing)
**Sources**: `specs/AI-055-budget-guard-readable-reserve/{proposal,tasks,verification}.md` + `git diff 2aa3ad1693bd95b19de67fd37aea1df8e798b6f1...HEAD` (`7d0c2b33` fix, `06f3100b` spec docs)

### Spec and task alignment

- All six acceptance criteria map to named tests and `features.json` entries; the diff scope matches the proposal (only `prbudget.go`, `prbudget_test.go`, one lesson, one runbook, spec artifacts).
- AC1–AC5 each have a failing-first test (verification records a mutation check that fails exactly the four new-behaviour tests against `origin/main`); AC6 is a regression head. `dotf spec review` did the diff-vs-`main` correctly (base `2aa3ad1693bd95b19de67fd37aea1df8e798b6f1` was stated in the launch request; not substituted).
- The TDD artifacts are honest about what the change does not prove (streamed wall clock is #3302; workflow-level token override skip still open).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | tests (fail-closed coverage) | This branch's own commit deleted the only tests covering two fail-closed branches of the guard. `7d0c2b33` removed `TestCheckReviewBudget_AFileWithoutAClipFailsClosed` and the second half of `TestCheckReviewBudget_HalfAReviewerFailsClosed` (workflow present, toml missing). Fresh whole-package coverage run in this session shows both branches now execute 0 times anywhere in the suite: `prbudget.go:259.29,262.3` (`max_model_tokens is absent or not positive`) and `prbudget.go:130.20,132.3` (error naming the toml on a half reviewer). The code still contains both branches — a coverage regression, not a behaviour change — but a regression that deletes either branch would now go red nowhere | `go test ./internal/harness -count=1 -coverprofile=…`: both blocks count 0; `git log -S "AFileWithoutAClipFailsClosed"` shows the test born in `3ef5dfa6` (#3289) and deleted in `7d0c2b33` of this PR | UNTESTED (that is the finding) | tests (restore or equivalent negative-path tests), plus tasks.md recording the deletion per the retention bar |
| Major | REAL | verification artifact | `verification.md` AC6 asserts that `AFileWithoutAClipFailsClosed` "all still pass". That test does not exist in the reviewed tree — it was deleted by this same change. A false named-artifact claim in the verification triad | file read of `verification.md` AC6 vs `grep -n AFileWithoutAClip internal/harness/prbudget_test.go` (no match, this session) | n/a — the claim itself is the defect | spec artifacts (`verification.md`; outside the contract set, editable without invalidating a review) |
| Minor | THEORETICAL | code | `declaresStreaming` matches key *presence* only, never the value: `LITELLM__FORCE_STREAMING: "false"` or `'[]'` (empty substring list, i.e. nothing forced) would still earn the `(streamed: …)` caveat. Advisory annotation only — no invariant depends on `streams` — so it misclassifies words, not arithmetic | code read of `declaresStreaming`; `envForceStreaming` prefix match over keys | UNTESTED (no test for a falsy/streaming-off value) | code (parse the value), or accept and document the presence semantics |
| Minor | THEORETICAL | code (message truth) | An unreadable-ness message that lies about unreadability: a declared cap of `"0"` or `"-5"` *is* a number, but `answerReserve` reports `… is %q, which this check cannot read as a number` for it. Fail-closed direction is correct in both cases; the wording is what misleads | code read of `answerReserve` (gap branch keyed on `s.outputCap <= 0`, not on parse failure); `"0"`/`"-5"` both parse via `envInt`/`Atoi` | UNTESTED for the `"0"`/`"-5"` variants (the named test covers `${{ vars… }}` and `"40.000"` only) | code (message wording) or tests (add the numeric-invalid case) |
| Minor | THEORETICAL | code (diagnostics) | A step missing *both* `CONFIG__AI_TIMEOUT` and `timeout-minutes` reports only the ai_timeout gap; the new minutes gap (AC5) is masked in that combination by the ordering in `stepCeiling`. Fail-closed either way | code read of `stepCeiling` branch order | UNTESTED for the both-missing combination | code (report both) or accept as deliberate ordering |
| Minor | SPECULATIVE | tests | Silent weakening of `AStepThatLowersItsOwnWindowIsMeasuredAgainstIt`: the rewrite dropped the negative assertion "a step window equal to the file's changes nothing" that the pre-change version carried. The remaining positive case still exercises the per-step window path, so the loss is small, but the deletion is not recorded | diff of the test between base and HEAD | UNTESTED (the dropped assertion) | tests (re-add or record the deletion) |
| Question | — | spec (scope) | The guard now hard-requires `CONFIG__MAX_OUTPUT_TOKENS` in the workflow step env. A cap declared in `.pr_agent.toml` (which PR-Agent also reads) is invisible and trips AC2's "no review step declares" message. Fail-closed and consistent with the proposal's stated risk; candidate for a follow-up ticket rather than this change | code read of `answerReserve` (steps only, never the toml) | n/a | vault (follow-up ticket) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All six criteria hold with named tests incl. a mutation check; the negative-path regression sits under Verification, not correctness |
| Verification | C | Most evidence is fresh and reproducible, but AC6 names a test that no longer exists — a claim the artifact cannot back |
| Scope | B | Diff matches the proposal's contract exactly; the blemish is deletions inside it that tasks.md does not record |
| Reliability | B | Every fail-closed path I probed still closes; the repository's own configuration passes the stricter guard (re-run in this session: exit 0, 19/19 guard tests, whole package ok 24.3s) |
| Maintainability | B | Functions stay small, CC low, messages name their files and numbers; `answerReserve`/`largestOutputCap` consolidation is cleaner than its parts |
| Handoff-readiness | C | Lesson 584 and the runbook addition are good handoff material, but verification.md's false test claim would send the next session chasing a ghost |

### Verdict
FAIL

One **REAL Major** (the deleted tests' fail-closed branches now uncovered) forces FAIL by `severity × reality`; two C grades independently force PASS-WITH-GAPS minimum rubric-wise, and a REAL major outranks that.

### Recommended next steps

Ordered fixes (this is a FAIL, so the contract set may change and a re-review follows):

- tests: re-add the two deleted negative-path cases — a toml with `custom_model_max_tokens` but no `max_model_tokens` must report `max_model_tokens is absent or not positive`, and a checkout with only the workflow must report an error naming the toml — then run `cd cli && go test ./internal/harness -count=1` fresh and paste the output.
- spec artifacts: correct `verification.md` AC6 to name the tests that actually exist after the restore, and record both deletions (plus the dropped equal-window assertion) in `tasks.md` with the retention-bar fields: the failure each deleted test could detect, the proof that remains, history (`git log -S`, born `3ef5dfa6`), and the focused command.
- No contract rewording is required beyond that; the acceptance criteria themselves are sound.
- Then run round 3 (`dotf spec review AI-055-budget-guard-readable-reserve`) — a REAL Major requires a fix and a re-review, not just a patch note.
- Track, do not block: the `declaresStreaming` value-blindness, the numeric-invalid-cap wording, the both-missing diagnostics ordering — disposition them in `verification.md` or a follow-up ticket; none gates this spec.
- After merge and before archive, the `Question` row (cap declared in the toml is invisible to AC2) is fold-ready into either #3301's follow-ups or a new ticket.

Session evidence produced in this review: suite 20/20 PASS + whole package `ok 24.328s`; `go vet ./internal/harness` exit 0; `go run ./cmd/dotf harness check --repo-root ..` exit 0 with the `(streamed: …)` caveat on the NaN step only; gofmt clean on LF-normalised copies (the working tree is CRLF from checkout, `i/lf w/crlf`, so the earlier `gofmt -d` full-file diff is a line-ending artifact, not a lint failure); coverage profile with the two zero-count blocks named above.
