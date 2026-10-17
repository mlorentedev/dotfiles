---
tags: [spec, verification]
created: "2026-10-17"
---

# Verification - CLI-094-stopped-review-leaves-a-record

> Spec stage: no implementation yet. Only AC6 (the measurement) has evidence; the other criteria are pending and none is claimed passing.

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

- [ ] AC1 stall detector stops a silent runner: pending
- [ ] AC2 a growing or briefly silent transcript is never stopped: pending
- [ ] AC3 prompt asks for a provisional `review.md`: pending
- [ ] AC4 stopped run reports kept or not rewritten, exits non-zero: pending
- [ ] AC5 `dotf spec archive` refuses PENDING by name: pending
- [x] AC6 measurement and N recorded above

## Promotions

None yet.
