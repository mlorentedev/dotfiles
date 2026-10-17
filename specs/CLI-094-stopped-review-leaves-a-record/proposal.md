---
id: "CLI-094-stopped-review-leaves-a-record"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-10-17"
issue: "mlorentedev/dotfiles#1975"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "Owner granted one override 2026-10-17 for one spec covering #1975-A and #2448-A: both share the launcher's termination path, so one spec is cheaper than two. WIP was 13 of 10; the previous override (CLI-131) was released by its archive. (13 active, limit 10, 2026-10-17)"
---

# CLI-094-stopped-review-leaves-a-record

> **Naming**: file lives at `<repo>/specs/CLI-094-stopped-review-leaves-a-record/proposal.md`. `CLI-094-stopped-review-leaves-a-record` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #1975: CLI-094: a wedged tool call inside dotf spec review silently costs the whole round -->

`dotf spec review` bounds the whole run (45 minutes) and nothing else, and a run the cap stops "writes no verdict, so the whole run is lost". Two measured rounds were lost that way for different reasons: a wedged tool call (a `find ~/Projects` held one call for six minutes, #1975) and a reviewer that was still collecting evidence after 241 shell commands when the cap fired (#2448, PI-002). Both end the same way: 45 minutes of wall clock, a `review.md` byte-identical to the previous round's, and nothing on disk to say why. The two share one fix surface, the launcher's termination path, so this is one spec for both tickets.

## What

- **Wedge detection (#1975, option 2).** While a reviewer runs, the launcher watches its transcript. If the file has not grown for `N` while the process group is alive, the launcher stops the group through the same path the cap uses, with an exit status of its own (not 124) and a stderr line that names the last tool call in the transcript (`tool_execution_update` command), instead of waiting for the cap. `N` is a launcher constant with a `--stall-timeout` override, set from measured transcript timestamps and larger than the longest legitimate single tool call (a full test run).
- **Write-first (#2448, option 2).** The reviewer prompt asks for a provisional `specs/<id>/review.md` in its first moves (frontmatter with `verdict: PENDING`, findings so far) and to refine it. The launcher, when a run is stopped by the cap or by the wedge detector, says which of two things is on disk: "provisional review.md kept (verdict PENDING)" or "review.md was not rewritten by this run". It exits non-zero in both.
- **PENDING is a named, refused verdict.** `dotf spec archive` reads `verdict: PENDING` as an unsigned, provisional record and refuses it with that wording, instead of the generic "unrecognized verdict" error.

## Out of scope

- A per-tool deadline passed to the runner (#1975 option 1): per runner, and the repo already chose a launcher-side bound once (HARNESS-152). Revisit only if a runner proves to need it.
- Steering the running reviewer at the soft mark (#2448 option 1): both runners are one-shot, so an input channel very likely does not exist.
- A second invocation that summarises the transcript into a verdict (#2448 option 3): it would produce a verdict nothing verified by running.
- Killing only the stray child process: finding it is per-OS. The first increment fails loudly and names the call.
- The `adversarial-review` skill text (vault, owner-lands). The write-first instruction is in `ReviewPromptIn`, which reaches both signers.
- The scope line in the prompt that stops the walk of the home directory: already landed in #2702.

## Risks / open questions

- **`N` must exceed the longest legitimate tool call.** A bats or go test run produces no transcript events while it runs, so a `N` that is too short kills a healthy review. Decided: measure `N` from event timestamps in surviving transcripts before choosing it (transcripts are gitignored, so only local ones are data), and record the measurement in `verification.md`. Open until that measurement exists.
- **A provisional `review.md` overwrites the previous round's file.** Same as today; the `review-round<N>.md` convention keeps the earlier rounds.
- **A PENDING file must never satisfy a gate.** Pinned by a test on `dotf spec archive`; it already refuses anything that is not PASS or PASS-WITH-GAPS.
- Shell portability: nothing here is shell; Go in `cli/`. The watcher must build for Windows (`GOOS=windows go vet`).

## Acceptance criteria

- [ ] A fake runner that stays alive and writes nothing to its transcript is stopped after the stall timeout, not the cap; the exit status is not 124 and stderr names the last tool call recorded in the transcript.
- [ ] A fake runner that keeps writing transcript events is never stopped by the stall detector, however long it runs.
- [ ] The reviewer prompt asks, for both signers, for a provisional `review.md` with `verdict: PENDING` before exploring.
- [ ] When a run is stopped (cap or wedge) the launcher prints whether a provisional `review.md` was kept or the file was not rewritten, and exits non-zero in both cases.
- [ ] `dotf spec archive` refuses a `review.md` with `verdict: PENDING` and says it is a provisional, unsigned record.
- [ ] `verification.md` records the measured longest legitimate gap between transcript events and the `N` chosen from it.

## References

- Bitácora board: #1975 (wedge, gates this spec) and #2448 (time budget advisory, covered by the same spec; closes with it).
- Related: HARNESS-152 (`DefaultReviewerTimeout`, `RunWithDeadline`), #2702 (scope line), #1710 (cap applies to pi).
- Advisor pass 2026-10-17: A on both tickets, one spec, comments on the two park posts.
