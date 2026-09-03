---
spec: "CLI-071-triage-queue-transport"
verdict: "PASS"
reviewed_sha: "afc9db6021763589f695c9bf9e1ecc898d4d1177"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-03"
---

## Adversarial review

**Scope**: CLI-071-triage-queue-transport / PR #1457 (merged), HEAD `afc9db6`
**Sources**:
- `specs/CLI-071-triage-queue-transport/{proposal.md,tasks.md,verification.md,features.json}`
- `cli/internal/prtriage/{fetch.go,fetch_test.go,prtriage_test.go}`
- `git diff fb0b7e9..afc9db6` (10 files, +1027/−76)
- `gh pr view 1457` (merged; head `fix/cli-071-triage-queue-transport`; body names `specs/CLI-071-triage-queue-transport/` and `Refs #1454`)
- Live REST verification against `mlorentedev/dotfiles`

### Spec and task alignment

Every acceptance criterion in `proposal.md` maps to a named test and a `features.json` entry. Task boxes in `tasks.md` are ticked downstream of the criteria. One box is unticked (see Minor-3). `proposal.md` `status:` is `verifying`, which is correct for the verification window. All out-of-scope items (distinct exit statuses, the triage algorithm, the skill/registry) were left untouched — `prtriage.go` (Evaluate/reviewOutput/lastTriage/normaliseLogin) has no diff in this change, so the algorithm and the contract the package must not erode are unchanged. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain in the three contract files.

### What I ran (independent of the author's claims)

- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` → clean (exit 0).
- `go test ./... -count=1` → all packages ok, no failures.
- `go test -race -count=3 ./internal/prtriage/` → clean (the package's only concurrency).
- `golangci-lint run` (v2.12.2, the `versions.conf` pin) → 0 issues.
- `gofmt -l internal/prtriage/` → clean.
- All seven `features.json` `verification` commands reproduced → `PASS` for each named test, using the `--- PASS:` grep form (confirmed non-vacuous: a `-run` no-match exits 0, so the grep form is what makes them broad).
- Mutation battery — seven edits, each reverted, each caught by its named test:
  | Mutation | Caught by |
  |---|---|
  | drop the `[bot]` fold in `normaliseLogin` | `TestFetchFoldsTheBotLoginSuffix` (queue empties) |
  | map `url` instead of `html_url` | `TestFetchWithRunnerDrivesTheWholePath` (`URL = ""`) |
  | `commentFanout = 1` (serialise) | `TestFetchFansOutConcurrently` (peak 1; elapsed ≥ serial) |
  | remove the comment page guard | `TestFetchRefusesTruncationOnBothAxes` |
  | remove the pull-request page guard | `TestFetchRefusesTruncationOnBothAxes` |
  | respawn `gh pr list` (GraphQL) | `TestFetchUsesRESTOnly` |
  | swallow a per-PR comment error | `TestFetchReportsFailureRatherThanAnEmptyQueue` |
- Live: built `./cmd/dotf`, ran `dotf pr triage-queue` in the repo → `[OK] no reviewer output is awaiting a disposition`, exit 0. Verified this is the *correct* verdict, not a green paint: PR #1419 has a `coderabbitai[bot]` review at `2026-09-02T02:09:11Z` and a re-triage at `2026-09-03T02:33:33Z` (the newer `## Review triage` supersedes), so `Evaluate` correctly returns "triaged".

### The one real correctness question — comment-set equivalence

The transport's central assumption is that `GET /repos/{o}/{r}/issues/{n}/comments` returns the *same* set the retired `gh pr list --json comments` returned. I tested it live against PR #1419:

- `gh pr list --json comments` → `['coderabbitai', 'mlorentedev', 'mlorentedev']` (3 comments)
- `gh api .../issues/1419/comments` → `['coderabbitai[bot]', 'mlorentedev', 'mlorentedev']` (3 comments)

Same set, same order; only the `[bot]` suffix differs, which `normaliseLogin` folds back to the bare registry login before comparison. So the join is correct and AC3's fold is confirmed load-bearing against real data, not merely a fixture.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Minor | THEORETICAL | maintainability | `fetchWith` is 53 lines, exceeding the repo's <40-line function guideline, though cyclomatic complexity is low (~7) and the structure (wind-up / bounded fan-out / reconcile) is clear. | `awk` span `102..154` for `fetchWith`; complexity from a manual read of the control flow; no complexity tool installed to confirm the exact count. | covered by the existing suite after a behavior-preserving extraction (all tests pass) | code — extract the per-PR fan-out into a helper to bring it under 40 lines |
| Minor | REAL | bookkeeping | `tasks.md` leaves `[ ]` on "PR opened referencing this spec folder with `Refs #1454` (not `Closes`)" although PR #1457 is merged and references both the spec folder and `Refs #1454`. | `gh pr view 1457` body contains `Spec: specs/CLI-071-triage-queue-transport/` and `Refs #1454`; head branch is `fix/cli-071-triage-queue-transport`. | UNTESTED (process bookkeeping, not testable) | spec artifacts — tick the box so the audit trail matches reality |
| Minor | SPECULATIVE | verification | AC5's literal "completes inside the 5s context" is not tested as a real 5s `context.WithTimeout` deadline; it is inferred from the observed-concurrency ceiling plus the author's live latency measurement. The budget *is* enforced (mem.go wraps the fetch in a real `5s` deadline) and the truncation guard caps the worst case, but a repo near the 100-PR cap with slow network could approach it. | concurrency cap `commentFanout=8`; truncation guard fires at `>=100`; author's live measurement 0.94s for 1+2 calls; no test injects a 5s deadline. | `TestFetchFansOutConcurrently` (covers peak concurrency, not a real deadline) | tests (optional) — a 5s-deadline test would make it literal, but is arguably over-engineering |
| Minor | THEORETICAL | UX | The comment-axis truncation error gives no remediation hint, unlike the PR-list axis which suggests `gh api .../pulls --paginate`. | read of `fetch.go`: the pull-request guard names the `--paginate` command; the comment guard names no workaround. | `TestFetchRefusesTruncationOnBothAxes` (covers the refusal, not the wording) | code — cosmetic wording only |
| Minor | THEORETICAL | scope/limit | Any repo with >=100 open PRs, or a PR with >=100 conversation comments, is not processable by the tool — it fails loud rather than answering from partial data. | `prLimit == commentLimit == 100`; a full page always refuses. | `TestFetchRefusesTruncationOnBothAxes` | spec — documented limitation; a pagination feature is a separate change |
| Minor | REAL (pre-existing) | error clarity | On a `gh` non-zero exit the surfaced message hides gh's own stderr reason (`HTTP 403`, `Not Found`); the user sees `gh api pulls: exit status 1` with no cause. Pre-existing — the prior `gh pr list` message had the same shape — and explicitly scoped out in `verification.md`. | verified by reading `execGH` (`.Output()` discards stderr); author documents it as out of scope. | UNTESTED (a two-line change to unwrap `*exec.ExitError`) | code — belongs to whoever owns the error contract, not this transport |

No Blocker, no Major. The `atl` (comment-set) question is resolved by live evidence rather than left open.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | A | All six criteria verified; negative paths covered (both truncation axes, error-not-empty, bot-login fold, empty-repo default); the one real mapping question confirmed against live data; no observed defect. |
| Verification | A | Every criterion reproducible with the command + output I re-ran here; `features.json` commands are non-vacuous; mutation battery (7 edits) all caught. |
| Scope | A | Diff matches the proposal exactly — prtriage transport + seam, one justified renderer-fixture string, the declared lesson; no unrelated files, nothing eroding the out-of-scope contracts. |
| Reliability | A | Read-only, idempotent by nature; list/per-PR failures and both truncation axes return errors instead of silently truncating; the shared exit status contract preserved. |
| Maintainability | B | Naming clear, comments explain WHY, cyclomatic complexity low, no dead code — but `fetchWith` is 53 lines, over the <40-line guideline (the one reason it is not A). |
| Handoff-readiness | A | Spec fully populated; the promised lesson captured in `docs/lessons/`; promotion candidates and archive checklist recorded; next steps unambiguous. |

### Verdict

**PASS** — no Blocker or Major; the rubric is all A with a single B (Maintainability, purely the 53-line `fetchWith`), so the mechanical aggregation is PASS. The change is correct, well-tested, scoped cleanly, and its only real correctness risk (comment-set equivalence) is confirmed against live data. The findings above are Minor and non-gating; they are listed for follow-up, not as pre-archive requirements.

`dotf spec archive` is **advisable** in the current state: the review is fresh on the current `reviewed_sha`, the contract files are unchanged since it, and the pool string `nan/deepseek-v4-flash` matches the registry.

### Recommended next steps (before archive)

- Optional (Maintainability): extract the per-PR fan-out out of `fetchWith` into a helper so it meets the <40-line guideline. Behavior-preserving; the existing suite would catch a regression.
- Optional (bookkeeping): tick the `[ ]` "PR opened referencing this spec folder" box in `tasks.md`, since PR #1457 does reference it (`Refs #1454`) — reconcile the audit trail.
- Optional (tests): add a literal 5s-deadline test for AC5 if "inside the 5s context" must be asserted as a timeout rather than as an observed-concurrency ceiling.
- Optional (UX): give the comment-axis truncation error a remediation hint, and surface gh's stderr on non-zero exit — the latter is a separate error-contract decision and can wait.
- Before archiving: run `dotf spec archive` (it will verify the gate, move the folder to `specs/archive/`, and require the promotion steps the author has already prepared).
