---
spec: "BUG-167-archive-quotes-dead-reviewer"
verdict: "PASS"
reviewed_sha: "a450087fb1c511262836a16b4d8c68ef45c422e8"
reviewer: "nan/glm5.3-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: BUG-167-archive-quotes-dead-reviewer — `git diff 0c6892f7cb520caa0443eb33619109794f9c6be2...HEAD` at a450087f. The range is the whole change: the four spec commits listed by the launcher, plus other work that arrived through merges (`git log --first-parent --no-merges` over the range returns exactly those four). `git diff <base>...HEAD -- cli/` shows only the spec's four code files, so the merged-in work does not touch the code this spec changes — no cross-interaction to judge.

**Sources**: `specs/BUG-167-archive-quotes-dead-reviewer/{proposal,tasks,verification}.md`, `features.json`, `review-request.json`, `cli/internal/spec/review_turncap.go`, `cli/internal/spec/review_turncap_test.go`, `cli/internal/spec/review.go`, `cli/internal/cmd/spec.go`, `review-round1.md` (round 1, FAIL, same reviewer).

### Spec and task alignment

- `tasks.md` claims all four boxes `[x]`; each maps to diff and to named tests confirmed below. No `[x]` without diff evidence.
- AC1 (refusal quotes a dead reviewer and names it) — **verified live this session** with a binary built from HEAD: a temp repo with a sidecar (`reviewer: nan/deepseek-v4-flash`) and a quota-error transcript produced `dotf spec archive` output ending `the review launched with nan/deepseek-v4-flash left no verdict. It wrote, on transcript: {"event":"result",…,"error":"Individual quota reached. Resets in 1h6m26s."}`; with a `.stderr` present the refusal quoted stderr instead (`fatal: bad credential for provider nan`) — stderr-first precedence as specified. Fixture under `mktemp -d`, removed afterwards.
- AC2 (both ends kept, cut on runes, valid UTF-8) — verified by test plus mutation (Finding-free check below); `readEnds` reads a large file by its first and last 4 KiB before the 400-byte cuts, so "keeps its first and last bytes" holds for very large transcripts too (`TestReviewerLastWordsQuotesBothEndsOfALongFile`).
- AC3 (turn-cap precedence) — `withTurnCapHint` returns the turn-cap hint before reaching the quote path, so a capped run never gets both; the three pre-existing turn-cap tests pass unchanged.
- The second-signer not-found branch now routes through `withTurnCapHint` (`review.go`), closing round 1's Finding 2; the proposal's "What" ("The launcher's startup-death message uses the same helper") matches `cmd/spec.go`'s `reviewerLastWords` delegating to `spec.ReviewerLastWords`.
- Out-of-scope declarations (option A stub, `--status`, startup-window widening) are consistent with the diff — none snuck in.
- Round-1 dispositions in `verification.md` cross-checked against the code: findings 1, 2, 4, 5 applied (each to a named test verified below); 3 and 6 declined with written reasons — the declines are defensible (the stale-stderr case surfaces at launch, not at archive; escaping the quote would alter what the reviewer wrote).
- `contract_digests` in `review-request.json` match `dotf spec digest BUG-167-archive-quotes-dead-reviewer` exactly. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain. `reviewed_sha` in the request equals this HEAD.

### Findings

Round 1's two REAL/THEORETICAL Majors are closed and red-green-verified this session; the minors it left are dispositioned. The table lists what remains open after this round.

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | resilience | If `harness/reviewer-pool.json` is unreadable, `withTurnCapHint` returns the bare error before the quote path, so a dead reviewer's output is not quoted even when the transcript exists. Plausible only in a repo without the pool file; in dotfiles the pool is committed. | Code read, `review_turncap.go` ~54 (`if perr != nil { return err }`). Not reproduced. | UNTESTED | code (move the quote above the pool lookup, or fall through on `perr`) |
| Minor | THEORETICAL | correctness | `readEnds` stats then `ReadAt`s without re-stat; a transcript truncated between the two (crash during write) could yield a tail buffer with stale bytes. Append-only transcripts make this unreachable in practice. | Code read. Not reproduced. | UNTESTED | — (surface only; do not gate) |
| Minor | SPECULATIVE | quality | Round-1 Finding 6 (a quoted line resembling the truncation notice) stays declined; the quote reproduces what the reviewer wrote, which is the right call. Recorded here so the disposition survives in the round-2 artifact. | `verification.md` disposition 6. | UNTESTED | — (declined; no action) |

**Round-1 regression checks (red-green, this session, mutations reverted, `git status` clean on `cli/` after each):**

- Byte-cut mutation of `headAndTail` (rune walk removed) → `TestHeadAndTailCutsOnRuneBoundaries` FAILS ("a rune was split"); real code passes. Round-1 Finding 1 closed.
- Wrap removed from `checkSignature`'s `SecondSigner` not-found branch → `TestArchiveGateQuotesADeadSecondSigner` FAILS. Round-1 Finding 2 closed.
- Quote path removed from `withTurnCapHint` → `TestArchiveGateQuotesWhatADeadReviewerSaid` FAILS. AC1's test discriminates.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All three ACs verified live at CLI level and by discriminating unit tests; negative paths (absent file, empty output, invalid UTF-8, long file, second signer, turn-cap precedence) each have a named test. |
| Verification       | A | `verification.md` names tests that now discriminate (mutation-proven), records the red-green checks, and states honestly what was not reproduced; this review re-ran the full suite, lint (darwin + `GOOS=windows`), and a live `dotf spec archive` repro from HEAD. |
| Scope              | A | Diff is exactly the spec's scope; merged-in work does not touch the changed code; out-of-scope items absent. |
| Reliability        | B | Absent/empty/invalid inputs handled and the 4 KiB ends-read bounds memory; the pool-unreadable edge (Minor above) returns the bare refusal. |
| Maintainability    | A | Small focused functions (well under length/complexity limits), why-comments carried over, export justified by the `cmd` caller, table-worthy tests. |
| Handoff-readiness  | A | Round-1 dispositions written with reasons, promotion candidates dispositioned, archive checklist pending as it should be pre-archive; round-1 live-repro gap closed. |

### Verdict
PASS

### Recommended next steps

The verdict is PASS, so the contract set (`proposal.md`, `tasks.md`, `features.json`) is closed — these are for the implementer to disposition in `verification.md` (applied, ticketed, or declined with a reason) or carry into a follow-up ticket; none gates the archive.

- Minor 1 (code): make the last-words quote independent of the pool file's readability, or record the limitation with a follow-up ticket.
- Minor 2: surface-only; no action needed beyond this record.
- `dotf spec archive` is advisable in the current state.
