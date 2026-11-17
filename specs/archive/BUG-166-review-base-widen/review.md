---
spec: "BUG-166-review-base-widen"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "6a18a1911f3c8d1f8f0da3493ddeef8d459d22d5"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: BUG-166-review-base-widen (whole change: `git diff 3ef5dfa6b7386ee2a39cb25e129caf915b3a119e...HEAD`)
**Sources**: `specs/BUG-166-review-base-widen/{proposal,tasks,verification}.md`, `features.json`, `review-request.json`, branch `fix/review-base-range` at `6a18a19`. Two first-parent commits on the branch (`153768da` implementation, `6a18a19` proposal docs); no merge into the branch, so the range holds only this spec's work.

### Spec and task alignment

Contract digests in `review-request.json` match `dotf spec digest BUG-166-review-base-widen` exactly (features.json, proposal.md, tasks.md) — the request describes the contract files as they stand. In this repository the widened base equals the traced base (`merge-base HEAD origin/main` = `3ef5dfa`, the parent of the spec-adding commit `153768da`), so this review's own scope is the correct whole change.

- **AC1** — covered, mutation-verified: `TestReviewBaseWidensToTheForkPoint` (base = fork point, `git diff --stat` holds `x.go`, `WidenedFrom` = traced) and `TestOwnCommitsCountTheWorkBeforeTheSpec` (the issue-less fix is own, 2 of 2). Removing the widening block turns both red; removing only the `OwnCommitsWidened` widened-rule turns the own test red (fix disappears from the own list). Both reverted, tree clean.
- **AC2** — covered, mutation-verified: `TestReviewBaseNeverNarrows` subtests "on main after a squash", "main merged into the branch after the spec" (HARNESS-142 shape), "no origin ref" each keep the traced base with empty `WidenedFrom`. Deleting the `isAncestor` guard turns the first two red — the guard is load-bearing, not decorative. A reviewer-built scratch repo for the proposal's remaining risk (local main ahead of a stale `origin/main`, spec in the later commit) widened to the stale fork with 3 of 3 commits own, exactly as the proposal's Risks section resolves it.
- **AC3** — covered: full `go test ./...` green; the `TestResolveReviewBase*` / `TestResolveReviewBaseDetail*` refusal and BUG-108 rename tests pass with assertions unchanged (the diff only widens two test-helper stub signatures). None of those test repos carries an `origin/*` ref, so they exercise the trace alone.
- **AC4** — covered, reproduced independently: `TestReviewScopeSaysTheBaseWasWidened` asserts both the `describeScope` line and the prompt's widened note. A dev build of this branch run as `dotf spec review X-42-a --dry-run` in a freshly built #3258-shaped repo printed `Scope: git diff 6d11aa97...HEAD (2 of 2 commits are the spec's own; ...; base widened from 32f13e020df1 to where the branch left the default branch)`, `git diff --stat <base>...HEAD` listed the production `x.go`, the prompt carried the widened paragraph, and no worktree was created.

Evidence commands run this session: `go build ./... && go vet ./... && go test ./...` (all packages ok), `golangci-lint run ./...` (0 issues), `GOOS=windows go vet ./...` (0), `GOOS=windows golangci-lint run ./...` (0 issues), the four `features.json` verification commands (all ok), three mutation probes (each red on its named test, reverted). No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in any contract file (the string appears only inside this run's own `review-transcript.jsonl`, which quotes the launcher prompt).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | reviewer prompt wording | On a widened review, `writeOwnCommits` still says the own commits are the ones that "touch the spec folder under any of their names" and closes with "A slice that neither touches the folder nor names the issue is not listed" — both false for the pre-spec fix the widening adds, contradicting the widened paragraph printed just above | observed in the dry run: `32f13e020df1 fix(x): stop the leak` is listed under "(they touch the spec folder under any of their names)" though it touches no spec file and names no issue | UNTESTED — `TestReviewScopeSaysTheBaseWasWidened` asserts the widened note but no test asserts the rule text under widening | code (prompt wording, `review_launch.go`/`cmd/spec.go`); disposition in `verification.md` or a follow-up ticket |
| Minor | SPECULATIVE | resolver robustness | `forkPoint` returns `""` when `merge-base` fails for the first *existing* ref instead of trying the remaining refs; only observable if `origin/HEAD` resolves yet has no merge base with HEAD while `origin/main` would (misconfigured symref to an unrelated branch) — effect is the safe one (base stands) | code read of `forkPoint`; not reproduced, no plausible standard configuration triggers it | UNTESTED | code (`continue` instead of `return ""`); track as a follow-up |
| Minor | REAL (gap), behavior probe-verified | test coverage | Two proposal-resolved risk scenarios have no committed regression test: stale `origin/main` behind local main with the spec in the later commit, and the BUG-108 rename trace composed with widening (rename + `origin/*` ref in one repo). Both pass reviewer-built scratch probes this session; only the three AC2 scenarios are committed | reviewer probes: rename+widen probe green (base widened, both folder names traced, 3 of 3 own); stale-ref dry run green as specified | UNTESTED (no committed test; reviewer probe only) | tests (follow-up ticket) |

No Blocker and no Major was found. The widen-only invariant, the refusal paths, and the scope-line/prompt claims are each backed by a named test that I proved goes red under mutation; the two end-to-end probes confirm the launcher behavior the ACs describe.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All four ACs verified with mutation-checked tests plus two end-to-end scratch probes; the only defect found is inaccurate prompt wording on widened reviews (minor, presentation only). |
| Verification       | A | Every claim in `verification.md` re-run and reproduced this session, including the dry run, both mutation checks, lint and cross-GOOS vet. |
| Scope              | A | Diff is exactly the proposal's change plus its contract files; two branch commits, no unrelated changes, no creep. |
| Reliability        | A | Every git failure path fails safe (base stands, never moves later); refusals and rename trace unchanged and green; provenance/archive consumers unaffected (`checkReviewProvenance` does not re-resolve the base). |
| Maintainability    | B | Small functions, WHY-comments, lint clean; minus one point for the now-inconsistent own-commit rule text in the prompt. |
| Handoff-readiness  | B | Complete proposal/tasks/verification triad with named evidence and promotion decisions; PR box still open (expected pre-PR), and this review's three minors need dispositions. |

### Verdict

**PASS WITH GAPS** — no Blockers, no Majors; rubric all B or above; three Minor findings tracked, each with a disposition line below. Verdict = the more severe of the two paths (rubric B+ / severity minors-only) → PASS WITH GAPS.

Dispositions (for the implementer to record in `verification.md` — applied, ticketed, or declined with a reason; none of these touches the contract set):

1. Prompt rule-text under widening (Minor/REAL) — proposed disposition: ticket a wording fix so `writeOwnCommits` names the third rule when `scope.WidenedFrom != ""`.
2. `forkPoint` ref-iteration nit (Minor/SPECULATIVE) — proposed disposition: decline or fold into the wording ticket; the failure mode is the safe one.
3. Missing committed tests for the stale-ref-ahead and rename+widen compositions (Minor, probe-verified) — proposed disposition: ticket as follow-up tests.

### Recommended next steps

- Record the three dispositions above in `verification.md` (it is excluded from the staleness check for exactly this purpose); no `proposal.md` / `tasks.md` / `features.json` change is proposed, so this verdict stays valid as written.
- Open the PR referenced by the still-unticked `tasks.md` closing box; the review itself is complete and does not block it.
- `dotf spec archive BUG-166-review-base-widen` is advisable once the dispositions are recorded: contract digests match, no draft tags remain, and this review carries a passing verdict. The archive gate itself was not executed (it would move the folder); its behavior here is UNVERIFIED by this run.
