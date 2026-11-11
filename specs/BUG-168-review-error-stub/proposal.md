---
id: "BUG-168-review-error-stub"
type: spec
status: verifying
created: "2026-11-11"
issue: "mlorentedev/dotfiles#1511"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# BUG-168: a review that ends without a verdict leaves review.md with verdict ERROR

## Why

<!-- from issue #1511: dotf spec review exits 0 when the review never ran -->

A detached review that dies after the launcher's startup window (#1511: a spent quota after 5.6s) leaves no `review.md`. An absent file is indistinguishable from a review nobody launched, so the gate is believed passed until `dotf spec archive` refuses much later. #3322 made that refusal quote the reviewer; this makes the failure an artifact on disk the moment the run ends, which is the option the owner chose on #1511 (A).

## What

When the run under `dotf spec deadline` ends and the review file is absent, the launcher writes it with `verdict: ERROR`, the reviewer and reviewed commit from the launch request, how the run ended (exit status, deadline or stall), pi's turn-cap line when it applies, and the reviewer's own last output (stderr first, then the transcript). The file is copied back like any review. `ParseReview` refuses `ERROR` by name, as it refuses `PENDING`, so `dotf spec archive` refuses it and the refusal says to run the review again. A provisional `PENDING` file, or any file the reviewer wrote, is left as it is.

## Out of scope

- `dotf spec review --status <id>` (option B on #1511).
- Making the detached launcher wait for the outcome, or widening its three-second startup window.
- A new `FallbackReasons` classification from the stub's text.

## Risks / open questions

- A reviewer that writes `review.md` and then crashes keeps its own file: the stub is written only when the file is absent, never over one.
- The stub quotes reviewer output into the spec folder; the launcher already prints the same output for startup deaths (#989) and the archive refusal quotes it (#3322), and pool CLIs print provider errors, not credentials.
- A stub rotates into `review-roundN.md` on the next launch like any round; that keeps the failed attempt on record.

## Acceptance criteria

- [x] AC1: A run that exits without writing the review file leaves one with `verdict: ERROR`, the request's reviewer and reviewed SHA, how it ended, and the reviewer's last output; it reaches the caller's spec folder.
- [x] AC2: A run that leaves a review file (signed or `PENDING`) is not overwritten.
- [x] AC3: `ParseReview` refuses `verdict: ERROR` by name, so `spec archive` and the foreground run's outcome both refuse it and say to run the review again.
- [x] AC4: A stopped run's stderr line names the ERROR record rather than "was not rewritten".

## References

- Issue #1511 (options comment issuecomment-6795326258); BUG-167 (archive quotes the dead reviewer, #3322)
