---
spec: "HARNESS-203-lease-push-command"
verdict: "PASS"
reviewed_sha: "43292f739630fd5f3e18d53b52549533d68a2b40"
reviewer: "nan/deepseek-v4-flash"
date: "2026-11-20"
---

## Adversarial review

**Scope**: `HARNESS-203-lease-push-command` — `dotf pr push-lease`, reviewed over the whole change
`git diff 31360ae33a738ab309af04a302cb8c71084d3e4d...HEAD` (11 branch commits, 13 files, 1233 insertions;
`git log --first-parent --no-merges` for that range lists exactly the spec's commits, so no foreign work
merged in between).
**Sources**: `specs/HARNESS-203-lease-push-command/{proposal,tasks,verification,features.json}`;
`cli/internal/pushlease/pushlease.go`, `cli/internal/cmd/pr_push_lease.go`, `cli/internal/claude/push_lease_rules_test.go`;
`docs/adr/adr-066-*.md`, `docs/lessons/lesson-615-*.md`, `AGENTS.md`.

Contract digests match the launcher's record: `dotf spec digest HARNESS-203-lease-push-command` returns
`features.json f231c73e…`, `proposal.md cd9159dc…`, `tasks.md 85ef71c5…`, identical to
`review-request.json.contract_digests`. The contract was not edited after the request.

### Spec and task alignment

- AC1 (lease push moves the remote and prints `<branch> <old>..<new>`; a stale `--expect` refuses with a
  non-zero exit and leaves the remote) — implemented in `Prepare`/`Push`, exercised by
  `TestPush_MovesTheRemoteAfterARebase`, `TestPush_StaleExpectLeavesTheRemote` (2 rows) and
  `TestPrPushLease_PushesAndPrintsTheMove`. Reproduced green in this session.
- AC2 (table of refusals, exit 1, reason on stderr, remote refs unchanged) — 16 rows in
  `TestPrepare_Refuses`, 5 in `TestPrepare_RemoteResolutionOrder`, plus `TestPrPushLease_Refusals`
  (exit 1 for refusals, 2 for usage). Refusal `Code`s are asserted, not prose, as `cli/AGENTS.md`
  requires. Reproduced green.
- AC3 (pre-push hook runs and its refusal stops the push) — `TestPush_PrePushHookRuns` installs a
  `pre-push` that writes a marker and exits 1; the test asserts the marker, the error, the untouched
  remote ref and empty stdout. No `--no-verify` anywhere in `pushArgs`. Reproduced green.
- AC4 (settings pinned, `AGENTS.md` line, ADR with alternatives) — `TestPushLeaseRules_RawForcedPushStaysDenied`,
  `TestPushLeaseRules_NoRuleNamesTheCommand`, `TestPushLeaseRules_MatcherMatches`; `ai/claude/settings.json`
  is untouched by the diff; the `AGENTS.md` line lands at file line ~291, **outside** the
  `BEGIN/END HARNESS GENERATED` markers (208–268), so `dotf harness refresh` will not wipe it; ADR-066
  exists, carries an options section with the three approaches and the measured numbers, and its number
  does not collide (`ls docs/adr | grep ^adr-066` → 1). AC4's last clause ("the repository's full gates
  pass") is the one clause not true in the reviewed state — see finding M3.
- Every `[x]` in `tasks.md` maps to a file in the diff; I found no ticked task without code or test
  behind it. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tag exists in `proposal.md`, `tasks.md`,
  `verification.md` or `features.json` (see M1 for where the session-start banner's count comes from).
- Spec vs code: no mismatch found. Everything the proposal names is implemented as described, including
  the four accepted risks; `--expect` is never derived, no refspec argument exists, no `--mirror`, no
  deletion, no tags.

**What I ran** (fresh, this session, in this worktree):

```
cd cli && go build ./...                                   -> exit 0
cd cli && go vet ./internal/pushlease ./internal/claude ./internal/cmd -> exit 0
cd cli && go test ./internal/pushlease -count=1 -timeout 10m -> ok 52.751s (all named tests + 23 subtests PASS)
cd cli && go test ./internal/claude -count=1 -timeout 5m     -> ok 1.869s
cd cli && go test ./internal/cmd -run TestPrPushLease -count=1 -timeout 5m -> ok 31.398s
committed blobs of all five changed Go files through `gofmt -d`      -> empty (clean)
```

`gofmt -l` over the working tree flags nearly every file, including untouched ones; that is the CRLF
checkout (`core.autocrlf`), exactly as `verification.md` states, and the committed blobs are clean —
verified per file with `git show HEAD:<path> | gofmt -d`.

**Not verified (declared UNVERIFIED, not counted against the change)**: `go test ./...` in full (the
author's 10-minute-timeout story in `internal/worktree` is plausible but only the author ran it);
`golangci-lint` (I have `go vet`, not the pinned linter); the mutation table (I read it and agree it is
credible, but re-running nine mutants does not fit this run); PR #3614's CI rollup (network/`gh`
out of scope for this review); ADR-066's measured #3608 numbers, which quote a branch not in this range.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Minor | REAL | tooling signal (pre-existing, outside this diff) | The session-start tag scanner flags this spec although no contract file carries a tag: `dirHasDraftTag` walks **every** file in the spec folder, while the archive pre-flight `spec.FindUnresolvedTags` skips review-machinery files via `IsReviewState`. The only carrier here is the launcher's `review-transcript.jsonl` (12 × `AGENT-DRAFT`, 7 × `AGENT-SUGGESTION`), because the reviewer read the skill text. Nets a permanent false positive on every spec reviewed with `dotf spec review`, which is how the launch banner for this very run read "2 with unresolved … tags: HARNESS-203". `dotf spec archive` itself is unaffected. | observed this session: banner + `grep -rl` over the spec dir returns only `review-transcript.jsonl`; `cli/internal/mem/session_start.go:241-254` has no `IsReviewState` skip while `cli/internal/spec/archive.go:212-232` does | UNTESTED — `session_start_test.go` covers only the positive tag case; no case asserts a review-state file is ignored | code (follow-up ticket; do **not** widen this PR: the function is untouched by the diff). Not blocking: the false positive disappears once the folder moves under `specs/archive/`, which `specs()` skips |
| Minor | THEORETICAL | test coverage / naming | AC2 requires a refusal for a `--remote` that is "a URL or path". The single row named `--remote given as a URL` supplies `f.remote`, a **filesystem path** (`…/remote.git`). The path spelling is tested; no row supplies an actual URL (`https://…`, `git@host:…`), so AC2's URL half has no named test, and the row's name misstates what it passes. Same refused code either way (`git remote get-url` fails), so the user-visible behaviour is right. | `pushlease_test.go`, row `{"--remote given as a URL", …}` vs the proposal's wording | `TestPrepare_Refuses/--remote_given_as_a_URL` passes, but exercises a path — the URL spelling is UNTESTED | tests (outside the contract set: adding a row does not invalidate this review) |
| Minor | REAL | handoff | `tasks.md` Closing checklist is entirely unticked (`[ ]` × 7: criteria coverage, features.json commands, build/vet/test, golangci-lint, scope, verification.md filled, PR opened) although `verification.md` records the evidence for four of them and the PR exists. Tasks no longer reflects reality, which is the one artifact a later reader trusts for "what is done". | `tasks.md` Closing section vs `verification.md` Evidence/Test status; my own build/vet/test runs | n/a — a documentation state, not a behaviour | spec artifacts, **contract set** (`tasks.md`). Do not tick them now: an edit after this review invalidates it. Disposition in `verification.md` (excluded from the staleness check) and tick at the archive commit |
| Minor | REAL | spec vs gates | AC4 ends "the repository's full gates pass". In the reviewed state `spec-gate` is red ("archive-on-merge violation": the PR closes #3612 while the spec is still active) and `review-attestation` is red (draft PR, no reviewer output). The criterion is therefore satisfiable only by a step that comes *after* this review. Disclosed in `verification.md` and in commit 43292f73, so nothing is hidden — but the criterion as written is not an outcome of the change alone. | `verification.md` Test status; commit 43292f73 body | UNTESTED as a gate (the two gates are red for structural ordering reasons); the CI jobs the change does control are reported green and I could not re-run them | spec artifacts (contract set) for the wording; nothing to fix in code. Structural for any spec archived in its own PR, so I am not escalating it |
| Question / assumption | SPECULATIVE | permission pin | The "no rule matches `dotf pr push-lease`" half of AC4 rests on a hand-rolled emulation, `leaseRuleMatches`, which is validated only by `TestPushLeaseRules_MatcherMatches` — i.e. against itself, never against Claude Code's real matcher. Adversarially: could a *broader* real matcher deny the command? No — the command text contains neither `git push` nor any other literal in the deny list, so no wildcard spelling of those rules can reach it. The conclusion survives even if the emulation is imperfect; only the confidence in it, not the outcome, depends on the emulation. | `ai/claude/settings.json` deny/allow/ask lists read in full against the command string | `TestPushLeaseRules_NoRuleNamesTheCommand` (direction that matters, both `deny` and `allow`/`ask`) | none needed; recorded so a later reader knows the pin is behavioural, not a call into Claude's own matcher |

Deliberate non-findings, after trying to break them:

- **Injection.** Branch names reach argv as single elements and cannot carry `:`; `ls-remote` patterns are
  exact-matched against the returned ref name, so a tail-matching pattern cannot smuggle a different ref;
  every git call is `exec.Command` with `-C` and `gitenv.WithoutRepository`, no shell. A branch or HEAD
  file containing a colon makes git reject the refspec, not run something else.
- **Lease integrity.** The pre-check (`remote sha == --expect`) followed by
  `--force-with-lease=refs/heads/<b>:<expect>` means the only window is a push landing between the read
  and the push, which the lease closes. Row (b) of `TestPush_StaleExpectLeavesTheRemote` is the case that
  would otherwise pass silently (git reports "up to date" without consulting the lease); lesson 615
  records it accurately and its number does not collide with the 605 existing lessons.
- **Protected branches.** `main`/`master` by `EqualFold` *plus* the remote default by the union of
  `ls-remote --symref` and `refs/remotes/<r>/HEAD`, with "neither known" a refusal. I could not find an
  ordering in which a branch that is the default on the remote is pushable: a stale `origin/HEAD` cannot
  let a new default through, because the union refuses on either account.
- **Blocking I/O / credentials.** `cmd.Stdin` is nil, so a credential prompt cannot hang the run; a
  remote that is offline is a refusal, never a push.
- **Test deletions.** None. The diff only adds tests.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B  | Every behaviour AC1–AC3 names is covered by a named test that I re-ran green, and the negative table is 16 rows deep; B rather than A because AC2's URL spelling is untested (name mislabels a path) and AC4's gates clause is not an outcome of the change alone |
| Verification       | A  | Red-first records, a nine-row mutation table with the one survivor honestly marked equivalent, and reproducible commands whose results I reproduced for all three packages; the out-of-repo claims are marked as such rather than asserted |
| Scope              | A  | 13 files, all named by the proposal's *What*; no deny/allow rule changed; the `AGENTS.md` line sits outside the generated block; no drive-by edits |
| Reliability        | A  | Refusals precede any network call where they can, stable codes, no stdin prompt, the TOCTOU window closed by the lease itself, and a failure of the push is always a non-zero exit with git's own output |
| Maintainability    | A  | Small functions, `Refusal` with stable codes, comments that explain the *why* (the up-to-date lease trap, the union of two default-branch sources); committed blobs gofmt-clean and `go vet` clean |
| Handoff-readiness  | B  | ADR + lesson + spec evidence all present and accurate, but the `tasks.md` Closing checklist is unticked and AC4's gate clause is deferred to the archive step |

### Verdict

PASS

No Blocker, no REAL Major, and no dimension graded C or D. The three REAL Minors are documented signal or
handoff defects, each with a disposition below; none of them makes the change's behaviour wrong.

`dotf spec archive` is **advisable** in this state, after the next steps below. Two conditions outside my
reach: the archive pre-flight will need its own run (the archive gate is out of scope here — UNVERIFIED),
and `proposal.md` declares `risk: high`, so this first signature must be followed by a second one from an
Anthropic pool member (`dotf spec review HARNESS-203-lease-push-command --second`); the archive gate
checks both.

### Recommended next steps

Contract set (`proposal.md`, `tasks.md`, `features.json`) is **closed** — any edit invalidates this
review, which is why none of the items below is a contract edit:

- **M2 (tests)** — add one row to `TestPrepare_Refuses` with a real URL spelling (`https://example.invalid/x.git`)
  and rename the existing row to `--remote given as a path`. Test-only; safe to apply before the archive.
- **M3 (gates)** — in `verification.md`, record the disposition: AC4's "full gates pass" is met by the
  archive commit, not by the reviewed state; name the two gates and the commit that turns them green.
  Then tick the `tasks.md` Closing boxes **in the archive commit**, after this review has been consumed,
  accepting that the staleness check will then see a new `tasks.md` — or leave them and record why.
  Do not tick them before archiving while this review is the one that authorises it.
- **M1 (tooling, pre-existing)** — file a ticket: `cli/internal/mem/session_start.go` `dirHasDraftTag`
  must skip `spec.IsReviewState` files the way `spec.FindUnresolvedTags` does, with a regression test that
  a spec folder carrying only a `review-transcript.jsonl` is not flagged. Keep it out of this PR (scope).
- **M4 (assumption)** — no action; noted so the pin's behavioural basis is on the record.
- Adversarial reminder carried from the range's own history: four prior rounds (round1–round4 artifacts in
  the spec folder) produced no verdict, so this is the first signature, not a re-review of a re-review.
  Nothing in the four earlier rounds is contradicted by anything I verified.
