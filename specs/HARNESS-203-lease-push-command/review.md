---
spec: "HARNESS-203-lease-push-command"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "2cae9c0721b691cd18667a790b5d95ad7acf62d0"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-23"
---
## Adversarial review

**Scope**: HARNESS-203-lease-push-command (#3612), whole branch change.
**Sources**: `specs/HARNESS-203-lease-push-command/{proposal,tasks,verification,features}.md|json`, `git diff 31360ae33a738ab309af04a302cb8c71084d3e4d...HEAD` (29 files, the 20 spec commits listed by the launcher — no merge-in work appeared in the range), `cli/internal/pushlease/pushlease.go` + tests, `cli/internal/cmd/pr_push_lease.go` + tests, `cli/internal/claude/push_lease_rules_test.go`, `docs/adr/adr-066-*.md`, `docs/lessons/lesson-615-*.md`, `AGENTS.md`, prior rounds `review-round1..7.md` and `review-second-round1.md`.

`contract_digests` in `review-request.json` equal the output of `dotf spec digest HARNESS-203-lease-push-command` (all three identical at review start), so the contract set is the one this review describes.

### Spec and task alignment

- All `[x]` implementation tasks carry diff evidence; the single open box (`tasks.md` Closing, PR + signatures line) is open by design — it waits on the signature rounds, two of which (`review-round6/7.md`) died without verdicts (#3615 class).
- AC1 ↔ `TestPush_MovesTheRemoteAfterARebase`, `TestPush_StaleExpectLeavesTheRemote` (2 rows), `TestPrPushLease_PushesAndPrintsTheMove`. AC2 ↔ `TestPrepare_Refuses` (17 rows), `TestPrepare_RemoteResolutionOrder`, `TestPrepare_RefusesASplitRemote`, `TestPrPushLease_Refusals`. AC3 ↔ `TestPush_PrePushHookRuns`. AC4 ↔ `TestPushLeaseRules_*` (3 tests), the `AGENTS.md` line (line 291, outside the generated block 208–268), ADR-066 with an `## Options considered` section covering all three alternatives.
- I re-ran, this session: `go test ./internal/pushlease ./internal/claude -count=1 -timeout 8m` → `ok … 51.5s` / `ok … 1.5s`; `go test ./internal/cmd -run TestPrPushLease -count=1` → `ok … 24.3s`; `go build ./... && go vet ./...` → clean; `gocyclo -top 5 internal/pushlease` → max 8 (`Prepare` 8, `resolveRemote` 8), under the bar of 10 (the second signature's F4, verified rather than taken on trust).
- Mutation probes (three mutants applied together in a **temp clone** outside the repo, tree left clean): split guard off, stale pre-check disabled, `--no-verify` added. All three were killed by exactly the claimed named tests: `TestPush_StaleExpectLeavesTheRemote/the_remote_already_equals_HEAD…` failed with `Everything up-to-date` + exit 0; `TestPush_PrePushHookRuns` failed with the push landing; both rows of `TestPrepare_RefusesASplitRemote` failed with the push target's `feat/x` force-updated — which is finding F1's damage reproduced under the mutant.
- The prior REAL finding F1 (pushurl vs fetch URL) is fixed in `resolveRemote` (`get-url --push --all` vs `get-url --all`, refusal before any network call, reason names the remote not a URL) with two fixture rows plus the `PushURLEqualToURLStillPushes` guard test — the mutant run proves both halves bite.
- Spec-vs-code: no mismatch found in the current contract. The proposal's `--expect`/refusal/argv descriptions match `pushlease.go` line by line; `ai/claude/settings.json` is not in the diff at all, so no deny/allow rule changed.
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tag exists in any contract file (grep over the folder returns only review-state files `review-round5.md`/`review-second-round1.md`, which quote the scanner's output; ticketed as #3616).
- Checked and found sound (argued against): argv-only push with a full-sha lease; `ls-remote` exact-name parsing; local checks before network; `GIT_TERMINAL_PROMPT=0` on every child (`TestGitEnv_NeverPromptsOnTheTerminal`); credential leak through `remote-unreadable` prose — I probed `git ls-remote` against a URL carrying `user:token@` and git redacted the userinfo from the error before `gitOutput` could wrap it.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | REAL | tooling signal (pre-existing, outside this diff) | The session-start banner flags this spec for unresolved `[AGENT-DRAFT]` tags although no contract file carries one: the scanner walks every file in the folder, including review-state files; `dotf spec archive`'s own pre-flight skips them, so the archive is unaffected | observed this session (launch banner) + `grep -l` returning only `review-round5.md` / `review-second-round1.md` (and transcripts); first-signature round 5 located the cause in `cli/internal/mem/session_start.go` | UNTESTED — no `session_start` case asserts a review-state file is ignored | code (follow-up, ticketed **#3616**; do not widen this PR) |
| Minor | THEORETICAL | resilience | No timeout on `ls-remote` or `push`: a remote that accepts the connection then stalls holds the agent's turn | code read of `gitOutput`/`Push`; no reproduction | UNTESTED | code — **ticketed #3617**, tracked |
| Minor | THEORETICAL | protection scope | Only `main`, `master` and the remote default are refused; `release/*`, `develop`, `gh-pages` can be force-updated by a caller passing their current sha | code read of `checkDefault`; owner accepted 2026-11-23 with measurements over 26 repos, recorded in `proposal.md` Risks and ADR-066 Consequences | UNTESTED (by design — none is refused) | spec — **owner-accepted**, closed |
| Minor | THEORETICAL | test coverage | The 64-hex (SHA-256) `--expect` path is accepted by `fullSHA` but never exercised — every fixture is SHA-1 (40 hex) | code read of `fullSHA` + `strings.ToLower` handling; no SHA-256 fixture exists | UNTESTED | tests (follow-up; the negative rows for short/symbolic expect are covered) |
| Minor | THEORETICAL | test coverage | At the `dotf` command level, a git push rejection (hook refusal, lost race) reaching `withExitCode(1)` after `Push` runs is not exercised end-to-end; the cmd tests only drive `Prepare` refusals — though it is the same handler line already tested, and `TestPush_PrePushHookRuns` covers the `Push` half at package level | code read of `pr_push_lease.go` RunE | UNTESTED (same code line as tested refusals) | tests (optional follow-up) |
| Question | SPECULATIVE | permission pin | AC4's "no rule matches" rests on `leaseRuleMatches`, an emulation of Claude Code's matcher, not the real matcher | first-signature disposition: command text contains no literal any deny rule names | `TestPushLeaseRules_MatcherMatches` pins the emulation's semantics | none — dispositioned |

No Blocker. No REAL Major. The REAL findings are both pre-existing, outside this diff, and ticketed (#3616, #3617).

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | AC1–AC4 verified green in my own runs; 20-row refusal table incl. real URL rows; three mutants killed exactly as the mutation table claims; no defect found this round. |
| Verification       | A | I reproduced suites, mutants, build/vet and gocyclo; red-green history, 11-row mutation table and CI results are recorded in `verification.md` (CI/full suite not re-run here — UNVERIFIED, Windows-suite cost documented). |
| Scope              | A | Diff is 29 files, every one named by the proposal; no deny/allow rule touched; `AGENTS.md` line outside the generated block; no drive-by edits. |
| Reliability        | B | Refusals precede network where possible, stable codes, prompt disabled, lease closes the TOCTOU window — but no timeout on the network calls (ticketed #3617), so a stalled remote still holds the turn. |
| Maintainability    | A | gocyclo max 8 measured this session, functions small, why-comments on the non-obvious traps (up-to-date lease, split URL), `go vet` clean. |
| Handoff-readiness  | A | ADR-066 with alternatives, lesson 615, AGENTS.md line, every prior finding dispositioned in `verification.md`, follow-ups ticketed. |

Aggregation: all B or above → PASS permitted on the rubric; the severity axis adds no Blocker and no REAL Major.

### Verdict
PASS WITH GAPS

Minors only, every one tracked with a disposition: two ticketed outside this diff (#3616, #3617), one owner-accepted in the contract, two thin UNTESTED edges, one dispositioned Question. The gap the second signature found REAL (split remote) is fixed, and I proved the fix's tests kill its mutant.

### Recommended next steps
Contract set (`proposal.md`, `tasks.md`, `features.json`) is closed — nothing below edits it; disposition these in `verification.md` or a follow-up ticket.
- Disposition the two thin UNTESTED edges (64-hex expect row; cmd-level push-rejection exit) as applied/ticketed/declined in `verification.md`.
- Keep #3616 (banner false positive) and #3617 (network timeout) as the tracked follow-ups; both are outside this diff.
- Signature freshness: this review is fresh at `2cae9c0` with digests matching `review-request.json`. Whether the two relaunched signatures the `risk: high` spec needs are both satisfied by the surviving rounds is the archive gate's call — **UNVERIFIED** from here; run `dotf spec archive` to learn it.
