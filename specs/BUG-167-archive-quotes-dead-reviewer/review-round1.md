---
spec: "BUG-167-archive-quotes-dead-reviewer"
verdict: "FAIL"
reviewed_sha: "64da07dc17b102e6ddc00162e64b9696ac9fbef7"
reviewer: "nan/glm5.3-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: BUG-167-archive-quotes-dead-reviewer — `git diff 0c6892f7cb520caa0443eb33619109794f9c6be2...HEAD` (the whole change; the range contains only the three spec commits — the `--stat` lists exactly the eight files below, no other work arrived through merges).

**Sources**: `specs/BUG-167-archive-quotes-dead-reviewer/{proposal,tasks,verification}.md`, `features.json`, `cli/internal/spec/review_turncap.go`, `cli/internal/spec/review_turncap_test.go`, `cli/internal/cmd/spec.go`.

### Spec and task alignment

- `tasks.md` claims all four boxes `[x]`; each maps to diff and to named tests confirmed below. No `[x]` without diff evidence was found.
- AC1 (refusal quotes the dead reviewer's error and names it) — **verified live in this session**, at CLI level: a temp repo with a sidecar (`reviewer: nan/deepseek-v4-flash`) and a quota-error transcript produced `dotf spec archive` output ending: `the review launched with nan/deepseek-v4-flash left no verdict. It wrote, on transcript: {"event":"result",…,"error":"Individual quota reached. Resets in 1h6m26s."}`; with a `.stderr` present the refusal instead said `It wrote, on stderr: boom: bad credential` — stderr-first precedence holds as specified.
- AC2 (both ends kept, cut on runes, valid UTF-8) — the code is correct, but see Finding 1: the named test does not prove the rune property.
- The "what" also reads somewhat wider than the code: see Finding 2 (second-signer slot).
- Out-of-scope declarations (option A stub, `--status`, startup-window widening) are consistent with the diff — none of them is snuck in.
- `contract_digests` in `review-request.json` match `dotf spec digest BUG-167-archive-quotes-dead-reviewer` exactly. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | tests | AC2's rune-boundary guarantee has no discriminating test. `TestReviewerLastWordsQuotesTheEnd` also passes with a2fdce83's fix reverted to a plain byte cut (`out[:half]` + `out[len(out)-half:]`), because the fixture's cut offsets (400 and 3026−400) land on ASCII bytes: 400−26=374, 374 mod 10 = 4 → `'e'` of `"usage: é\n"`; tail offset likewise lands on `'u'`. The very regression a2fdce83 exists to fix would reintroduce silently. | Mutation reproduced this session: byte-cut variant written into `review_turncap.go`, `go test -run TestReviewerLastQuotesTheEnd -count=1` → `ok`; reverted, test still `ok`. Fixture offsets computed from the literals. | `TestReviewerLastWordsQuotesTheEnd` — non-discriminating → the property is UNTESTED | tests (craft input whose byte-400 / tail-400 boundary falls inside a multi-byte rune, then red-green it) |
| Major | THEORETICAL | coverage | A dead *second-signer* review left no quote either: `checkSignature`'s not-found branch returns the plain "needs a second signature" message for `SecondSigner` without `withTurnCapHint`, so when `review-second.md` is absent the refusal names no reviewer and quotes nothing — the exact "reads as nobody ran it" confusion the spec describes, for `risk: high` specs. The stale-verdict branch does get the hint, only the absent branch misses it. | Code read, `review.go` ~389–392 (the `slot.Name == SecondSigner.Name` early return). Not reproduced. | UNTESTED | code + tests |
| Minor | THEORETICAL | correctness | A relaunch that dies before tmux starts its command leaves the previous attempt's `…transcript.jsonl.stderr` in place; `ReviewerLastWords` then quotes the old attempt's words as the current run's. `TmuxWrap`'s `2>` truncates only once the tmux command starts. | Code read of `TmuxWrap` (`review_launch.go` ~537); no repro. | UNTESTED | code |
| Minor | THEORETICAL | performance | `ReviewerLastWords` reads the whole transcript with `os.ReadFile` only to keep 800 bytes; transcripts can be very large by design (#995: "correct, just enormous"). The capped short-circuit avoids the read, a killed long run does not. | Code read. | UNTESTED | code |
| Minor | THEORETICAL | correctness | A hard-killed process can leave a truncated final multi-byte rune; rune-boundary cutting preserves validity only when the input itself is valid UTF-8, and an invalid tail propagates into the quote (AC2 says "the quote is valid UTF-8"). | Standalone probe of the verbatim algorithm (mktemp): input ending in a lone `\xc3` → output flagged invalid by `utf8.ValidString`. | UNTESTED | code |
| Minor | SPECULATIVE | quality | Quoted reviewer output can itself contain a line resembling the truncation notice, which could confuse a reader about where the quote ends (probe: a fake `(truncated; full stderr at /fake)` line survives inside the quote). | Probe case 5. | UNTESTED | — (surface only) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All ACs behave as specified at CLI and unit level (verified live), but AC2's rune guarantee rests on an unguarded property (Finding 1) and one no-verdict shape is uncovered (Finding 2). |
| Verification       | C | `verification.md`'s AC2 evidence is a test that a byte-cutting defect passes; one acceptance criterion's named evidence does not discriminate the fix. |
| Scope              | A | Eight files, exactly the spec's scope; the range contains no other work and no creep. |
| Reliability        | B | Absent/empty inputs, empty-output fallbacks and rune boundaries handled; minor stale-file and memory gaps remain. |
| Maintainability    | B | Small focused functions (well under complexity limits), why-comments carried over, export justified by the `cmd` caller. |
| Handoff-readiness  | B | Artifacts complete, honest note that a live repro was not run (this review closed that gap for AC1), promotion dispositions recorded. |

### Verdict
FAIL

### Recommended next steps

The verdict is FAIL, so these are the round's work, followed by a re-review; none asks for a contract-set edit — `proposal.md`, `tasks.md` and `features.json` stay as this review describes them.

- Finding 1 (tests): add a case to `TestReviewerLastWordsQuotesTheEnd` whose cut points straddle multi-byte runes at both ends, verify it fails against a byte-cutting implementation, then passes against the real one. Red-green, not red-once.
- Finding 2 (code + tests): either route the `SecondSigner` not-found branch through `withTurnCapHint` with a named test, or record the limitation as a decision in this spec's `verification.md` plus a follow-up ticket — applied, ticketed, or declined with a reason.
- Findings 3–6: disposition in `verification.md` (applied, ticketed, or declined with a reason); none gates this change on its own.

Evidence carried into this review: `go vet` and `golangci-lint run` clean (darwin and `GOOS=windows`), `go test ./internal/spec ./internal/cmd` green, `dotf spec digest` matching `review-request.json`, live `dotf spec archive` refusals quoted above, mutation red-green probe, and standalone rune-boundary probes under `/tmp/probe-d167` (nothing left changed in the repository outside the spec folder).
