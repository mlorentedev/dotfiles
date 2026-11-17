---
spec: "BUG-170-review-not-on-head"
verdict: "PASS"
reviewed_sha: "5d6a957b9192088492ccfbc7957992553f9a78d0"
reviewer: "nan/glm5.3-flash"
date: "2026-11-12"
---

## Adversarial review

**Scope**: BUG-170-review-not-on-head (`dotf pr triage-queue` / `pr land` vs #3243, ts-bridge#687)
**Sources**: `specs/BUG-170-review-not-on-head/{proposal,tasks,verification,features.json}` + `git diff 027a882bc695dbc1c71f49d63b7339e7da31e44b...HEAD`. The range holds exactly one branch commit (`5d6a957b`, `git log --first-parent --no-merges` confirms; `027a882b` is its direct parent), so the whole change reviewed is the spec's work.

### Spec and task alignment
- AC1 (hold a non-draft PR whose newest head `review` check is not `success`, repo-wide and `--pr`, reason naming head + reading): implemented in `withHeadReview` (`cli/internal/prtriage/headreview.go`), wired after `withReproductions` in `pendingOf`, failing reads error the whole queue (fail-closed). Covered repo-wide and `--pr` per subtest by `TestReviewOnHeadHoldsAQueueTheCommentsLeaveClear` (8 readings × 2 scopes), including cancelled-then-success on one head not listing.
- AC2 (draft and exempt-diff PRs not listed for their head review): drafts' check runs are never read (`withComments` gates the fetch); exempt diffs are dropped by the rewritten `withoutExempt`. Covered by `TestReviewOnHeadIgnoresADraft` (asserts no `/check-runs` call was made) and `TestReviewOnHeadLetsAnExemptDiffThrough` (exempt stays quiet, non-exempt held).
- AC3 (`pr land` refuses skipped/absent review under both names via one shared matcher): `prtriage.IsReviewCheck` (`review` or `<caller> / review`) used by `reviewReasons` in `Decide` and by `manualMerge`'s red-check predicate. Covered by `TestDecide_ReviewDidNotRunOnTheHead` (skipped under both names, absent, look-alike-only, `review / review` passing lands) and `TestIsReviewCheck`.
- tasks.md: all implementation boxes ticked and each maps to named tests in the diff; the one unticked box ("verification.md filled in; independent review; archive") correctly awaits this review and the archive. No `[x]` without diff evidence found.
- The contract digests in `review-request.json` match `dotf spec digest BUG-170-review-not-on-head` exactly (proposal, tasks, features.json), so this review reads the recorded contract.
- The `withoutExempt` rewrite is a necessary companion, not scope creep: with the old re-evaluation restored, a PR pending only for its head review is re-evaluated by `Evaluate` as "no reviewer output" and released — `TestReviewOnHeadLetsAnExemptDiffThrough` turns red under that mutation (reproduced in this session). The new rule (drop only when the diff is exempt) is a superset of the old kept-set and strictly fail-closed.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | SPECULATIVE | headreview.go | Newest-wins ordering degrades when two review runs share an identical `started_at` (first in the API answer wins) or omit `started_at` (zero time). GitHub returns newest-first and the API always sets `started_at`, so no observed failure path. | code read of `fetchHeadReview`; no repro | UNTESTED (no equal-timestamp fixture) | tests (a fixture with two identical `started_at` runs would pin the tie rule) — surface only |
| Minor | SPECULATIVE | test environment | One unreproduced transient: the very first combined `go test` run in this fresh worktree reported `FAIL` for `internal/prtriage` (0.533s) with no visible case; six subsequent runs including `-race` pass. Cannot reproduce, no diagnosis. | observed once, this session, tool output truncated; not reproducible | UNTESTED | none (observation; re-run if it recurs) |
| Question | THEORETICAL | verification.md | The recorded live smoke evidence ("held #3410 … on a2c3c76 reads in_progress") reflects the state when it was captured; re-running `dotf pr triage-queue` live in this session holds #3410 for a different reason (1 REAL finding lacks reproduction) and no PR is currently head-review-held in dotfiles. The command's behavior still matches the spec (drafts #3403/#3399 listed only for reviewer output, release #3383 quiet, exit 1), but the exact recorded reading is no longer observable. | live run, this session (exit 1, 4 PRs listed) | `TestReviewOnHeadHoldsAQueueTheCommentsLeaveClear` proves the reading on fixtures | none (verification.md is outside the contract set; expected drift of live data, not an evidence defect) |

No Blocker or Major findings. Fail-closed behavior was probed adversarially and holds: an unreadable check-run page, a full 100-run page, an unknown head SHA, and a look-alike `preview` check all error or hold rather than reading clear; a superseded cancelled run does not hold; a PR pending for both reviewer output and a skipped review is refused by `pr land` through `reviewReasons` even though it is not tagged `HeadUnreviewed`; the exemption only quiets the queue, never `Decide` (a failed review on a release PR still refuses the merge via the generic red-check rule). No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain in the spec folder.

### Verification evidence (this session)
- `go test -count=1 ./...` in `cli/`: 59 packages ok (matches verification.md).
- `go test -count=1 -race ./internal/prtriage/ ./internal/prland/`: ok.
- `go vet ./...` and `GOOS=windows go vet ./...`: clean; `golangci-lint run` natively and with `GOOS=windows`: 0 issues; `gofmt -l`: empty.
- features.json's three `verification` commands: all ok.
- Mutations (each reverted, tree clean afterwards except spec deliverables): `withHeadReview` disconnected → `TestReviewOnHeadHoldsAQueueTheCommentsLeaveClear` + `TestReviewOnHeadLetsAnExemptDiffThrough` fail (12 subtests + 2 parents); `IsReviewCheck` widened to `Contains` → `TestIsReviewCheck`, the hold test, and `TestDecide_ReviewDidNotRunOnTheHead` fail; `withoutExempt` restored to re-evaluating comments → `TestReviewOnHeadLetsAnExemptDiffThrough` fails. Red-green confirmed: the new tests discriminate the fix, not just pass.
- Live smoke: `go run -C cli ./cmd/dotf pr triage-queue` — behaves per spec (see Question row).

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All three ACs verified with named tests including negative paths (failed read, full page, unknown head, look-alike, draft, exempt, superseded run); mutation checks prove the tests bite. |
| Verification       | A | Reproducible commands throughout (suite, vet/lint ×2 GOOS, three feature commands, three mutations, live smoke); every claim in verification.md reproduced or explained. |
| Scope              | A | One commit; diff matches proposal exactly; the `withoutExempt` rewrite is a required consequence of the head-review hold, documented in verification.md. |
| Reliability        | A | Every unreadable state errors or holds (fail-closed), fan-out is bounded, drafts cost no call, exemption only quiets, `pr land` never reports a head-review hold as untriaged output. |
| Maintainability    | A | Small focused functions (largest ~30 lines), one shared matcher as the single place both commands ask, comments explain why (incident and issue references). |
| Handoff-readiness  | A | Lesson-592 written in-diff, promotion candidates dispositioned with reasons, features.json contract present with all states `pending` awaiting harness capture (correct per the template's pass-state gating). |

### Verdict
PASS — no blockers, no REAL or THEORETICAL majors; rubric all A; two SPECULATIVE minors and one data-drift question surfaced above, each tracked and none verdict-moving.

`dotf spec archive` / `/spec archive` is **advisable** in the current state: the review is fresh, the contract digests match, and the last tasks.md box (this review + archive) is the only open item.

### Recommended next steps
- (verification.md, outside the contract set — for the implementer to disposition: applied / ticketed / declined)
  - Optionally add a fixture with two review runs sharing an identical `started_at` (and one with a missing `started_at`) to `TestReviewOnHeadHoldsAQueueTheCommentsLeaveClear` to pin the tie rule, or decline as GitHub-guaranteed-uniform data.
  - If the one-off `internal/prtriage` FAIL recurs in a fresh worktree, capture the case name and file it; do not treat this single unreproduced observation as a defect.
  - Re-run the live smoke after archiving if a durable record of the head-review hold on a live PR is wanted; current dotfiles state offers no head-review-held PR to observe.
