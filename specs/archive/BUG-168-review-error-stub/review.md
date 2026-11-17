---
spec: "BUG-168-review-error-stub"
verdict: "PASS"
reviewed_sha: "d50586cbafc9e21a8706c31ef6de9707d4a0002e"
reviewer: "nan/deepseek-v4-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: BUG-168-review-error-stub — `git diff af73427ae6eaad02a6aa35d22914eeb5249303cc...HEAD` at d50586cb. `git log --first-parent --no-merges` over that base returns exactly the three commits the launcher named (`00baeaa0`, `c36eb7b5`, `d50586cb`), and `git diff <base>...HEAD --stat` shows only `cli/` files for this feature plus the spec folder and the BUG-167 archive, so no other work arrived through a merge and there is no cross-interaction to judge. The base is the `base_sha` the launcher recorded in `review-request.json`, and `dotf spec digest BUG-168-review-error-stub` reproduces the four `contract_digests` byte-for-byte, so the contract I read is the contract the request was taken over.

**Sources**: `specs/BUG-168-review-error-stub/{proposal,tasks,verification}.md`, `features.json`, `review-request.json`; `cli/internal/spec/{review.go,review_error_stub.go,review_request.go,review_turncap.go,stall.go}` and their tests; `cli/internal/cmd/{spec.go,spec_deadline_error_stub_unix_test.go,spec_deadline_unix_test.go,spec_review_worktree.go}`.

### Spec and task alignment

- AC1 (stub on a verdictless run) — implemented by `spec.WriteErrorStub` and wired in `spec deadline` both after a normal `RunWithLimits` return and on its error path, before `finishReviewTree` copies the file back. Verified in practice, not only by reading: `TestDeadlineWritesAnErrorStubWhenTheRunSignedNothing` passes for exit 0, exit 3 and the deadline, and asserts the stub carries `verdict: ERROR`, the request's reviewer and SHA, how the run ended, the reviewer's stderr, and that it reaches the caller's folder through the copy-back; `TestWriteErrorStubRecordsTheRunThatSignedNothing` runs the first and second slots.
- AC2 (never overwrite the reviewer's file) — `os.Lstat` guard in `WriteErrorStub`; `TestWriteErrorStubLeavesAnExistingReviewFile` (PENDING and PASS) and `TestDeadlineKeepsTheReviewersOwnFile` pass. I reproduced the red state: flipping the guard to `err == nil && errors.Is(err, fs.ErrNotExist)` turns both tests red ("the reviewer's file was changed", "want the reviewer's PENDING file untouched"), so the guard is the thing the tests measure.
- AC3 (`ParseReview` refuses ERROR by name) — `VerdictError`, the dedicated branch in `parseReviewAs` before the unrecognized-verdict branch, and `Blocks()` already excludes anything but PASS/PASS-WITH-GAPS. `TestParseReviewRefusesAnErrorVerdictByName` and `TestVerifyReviewProducedReportsAnErrorStub` pass; the latter also proves the `FindReviewIn` path returns a populated verdict beside the parse error, which is what `VerifyReviewProducedIn` branches on. `spec archive` reaches the same refusal through `checkSignature` → `FindReviewIn` err, and no other consumer of `FindReview*` exists (`grep` shows only `signedFail`, `checkSignature`, `VerifyReviewProducedIn`), so no path treats an ERROR file as a verdict.
- AC4 (stopped-run line names the ERROR record) — `StoppedRunRecord`'s new case before the generic `perr == nil` branch; `TestStoppedRunRecordNamesAnErrorStub` and the updated `TestSpecDeadlineStopsAStalledRunAndNamesTheLastCall` pass. Removing the `writeErrorStub` call makes that updated test fail on `review.md records verdict ERROR`, so the assertion is load-bearing rather than satisfied by an older message.
- Every `[x]` in `tasks.md` maps to code and a named test in the diff; the one open closing box (`verification.md` filled in; independent review; archive) is correctly still open, and `verification.md` is accurate about what exists — I re-ran its commands (build, `go vet` on both `GOOS`, `golangci-lint run` and `GOOS=windows golangci-lint run` at 0 issues, `go test ./...` exit 0) and reproduced both mutations it claims.
- The only diff change outside the feature's own files is extracting `runnerOf` out of `withTurnCapHint` so the stub and the archive hint resolve the pool member the same way; it is documented in `verification.md`'s decisions and the full suite stays green.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | docs/help | `spec deadline`'s own `Long` text still says "a stopped run also says whether review.md holds a provisional record or was not rewritten". After this change the command writes and reports an ERROR record, so the sentence describes the old behaviour; AC4 fixed the stderr line and the test, not the help. | `git diff` touches `spec.go`'s `runEnding`/`writeErrorStub` but not the `Long` block at the top of `newSpecDeadlineCmd`. | UNTESTED (help text; no test asserts it) | code (Long string) |
| Minor | THEORETICAL | launcher message | For a death *inside* the 3s startup window (#989), `confirmLaunched` still says "Nothing was reviewed and <review.md> was not written", yet the reviewer's fast death lets `spec deadline` write the ERROR stub and copy it back (deadline is synchronous before its exit, and `finish()` then finds nothing left to do). The caller is then told nothing was written while an ERROR file sits in the folder. | Code read of `confirmLaunched` (message) versus `spec deadline`'s synchronous `writeErrorStub` before `finishReviewTree`; #989 measured 0.54s, inside the window. Not reproduced here. | UNTESTED (`grep confirmLaunched` in `internal/cmd/*_test.go` returns nothing) | code (message) or spec (name the case in *Out of scope*) |
| Minor | THEORETICAL | resilience | `WriteErrorStub` swallows the sidecar error (`req, _, _ := ReadReviewRequestIn(...)`), so a damaged `review-request.json` yields a stub with empty `reviewer:`/`reviewed_sha:` and no diagnosis — the same silent-drop shape `review_request.go` explicitly forbids for that file ("Loud, not skipped … exactly when the file that carries it is damaged"). | Code read; the stub is still a refusal, so the gate is not weakened, only the record loses who was launched. | UNTESTED for the malformed case (`TestWriteErrorStubWithoutOutputOrRequest` covers an *absent* request) | code |
| Minor | THEORETICAL | reporting fidelity | `runEnding` checks `Stop == StopDeadline` first, so a deadline stop that also carried a nonzero status reports only "stopped at the 45m deadline"; AC1's "exit status, deadline or stall" reads as one cause, but the status is available and dropped on that branch. | `runEnding` switch order; the `deadline` subtest of `TestDeadlineWritesAnErrorStubWhenTheRunSignedNothing` asserts only the word "deadline". | `TestDeadlineWritesAnErrorStubWhenTheRunSignedNothing` covers the branch but not the omitted status | code |
| Minor | THEORETICAL | untested branch | The start-failure wiring (`writeErrorStub` on `RunWithLimits`'s error, "the reviewer could not be started") has no named test; no `spec deadline --review` test launches a binary that cannot start. | `RunWithLimits` returns an error only for an empty argv or a failed `Start`, so the branch is reachable but never exercised. | UNTESTED | tests |
| Minor | SPECULATIVE | secrets-in-artifact | The stub copies the reviewer's stderr/transcript ends into a file that is meant to be committed and copied back, so provider output enters git history on the failure path. The proposal names the risk and argues the same bytes already reach the operator's terminal (#989) and the archive refusal (#3322); the quote is bounded (400 bytes per end). | `ReviewerLastWords`/`headAndTail` bounds; proposal "Risks / open questions". | UNTESTED (no guard; by design) | — (surface only; do not gate) |

No Blocker and no Major. The feature's own primary paths are covered by named tests whose red state I reproduced, and the two branches that are not covered (start failure, launcher window message) are the ones this table records.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | Every AC has a named passing test, including the negative paths (PENDING/PASS kept, no request, no output, nonzero exit, deadline, second slot); the ERROR refusal sits before the unrecognized-verdict branch so it is a known verdict, not a malformed one. |
| Verification       | A | `verification.md`'s commands reproduce in this worktree (build, both `vet`s, `go test ./...` exit 0, both `golangci-lint` runs 0 issues) and both of its mutation claims go red when I apply them. |
| Scope              | A | Diff is the feature, its tests, the spec folder and the BUG-167 archive; the one extra (`runnerOf` extraction) is required by the stub and recorded in `verification.md`. |
| Reliability        | B | Write failures, a kept reviewer file and a missing request are handled; the malformed request is silently dropped and the startup-death message can contradict the file this change now leaves. |
| Maintainability    | A | Small functions (`runEnding`, `writeErrorStub`, `slotOfReview`), comments state why, no dead code, no magic strings beyond the existing slot constants, lint clean on both platforms. |
| Handoff-readiness  | A | Proposal/tasks/verification updated in-session, decisions recorded, promotion candidates dispositioned with reasons rather than deferred. |

### Verdict
PASS

Per the skill's rule, the verdict follows `severity × reality`: six Minor findings, none REAL, no Blocker, and no rubric dimension below B, so nothing here can move it below PASS. The minors are tracked, not fixed.

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is closed — this PASS depends on it staying as it is, so apply none of the following there; record the dispositions in `verification.md` or carry them into a follow-up ticket.

- (code) Update `newSpecDeadlineCmd`'s `Long` text to say a run that leaves no review file gets one with `verdict: ERROR`, replacing "or was not rewritten".
- (code, or spec) Either make `confirmLaunched`'s startup-death message stop claiming the file was not written, or name that path in `proposal.md`'s *Out of scope* — which, being a contract edit, belongs to a follow-up spec rather than this archive.
- (tests) Add a `spec deadline --review` case whose command cannot start, asserting the stub's "could not be started" line, and a malformed `review-request.json` case if you decide the swallowed error should be diagnosed.
- (code) Append the exit status to the deadline branch of `runEnding`, and record the disposition for the bounded reviewer-output quote in `verification.md` (kept as designed, per the proposal's risk note).
