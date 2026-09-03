---
spec: "CLI-071-triage-queue-transport"
verdict: "PASS"
reviewed_sha: "f3c4b88ce358e6bc863d3133a10c1ae169e0fd39"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-03"
---

## Adversarial review

**Scope**: CLI-071-triage-queue-transport / PR #1457 (merged) + the follow-up `f3c4b88` that applied the prior review's findings. HEAD `f3c4b88`.
**Sources**:
- `specs/CLI-071-triage-queue-transport/{proposal.md,tasks.md,verification.md,features.json}`
- `cli/internal/prtriage/{fetch.go,fetch_test.go,prtriage.go,prtriage_test.go}`
- `cli/internal/cmd/{pr.go,mem.go}`, `cli/internal/mem/session_start_triage_test.go`
- `git diff fb0b7e9..HEAD` (12 files, +1158/−75 — the full change under review)

This is a **re-review of the current state**, not of the earlier `afc9db6` snapshot: the previous review passed against `afc9db6` (rubric A/A/A/A/**B**/A, six Minor), and the author then committed `f3c4b88` applying four of them (Maintainability extraction, tasks.md bookkeeping tick, comment-axis remediation hint) and declining two with written reasons. The prior `review.md` is now stale against HEAD, so I re-verified from the working tree.

### Spec and task alignment

Every acceptance criterion maps to a named test and a `features.json` entry. The algorithm in `prtriage.go` (`Evaluate`/`reviewOutput`/`lastTriage`/`normaliseLogin`/`Queue`) has **no diff** in this PR, and the exit-status contract in `cmd/pr.go` has **no diff** — so the two explicitly out-of-scope contracts (the triage algorithm, the shared exit status) are preserved exactly. No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in the three contract files. `proposal.md` `status:` is `verifying`, correct for the verification window. The tasks.md "PR opened" box is now `[x]` (Applies the prior finding #2), and the comment-axis truncation error now carries a `--paginate` remediation hint (finding #4 applied).

### What I ran (independent of the author's claims)

- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` → clean (exit 0).
- `gofmt -l internal/prtriage/` → clean.
- `go test ./... -count=1` → all packages ok, no failures.
- `go test -race -count=3 ./internal/prtriage/` → clean (the package's only concurrency).
- `golangci-lint run` (v2.12.2, the `versions.conf` pin, installed) → 0 issues.
- All seven distinct `features.json` `verification` test names reproduced → `PASS` each, using the `--- PASS:` grep form. **Confirmed non-vacuous**: `go test -run '^TestThisDoesNotExist$'` exits 0 with "no tests to run", so the plain-`-run` form would be unfailable; the grep form is what makes these broad (the author's trap is real and this change avoided it).
- **Mutation battery** — seven edits, each reverted to `git diff --quiet` after, each caught by its named test:
  | Mutation | Caught by |
  |---|---|
  | drop the `[bot]` fold (`TrimSuffix`) in `normaliseLogin` | `TestFetchFoldsTheBotLoginSuffix` (queue empties) |
  | map `url` instead of `html_url` (`URL: ""`) | `TestFetchWithRunnerDrivesTheWholePath` (`URL = ""`) |
  | `commentFanout = 1` (serialise) | `TestFetchFansOutConcurrently` (peak 1; elapsed ≥ serial) |
  | remove the comment page guard (`if false`) | `TestFetchRefusesTruncationOnBothAxes` |
  | remove the pull-request page guard (`if false`) | `TestFetchRefusesTruncationOnBothAxes` |
  | respawn `gh pr list` (GraphQL) | `TestFetchUsesRESTOnly` (asserts request strings) |
  | swallow a per-PR comment error (error-return loop neutralised) | `TestFetchReportsFailureRatherThanAnEmptyQueue` |
- **Live, read-only**: built `./cmd/dotf`, ran `dotf pr triage-queue` in the repo → `1 pull request(s) awaiting a disposition: #1459 … github-actions reviewed, never triaged`, exit 1. Independently verified this is the **correct** verdict, not green paint:
  - PR #1419 is correctly **not** pending: `gh api .../issues/1419/comments` shows `coderabbitai[bot] @ 2026-09-02T02:09:11Z`, then `mlorentedev @ 2026-09-03T02:33:33Z ## Review triage` (a later re-triage), so `Evaluate` returns "triaged".
  - PR #1459 is correctly **pending**: `github-actions[bot] @ 2026-09-03T03:36:13Z ## PR Reviewer Guide` with no subsequent triage, so `Evaluate` returns "github-actions reviewed, never triaged".
  - **Comment-set equivalence** (the transport's central assumption) confirmed: `GET /repos/{o}/{r}/issues/{n}/comments` returns the reviewer output `gh pr list --json comments` produced, differing only by the REST `[bot]` suffix that `normaliseLogin` folds to the registry's bare login. The join is sound against real data, not only a fixture.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Minor | THEORETICAL | testability | AC1 says `FetchWithRegistry` runs end-to-end in a test, but the test drives `fetchWith`, not `FetchWithRegistry`. The exported `Fetch` / `FetchWithRegistry` that callers actually use (pr.go:61, mem.go:206) are covered only transitively, so a regression introduced in the one-line wrapper (`FetchWithRegistry → fetchWith(ctx, execGH, …)`) would go uncaught by any named test. | grep: no call to `FetchWithRegistry`/`Fetch` in any `*_test.go`; every test calls `fetchWith` directly. The wrapper is one line and the live smoke test covers the real path, so this is a coverage-breadth nit, not a defect. | UNTESTED (no named test drives `FetchWithRegistry`/`Fetch`) | tests — add a small test calling `FetchWithRegistry` with a fake; or spec — reword AC1 to name `fetchWith` |
| Minor | SPECULATIVE | latency | AC5 asserts an observed concurrency ceiling (peak ≥2, ≤8) and `elapsed < serial`, not a literal 5s `context.WithTimeout` deadline. A repo near the 100-PR cap with slow network could approach the 5s budget; the real deadline is enforced in `mem.go` and a timeout fails loud (never silent/empty), so the mechanism is safe but "completes inside the 5s context" is inferred, not asserted by a deadline test. | `commentFanout=8`; truncation guard at `≥100`; live measurement ~0.95s for 1+2 calls; no test injects a 5s deadline. | `TestFetchFansOutConcurrently` (peak concurrency; not a real deadline) | — (author declined a literal deadline test as documented over-engineering; non-gating) |
| Minor | THEORETICAL | limit | A repo with ≥100 open PRs, or any PR with ≥100 issue comments, is not processable — it fails loud rather than answering from partial data. This is the designed behaviour and is recorded as an explicit documented limitation. | `prLimit == commentLimit == 100`; a full page always refuses. | `TestFetchRefusesTruncationOnBothAxes` | spec — documented limitation; a pagination feature is a separate change |
| Minor | REAL (pre-existing) | error clarity | On a `gh` non-zero exit the surfaced message hides gh's own stderr reason (`HTTP 403`, `Not Found`); the user sees `gh api pulls: exit status 1` with no cause. Pre-existing — the prior `gh pr list` message had the same shape — and explicitly scoped out in `verification.md` and the PR body before the review ran. | `execGH` uses `.Output()`, which discards stderr; author documents it as out of scope. | UNTESTED (a two-line change to unwrap `*exec.ExitError`) | code — belongs to whoever owns the error contract, not this transport |

No Blocker, no Major. The prior review's sole Maintainability deduction (the 53-line `fetchWith`) is resolved here, so it no longer appears. The prior four *applied* findings are all confirmed in the current tree.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | A | All six criteria verified; negative paths covered (both truncation axes, error-not-empty, bot-login fold, empty-repo default); the one real mapping question (comment-set equivalence) confirmed against live data; the out-of-scope algorithm and exit contract are untouched; no observed defect. |
| Verification | A | Every criterion reproducible with the exact command + output I re-ran here; `features.json` commands confirmed non-vacuous against the `-run` no-match trap; 7-edit mutation battery each caught by a named test; the live verdict independently confirmed correct rather than green paint. |
| Scope | A | Diff matches the proposal exactly — prtriage transport + seam, the justified renderer-fixture string, and the declared lesson; `prtriage.go` and `pr.go` untouched; no unrelated files, nothing eroding the out-of-scope contracts. |
| Reliability | A | Read-only and idempotent; the list call, any per-PR call, and both truncation axes return errors instead of silently truncating; context cancellation fails loud (never an empty queue); the shared exit-status contract preserved. |
| Maintainability | A | Naming clear, comments explain WHY, low cyclomatic complexity, no dead code introduced — and the single prior blemish (53-line `fetchWith`) is fixed by the `withComments` extraction, leaving `fetchWith` ~21–29 lines and the helper 38, both under the <40 guideline. |
| Handoff-readiness | A | Spec fully populated; the promised lesson captured and indexed (`lesson-260-the-path-with-no-seam-is-the-path-with-no-test.md`); promotion candidates and archive checklist recorded; next steps unambiguous. |

### Verdict

**PASS** — no Blocker or Major; the rubric is all A. The change is correct, well-tested to a breadth unusual for a thin transport change, scoped cleanly, and its only real correctness risk (comment-set equivalence) is confirmed against live data. The remaining findings are Minor and non-gating: one coverage-breadth nit (the exported `FetchWithRegistry` is covered transitively), one already-declined latency-test question, one documented hard limit, and one pre-existing error-clarity item that was disclosed and scoped out before this review ran. All four are listed for follow-up, not as pre-archive requirements.

`dotf spec archive` is **advisable** in the current state: the review is fresh on the current `reviewed_sha` (`f3c4b88`), `proposal.md` / `tasks.md` / `features.json` are unchanged since then (the staleness watch is clean), and the pool string `nan/deepseek-v4-flash` matches the registry.

### Recommended next steps (before archive)

- Optional (tests): add a small test that calls the exported `FetchWithRegistry` (or `Fetch`) with a fake, closing the only coverage-breadth gap — one line of wiring and one behavior-preserving assertion.
- Optional (spec): reword AC1 to name `fetchWith` (what the test actually drives) instead of `FetchWithRegistry`, which is covered only transitively — or leave it; the intent ("the fetch path is testable") is satisfied.
- No gating action required for the two declined items: the literal-5s-deadline test and surfacing gh's stderr are decisions already recorded with reasons in `verification.md`, not withholdings.
- Before archiving: run `dotf spec archive` — it will verify the gate, move the folder to `specs/archive/`, and require the promotion steps the author has already prepared (lesson captured / indexed; board ticket closed).
