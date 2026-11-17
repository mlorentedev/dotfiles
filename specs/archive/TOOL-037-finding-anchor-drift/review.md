---
spec: "TOOL-037-finding-anchor-drift"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "250c2f58e9cd060cd0ebce7f282dde5244110695"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-13"
---
## Adversarial review

**Scope**: TOOL-037-finding-anchor-drift (spec commit ee159b97 of 3 in range; reviewed as the whole change `a3519fdf27c2a1991a7502a55221f93088dee699...HEAD`)
**Sources**: specs/TOOL-037-finding-anchor-drift/spec.md, specs/TOOL-037-finding-anchor-drift/review-request.json, `git show ee159b97`, `git diff a3519fdf...HEAD`, issues #3455 / #3421

### Spec and task alignment

Fast-track spec (contract set = `spec.md` only; `dotf spec digest TOOL-037-finding-anchor-drift` reproduces the recorded `spec.md` digest `8b170c96…5199` exactly — not stale). Checklist vs. diff:

- `[x]` `restInline.OriginalLine` carried into `Comment.OriginalLine` — confirmed (`fetch.go`).
- `[x]` `Comment.Anchors()` + `rowFor`/`rowForLine` — confirmed (`prtriage.go`, `reproduction.go`); `Anchors()` returns the reviewer's `original_line` when GitHub reports one, the current `line` only otherwise — matches Scope bullet 2, including the decision *not* to accept either anchor.
- [x] `Comment.Where()` used by both `dotf pr` print sites — confirmed by grep (`pr.go:112`, `pr.go:147`), but only asserted in tests through the `Line`-only fallback; see Finding 1.
- [x] The three named tests exist and pass (`go test -run 'TestFetchKeepsTheOriginalLine…|TestFetchMatchesARowOn…|TestFetchDoesNotClear…'` → PASS).
- [x] Lesson 598 present and consistent with the final design.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags in the spec folder.
- Spec's out-of-scope claim verified against reality: issue #3455's comment ("Left open by #3457: a file-level review comment has neither `line` nor `original_line`… `rowFor` needs a path-only match") exists verbatim.

Range interaction: the other two commits in the range (0ad02212 mise/Git Bash, 250c2f58 fleet-lint / CI-019) touch `tools/` and `fleetlint`/`cmd/lint_fleet.go` only — no overlap with `prtriage` or `pr.go` logic beyond sharing the `cmd` package test run, which is green.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Major | THEORETICAL | tests / acceptance | Scope bullet 3 ("queue prints a finding as the reviewer did") and checklist item 3 have no test asserting the *output*. Probe A (revert `pr.go`'s two print sites to the pre-fix format — `REAL %s:%d` and the `Line>0` branch) leaves the **entire** `go test ./internal/cmd/` suite green (58.3s, `ok`). `Where()`'s `OriginalLine` branch is never asserted anywhere: every existing output test feeds `Line`-only comments, where old and new code print identically. A regression re-introducing the incident's headline symptom (`…verification.md:0`) would pass CI silently. The shipped code is correct today, so the failure itself is unobserved → THEORETICAL; the coverage hole is probe-demonstrated this session. | probe A (this session); `pr_triage_scoped_test.go` uses `Line`-only comments; `TestPrTriageQueue_NamesUnreproducedFindings` uses `Line: 4` | **UNTESTED** — needs a named case (e.g. an `Unreproduced` finding with `Line: 0, OriginalLine: 39` asserting `verification.md:39`, never `:0`) | tests (outside contract set; disposition in `verification.md`) |
| Minor | REAL (spec-declared) | scope | A file-level REAL finding (both anchors null) can never match a row → PR held indefinitely. Explicitly out of scope in `spec.md`; the promised "#3455 note" exists verbatim (verified via `gh issue view 3455`). Fail-closed, tracked — not a spec violation. | spec Scope + #3455 comment | UNTESTED (out of scope by declaration) | spec follow-up already noted; re-check if a reviewer starts posting file-level REAL findings |
| Minor | THEORETICAL | compat / migration | Rows written under the **old** key (current line, after a shift, as the pre-fix queue printed it) stop matching once this lands: a previously cleared PR re-holds until its row is re-keyed to the reviewer's line. Intended fail-closed direction, and `TestFetchMatchesARowOnTheReviewersLineOfAShiftedFinding` "the current line" case proves the behavior (`held=1`), but the spec does not call out the one-time friction for PRs triaged pre-fix. | code read of `rowFor` key change; no incident cited | `TestFetchMatchesARowOnTheReviewersLineOfAShiftedFinding` ("the current line" case) | spec/verification note or follow-up ticket — **not** a `spec.md` edit under this verdict |
| Minor | REAL (code read) | spec/docs drift | Checklist says "`rowFor` trying each anchor", but the final `Anchors()` returns at most one element, so the `rowFor` loop can never iterate twice — residue of the intermediate two-anchor design. The commit message also cites `restInline.anchor()`, which does not exist (the method is `Comment.Anchors()`). Cosmetic; the Scope bullet itself is accurate. | `prtriage.go` `Anchors()`; `git show ee159b97` message | n/a (doc) | spec wording / commit prose — contract set: disposition, do not edit under a passing verdict |

No test was deleted or weakened in this diff; all test changes are additive.

**Severity × reality**: the single Major is THEORETICAL (correct code today; a probe shows a future regression would be undetected), so it is tracked with a disposition, not blocking. No Blocker, no REAL Major.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Gate criteria verified with three named tests plus two mutation probes (fail-open and fallback removal both caught); one display criterion lacks test coverage (Finding 1). |
| Verification       | B | build/vet/test/lint + probes all reproduced this session; the live `triage-queue --pr 3421` leg of the Evidence can no longer be re-run (#3421 is closed) — historical only, marked UNVERIFIED. |
| Scope              | B | Spec's commit matches the spec exactly; the two unrelated commits in the range (CI-019, mise lesson) do not interact with this change. |
| Reliability        | B | Fail-closed throughout (unmatched finding → held); the only permanently unmatched shape (file-level REAL) is explicitly scoped out and ticketed on #3455. |
| Maintainability    | B | Small functions, `golangci-lint` 0 issues, gofmt clean; minor dead generality in the single-element `Anchors()` loop. |
| Handoff-readiness  | A | Spec + lesson 598 + promotion candidates + explicit Next step written in the same change. |

Aggregation: no D, no C → all B or above; severity axis carries the THEORETICAL Major → **PASS WITH GAPS**.

### Evidence (produced this session)

```
go build ./... && go vet ./internal/prtriage/ ./internal/cmd/     -> ok (VET_OK)
go test ./internal/prtriage/ ./internal/cmd/ -timeout 5m         -> ok, 0 failures (fresh, final state)
golangci-lint run ./internal/prtriage/... ./internal/cmd/...      -> 0 issues
probe B: Anchors() ignoring original_line  -> all 3 named tests FAIL (guard proven)
probe C: Anchors() returning both anchors  -> TestFetchDoesNotClearAFindingOnAnotherFindingsRow
                                               + shifted-current-line case FAIL (fail-open guarded)
probe A: pr.go print sites reverted        -> full ./internal/cmd/ suite ok 58.3s (coverage hole proven;
                                               tree reverted, `git status` clean outside spec folder)
dotf spec digest TOOL-037-finding-anchor-drift                    -> spec.md digest matches review-request.json
gh issue view 3455 (body + comments)                              -> file-level note verified verbatim; issue CLOSED
go run ./cmd/dotf pr triage-queue --pr 3421                       -> refuses: "#3421 is closed" — the live
                                               Evidence legs are no longer re-runnable (UNVERIFIED as stated)
```

### Verdict

**PASS WITH GAPS** — no Blocker, no REAL Major; one THEORETICAL Major (print-site output coverage, UNTESTED) plus three Minors, each with a disposition line above. Rubric all B+.

### Recommended next steps

Route each to a disposition (applied / ticketed / declined with a reason) in `verification.md` — none of these is a contract-set edit, and `spec.md` must stay untouched under this verdict:

1. Add a named regression test for Finding 1: an `Unreproduced` finding with `Line: 0, OriginalLine: 39` asserted to print `verification.md:39` (never `:0`), at the `cmd` output surface where the incident was first reported. Outside the contract set → does not invalidate this review.
2. Record Finding 2 (pre-fix rows keyed by a shifted current line re-hold once) as a known one-time friction in `verification.md`, or file a follow-up ticket if a real PR trips it.
3. Finding 3's follow-up trigger (a reviewer posting file-level REAL findings) is already tracked on #3455; no action until then.
4. Finding 4 is prose drift; fix at the next spec edit or leave.

**Archive**: `dotf spec archive TOOL-037-finding-anchor-drift` is **advisable** once #3457 is merged to main, provided the gaps above are dispositioned in `verification.md` first; this fresh, passing `review.md` satisfies the review pre-flight. The UNVERIFIED live-#3421 evidence legs cannot be re-run (PR closed) — the unit tests and the probes in this review carry that load.
