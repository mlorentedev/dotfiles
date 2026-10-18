---
spec: "CLI-094-stopped-review-leaves-a-record"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "57d7351f9b334be68e8f5a85c1d03043f8733eab"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-10-17"
---
## Adversarial review

**Scope**: CLI-094-stopped-review-leaves-a-record (round 2 — the WHOLE change, base `f13d0b606a6818ec617b2c5deff203c692f090e5` as resolved by the launcher)
**Sources**: specs/CLI-094-stopped-review-leaves-a-record/{proposal,tasks,verification,features}.json, `git diff f13d0b60...HEAD`, review-round1.md, review-request.json

### Spec and task alignment

- The spec's own change is five commits: `9c3fcae7` (#2712, spec open), `6f0f1dfe` (#2719, PENDING refusal), `e2d366a8` (#2722, write-first prompt), `eadb884d` (#2726, stall watcher + stop record), `57d7351f` (#2743, per-event flush — round-1 fix, is HEAD). Per-commit `git show --stat` confirms each touches only the four spec files, `cli/internal/{spec,cmd}` files for this feature, and one lesson (`docs/lessons/lesson-482-a-test-that-pre-writes-an-artifact-never-runs-the-producer-that-must-write-it.md`). No creep within the spec's commits.
- The launcher's base sweeps in ~25 unrelated PRs (#2707–#2751: converge, sshtarget, doctor, prland, mise, …) that landed on main in the same window; this review runs at `origin/main` HEAD. Those carry their own specs (OPS-053, PLAT-*, …) and are outside this spec's change — see finding F4 on the ticked scope checkbox.
- All six acceptance criteria in `proposal.md` are `[x]` and each maps to a `features.json` entry with non-empty evidence. I re-ran all six `verification` commands fresh: exit 0 (AC1–AC6).
- Round-1 FAIL dispositions verified against the code, not the prose:
  - **Blocker (REAL) — sink buffering**: FIXED at `57d7351f`. Fresh: `go test ./internal/spec/ -run 'TestSinkTranscriptStoresAnEventBeforeTheStreamEnds|TestRunWithLimitsNamesTheWedgedCallThroughARealSink' -count=1` passes; a mutation deleting the per-event `w.Flush()` makes BOTH fail (2 failures); a live production-shaped run (`spec deadline 30s --stall-timeout 3s --transcript T -- sh -c 'echo <event>; sleep 60' | dotf spec transcript-sink T`) left the event on disk mid-run, stalled at 3s, and stderr read `last tool call recorded: bash: find ~/Projects - name x`.
  - **Minor (REAL) — hardcoded `review.md`**: FIXED — `StoppedRunRecord` uses `filepath.Base(reviewPath)`; mutation restoring the hardcode fails `TestStoppedRunRecordNamesTheFileItWasAskedAbout` with exactly the round-1 symptom.
  - **Minor (REAL) — proposal checkboxes unticked**: FIXED — all six ticked.
  - **Minor (THEORETICAL) — `LastToolCall` pi-shape only**: accepted and documented in proposal + code comment; carried as F3.
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in any spec file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | stall-detection / false positive | The detector's real clock is "no growth of the **filtered** transcript". A healthy reviewer whose single model message streams longer than N with no tool call emits only `message_update`/`*_delta` frames (dropped by the sink), so nothing reaches disk and the run is killed at minute 10 — the round-loss this spec exists to prevent. The proposal bounds N against tool-call duration (96.9s, n=3 transcripts) only; verification.md concedes there are "no per-event timestamps", so inter-event gaps (the quantity the watcher actually measures) were never measured. This run's own turn durations ran to 110s (`durationMs`) — 5.5× under N, but a thinking-heavy turn could cross it. Escape exists: `--stall-timeout 0`. | code read of `isIncrementalFrame` + `stallWatch`; live: `message_end` in this run's stored transcript carries no top-level timestamp (`{message, type}` only), so the gap cannot be measured post-hoc — exactly verification.md's note | UNTESTED (no test drives an active stream with a >N gap between settled events; `TestRunWithLimitsToleratesSilenceShorterThanTheTimeout` covers <N only) | spec (disposition in verification.md: revisit N against inter-event gaps on the next transcripts, per proposal's own commitment) / follow-up ticket |
| Minor | REAL (observed code path) | detached launch / AC4 discoverability | In the default detached mode `TmuxWrap` runs the deadline with `2> <transcript>.stderr`, so the AC4 stop lines (`[review] stalled…`, `last tool call recorded…`, `provisional review.md kept…`) land only in the side file; a watcher attached to the tmux pane sees the stream freeze with no reason line. Foreground shows them live. Pre-existing routing (the cap message always went there too). | `review_launch.go` TmuxWrap; live repro above printed all three lines to the redirected stderr file, none to the sink's stdout | UNTESTED (no test asserts stop-line routing in tmux mode; `TestSpecDeadlineReportsAProvisionalReviewKeptAtTheCap` covers the deadline's own stderr writer) | code (or accepted with a disposition — archive already refuses PENDING by name regardless) |
| Minor | REAL (accepted, documented) | LastToolCall / pi-shape | For a non-pi (agy) reviewer `LastToolCall` returns `""`: the stall stops the run but names no call, so AC1's "stderr names the last tool call" holds for pi only. Round 1 raised this; proposal amended ("another runner's stream gets no such line") and the code comment states the rule — a guessed wedge is worse than none. | `stall.go` LastToolCall shape check; proposal What section | `TestLastToolCallSaysNothingWhenThereIsNothingToName` (pins the no-guess behavior) | accepted — spec documents it; no code fix required |
| Minor | REAL (observed) | tasks.md claim vs review base | tasks.md ticks "No unrelated changes in the diff (no scope creep)", but the launcher-resolved diff `f13d0b60...HEAD` demonstrably contains ~25 unrelated PRs (see alignment). True per-commit for this spec's five commits; misleading read against the base the reviewer is handed. | `git diff --stat f13d0b60...HEAD` (102 files) vs `git show --stat` of the five spec commits (spec + CLI-094 files + 1 lesson) | n/a (spec prose) | spec — contract set: do NOT edit now; record as a disposition in verification.md (excluded from staleness) |
| Question | SPECULATIVE | exit codes | A runner that exits 125 by itself is indistinguishable from a stall stop (same accepted ambiguity as 124 vs the cap). No observed case; pi exits 0/1/2 in practice. | `deadline.go` ExitStalled; pass-through covered generally by `TestSpecDeadlinePassesTheRunnersStatusThrough` | UNTESTED for the 125 collision specifically | — (surface only; do not gate) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All six ACs verified live (exit 125/124/1, named call, record line, PENDING refusal at archive) and by test; one theoretical false-positive path (F1) on an otherwise sound detector. |
| Verification       | B | Every AC has a named, mutation-killed test (6/6 mutations red-green, incl. the round-1 blocker through the real sink) and all six `features.json` commands re-ran green; but N was derived from tool-call durations while the watcher measures inter-event gaps — a method mismatch the spec half-acknowledges. |
| Scope              | B | The spec's own five commits contain no unrelated changes; the launcher base sweeps in ~25 other PRs, and the ticked "no scope creep" checkbox reads false against that base (F4). |
| Reliability        | B | Error paths handled and tested (missing transcript = silence, unreadable review, negative flags, group kill, signals); stop lines reach only the `.stderr` side file in the default detached mode (F2). |
| Maintainability    | A | `golangci-lint run ./internal/spec/... ./internal/cmd/...`: 0 issues; gofmt clean; small functions with WHY comments; seams keep the tests off real time and real tmux. |
| Handoff-readiness  | A | proposal/tasks/verification updated in-session, round-1 FAIL preserved as review-round1.md, lesson-482 committed, dispositions recorded in verification.md, measurement documented. |

### Verdict

**PASS WITH GAPS**

No Blocker; no REAL Major — the round-1 REAL Blocker is fixed, mutation-killed and live-reproduced. The one open Major (F1) is THEORETICAL and carries the spec's own escape hatch (`--stall-timeout`) plus its own commitment to revisit N, so it is tracked, not blocking. Per severity × reality: a THEORETICAL Major sets PASS WITH GAPS; the rubric (all B or above) does not escalate further.

`dotf spec archive` **is advisable** in this state: the frontmatter above is the format it parses, `reviewed_sha` is the commit examined, `reviewer` matches the pool string exactly, and the contract set (proposal.md / tasks.md / features.json) is untouched — verified `git status` clean for tracked files.

### Recommended next steps

(Routed to `verification.md` dispositions or a follow-up ticket — the contract set is closed under this verdict; none of these asks for a contract edit, and none is headed "before archive".)

1. **F1 (Major, THEORETICAL)** — disposition in verification.md per the proposal's own commitment: when the next real transcripts land, measure the longest gap *between settled events* (not tool-call `durationMs`) and revisit N; consider a follow-up ticket on the bitácora if a >10-minute tool-less stretch is observed. Until then, `--stall-timeout 0` is the documented escape.
2. **F2 (Minor)** — decide: accept the `.stderr`-side-file routing with a line in verification.md (it matches the pre-existing cap routing), or ticket surfacing the stop reason in the pane.
3. **F4 (Minor)** — record as a disposition in verification.md that the scope checkbox holds per-commit, not against the launcher's interleaved base; do not edit tasks.md (contract set).
4. **F3** — no action: accepted and documented in the proposal.
