---
tags: [spec, verification]
created: "2026-10-17"
---

# Verification - CLI-094-stopped-review-leaves-a-record

> Implementation complete (2026-10-17); the independent archive review is pending, so nothing here is a verdict.

## Measurement for the stall timeout (AC6)

Command (run from `~/Projects/worktrees/dotfiles`): a script over every distinct `*/specs/**/review-transcript.jsonl`, reading `durationMs` from the tool-result events. Only 4 distinct transcripts survive (transcripts are gitignored; the rest are byte copies of the archived HARNESS-071 one); 3 carry `durationMs`.

| Transcript | Tool calls | Longest | p99 |
|---|---|---|---|
| OPS-053-self-deploy-trigger | 46 | 96.9s | 1.1s |
| CLI-131-lessons-index-on-read | 42 | 32.5s | 27.1s |
| CLI-119-lane-next | 49 | 12.6s | 12.4s |

- longest legitimate tool call: 96.9s
- stall timeout N: 10 minutes

N is about six times the longest call measured. The sample is thin (3 transcripts, all pi; agy's stream may not carry `durationMs`), so this is the starting value, with `--stall-timeout` as the override. It also stops the wedge of #1975 (about 6 minutes of silence) at minute 10 instead of at the 45-minute cap.

Note on what the transcript records: `durationMs` is per tool call and there are no per-event timestamps beyond the session header, so the stall watcher measures file growth (mtime of the transcript the sink writes), not event gaps. A silent call that lasts L seconds is an L-second gap in that mtime, which is what the 96.9s bounds.

## Acceptance criteria

- [x] AC1 stall detector stops a silent runner: `cd cli && go test ./internal/spec/ ./internal/cmd/ -run 'TestRunWithLimitsStopsARunnerWhoseTranscriptGoesSilent|TestLastToolCall|TestSpecDeadlineStopsAStalledRunAndNamesTheLastCall' -count=1` exits 0. The stop exits 125 (not the cap's 124) and stderr names `bash: find ~/Projects`. Mutations that failed the tests: `stalled()` never true; `ExitStalled` set to 124. Round 1 of the archive review found the sink buffering events until EOF, which left the watched file empty in the real pipeline; `cd cli && go test ./internal/spec/ -run 'TestSinkTranscriptStoresAnEventBeforeTheStreamEnds|TestRunWithLimitsNamesTheWedgedCallThroughARealSink' -count=1` exits 0 after the per-event flush and both failed before it (lesson 482).
- [x] AC2 a growing or briefly silent transcript is never stopped: `cd cli && go test ./internal/spec/ -run 'TestRunWithLimitsNeverStopsARunnerThatKeepsWriting|TestRunWithLimitsToleratesSilenceShorterThanTheTimeout' -count=1` exits 0. Mutation that failed the first: growth no longer counted as progress. The tests use the ratio 97s : 10 min at scale (0.6s silence against a 3s timeout), not the real 97s.
- [x] AC3 prompt asks for a provisional `review.md`: `cd cli && go test ./internal/spec/ -run TestReviewPromptAsksForAProvisionalReviewFirst -count=1` exits 0 for both signers; the test failed before the change (none of `provisional`, `verdict: PENDING`, `before you explore` in the prompt).
- [x] AC4 stopped run reports kept or not rewritten, exits non-zero: `cd cli && go test ./internal/spec/ ./internal/cmd/ -run 'TestStoppedRunRecord|TestSpecDeadlineStopsAStalledRunAndNamesTheLastCall|TestSpecDeadlineReportsAProvisionalReviewKeptAtTheCap' -count=1` exits 0: "review.md was not rewritten by this run", "provisional review.md kept (verdict PENDING)", or the signed verdict, at both the cap (124) and the stall (125).
- [x] AC5 `dotf spec archive` refuses PENDING by name: `cd cli && go test ./internal/spec/ -run 'TestParseReviewRefusesAPendingVerdictByName|TestFindReviewRefusesAPendingReviewFile' -count=1` exits 0; both tests failed before the change (the message said "unrecognized verdict"). Archive reads review.md through `FindReview`, which the second test covers.
- [x] AC6 measurement and N recorded above

## Archive review

- Round 1 (agy/gemini-3.8-flash-high, 2026-10-17): FAIL. Blocker: `SinkTranscript` flushed only at EOF, so the file the detector reads stayed empty while a reviewer ran (fixed: flush per event, with a test through the real sink). Minor, fixed: `StoppedRunRecord` named `review.md` for the second signer's `review-second.md`. Minor, accepted: `LastToolCall` reads pi's event shape only, so an agy stall is stopped but names no call (the proposal now says so; the stall check itself only watches the file). Minor, fixed: acceptance boxes in `proposal.md` ticked.
- The round was a live run of the detector and the write-first prompt on agy. File times: launched 14:51, `review.md` last written 14:58 (about 7 minutes, under both the 10-minute stall timeout and the 45-minute cap). The FAIL itself is kept as `review-round1.md` by the round-2 review branch, so a reader can check it there; this PR does not carry it.

## Promotions

None yet.
