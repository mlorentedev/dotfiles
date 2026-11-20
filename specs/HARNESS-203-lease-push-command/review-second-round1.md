---
spec: "HARNESS-203-lease-push-command"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "22709ec48af6165a79fe1537ce751ce7ee81cba5"
reviewer: "anthropic-review/claude-sonnet-5-5"
date: "2026-11-20"
---
## Adversarial review

**Scope**: HARNESS-203-lease-push-command (#3612), second signature, `risk: high`.
**Sources**: `specs/HARNESS-203-lease-push-command/{proposal,tasks,verification}.md`, `git diff 31360ae33a73...HEAD` (13 spec commits), `cli/internal/pushlease/pushlease.go`, `cli/internal/cmd/pr_push_lease.go`, the tests beside them, `cli/internal/claude/push_lease_rules_test.go`, `ai/claude/settings.json`, ADR-066. I did not read `review.md` or its transcript. `contract_digests` in `review-second-request.json` equal the output of `dotf spec digest HARNESS-203-lease-push-command` (all three digests identical).

### Spec and task alignment
- All `[x]` tasks have diff evidence. The only open box is the PR/review line, which is expected at this stage.
- Each of AC1-AC4 maps to named tests. AC1 `TestPush_MovesTheRemoteAfterARebase` and `TestPush_StaleExpectLeavesTheRemote`. AC2 `TestPrepare_Refuses` and `TestPrepare_RemoteResolutionOrder`. AC3 `TestPush_PrePushHookRuns`. AC4 `TestPushLeaseRules_*`.
- I ran `go test ./internal/pushlease` (ok, 91 s), `go test ./internal/claude` (ok) and `go test ./internal/cmd -run TestPrPushLease` (ok, 40 s).
- The full `./internal/cmd` package timed out at 15 min on this Windows host, in `TestReviewWorktreeDropsACommittedVerdictOutOfRealGit` (`worktree.AddDetached`). That test is unrelated to this diff and is the same slow-git-spawn class `verification.md` documents for `internal/worktree`. I do not count it against the change.
- Spec vs code mismatch: the proposal says the remote is read with `git ls-remote <remote>`, and the default-branch and stale-lease guards depend on that read describing the repository that gets pushed. When `remote.<name>.pushurl` differs from `url`, that no longer holds (F1).
- No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags appear in the contract files. The session banner flags the spec through the transcript file, which is already ticketed as #3616.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | REAL | pushurl / guard | `ls-remote` and `remote get-url` read the remote's fetch `url`, but `git push` goes to `remote.<n>.pushurl`. With a distinct pushurl the default-branch refusal and the stale-lease pre-check describe a different repository than the one written. **Reproduced** in a scratch dir with two bare repos A (url, default `main`) and B (pushurl, HEAD set to `feat/x`). (1) `--expect` = B's current sha (A behind) is refused as "stale lease: origin/feat/x is <A's sha>". (2) With B's `feat/x` equal to A's, `dotf pr push-lease --expect <sha>` force-updated B's default branch `feat/x`, exit 0, instead of `default-branch`. Git's own lease still protects the sha, and the `main`/`master` backstop is name-based, so the damage is bounded. The pushurl pair is a rare config, but the guard's stated invariant ("never the remote's default") does not hold there. | probe output (commands run during this review) | UNTESTED. No fixture sets `remote.<n>.pushurl`. | code: resolve with `git remote get-url --push` and `ls-remote` that URL, or refuse when pushurl is set and differs. Add a row to `TestPrepare_Refuses`. |
| Minor | THEORETICAL | scope of protection | Only `main`, `master` and the remote default are protected. `release/*`, `develop`, a `gh-pages` or other shared long-lived branch can be force-updated by anyone who passes its current sha. ADR-066 and the proposal never say this is out of scope. The raw-git deny rule did cover those, and the command is now the sanctioned path. | code read of `Prepare` and `checkDefault`; no repro against a server-side branch protection | UNTESTED (no row for a non-default long-lived branch, correctly so, since none is refused) | spec/ADR: state it as an accepted risk, or add a configurable protected-pattern list in a follow-up |
| Minor | THEORETICAL | resilience | No timeout or `GIT_TERMINAL_PROMPT=0` on `ls-remote` or `push`. A hung remote or a credential prompt stalls the agent turn. Precedent in `lane/claim.go` was not checked for a timeout. | code read of `gitOutput` / `Push` | UNTESTED | code (context with deadline, or document) |
| Minor | REAL | maintainability | `Prepare` has about 11 decision points (cyclomatic complexity about 11) and spans 40 lines. The repo's lint gate (15, `internal/cmd` only) passes, but the rubric's A bar is 10 and the AGENTS.md bar is "< 10". I counted by hand; gocyclo is not installed here. | `pushlease.go` lines 73-112 | n/a | code: extract the stale-lease comparison into a helper, as `checkDefault` already is |
| Minor | THEORETICAL | spec semantics | The `--expect` protection against the agent is advisory. The agent can read the remote sha right before pushing and pass it (`--expect $(git ls-remote ...)`), which makes the pre-check and lease pass by construction. ADR-066 accepts this ("overwrites those commits"). The accepted risk is documented, so I record it without raising its severity. | ADR-066 Consequences; code read | n/a | none; the residual risk is owned |
| Question | n/a | verification | `verification.md` says the command was never run against GitHub. A real-remote smoke test would exercise `ls-remote --symref` output on a hosted server and a pre-push hook that reads stdin. Local bare repos cover git semantics, not the hosting layer. | `verification.md` "Manual smoke test: none" | UNTESTED | verification.md: record one run, or decline with a reason |

Checked and found sound (argued against, not praised):
- Lease and argv construction: `--force-with-lease=refs/heads/<b>:<sha>` with a sha-based refspec. No shell, and an `--expect` that is not full hex is refused.
- The `ls-remote` exact-name matching is correct, so tail-matching `refs/*/HEAD` cannot spoof the default.
- Option-like or URL `--remote` values are refused. The `--no-verify` mutant was reported killed, and the hook test passed in my run.
- Deny globs in `ai/claude/settings.json` are unchanged and the allow list holds no `dotf pr` rule. I re-read the file and the matcher test passes.
- AC2's remote-ref-unchanged assertion is real (it compares the bare repo's refs).

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | AC1-AC4 hold in tests; the pushurl divergence (F1) is a reproduced edge case. |
| Verification       | B | Named tests, red-green history and a mutation table. No pushurl or hosted-remote coverage. |
| Scope              | A | The diff matches the proposal; no deny rule changed and no unrelated files touched. |
| Reliability        | B | Refuses on every uncertain read; no timeout on the network calls. |
| Maintainability    | B | `Prepare` is CC about 11, above the A bar of 10 but below the lint gate of 15; the code is clearly commented. |
| Handoff-readiness  | A | ADR-066, lesson 615, AGENTS.md line, verification dispositions and ticketed M1. |

### Verdict
PASS WITH GAPS

No Blocker. No REAL Major. The one REAL finding (F1) is Minor: the pushurl-vs-url mismatch is reproduced, but it needs unusual config and git's own lease plus the name-based `main`/`master` backstop bound the damage. All rubric grades are B or A, so the aggregation rule alone would allow PASS. I record the gaps because F1 contradicts the spec's stated invariant under that config, and it is UNTESTED.

### Recommended next steps
Contract set (`proposal.md`, `tasks.md`, `features.json`) is closed. Nothing below edits it.
- Disposition each finding in `verification.md` (applied, ticketed, or declined with a reason). Ticket F1 with a root cause: reading `url` instead of the push URL.
- Optionally apply the code and test fix for F1 before merge (outside the contract set), using `git remote get-url --push`, or file it.
- Record in ADR-066 or `verification.md` that long-lived branches other than the default are not protected (F2), or file a follow-up.
- Record F3 (timeouts), or decline it with a reason.
- Add one real-remote smoke run to `verification.md`, or state that it was declined and why.
- `dotf spec archive` is advisable once these are dispositioned in `verification.md`, together with the first signature, since this is the second signature the `risk: high` spec requires.
