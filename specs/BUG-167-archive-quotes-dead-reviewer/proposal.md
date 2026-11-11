---
id: "BUG-167-archive-quotes-dead-reviewer"
type: spec
status: verifying
created: "2026-11-11"
issue: "mlorentedev/dotfiles#1511"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# BUG-167: archive quotes what a dead reviewer said

## Why

<!-- from issue #1511: dotf spec review exits 0 when the review never ran -->

A detached review that dies after the launcher's three-second startup window (#1511: a spent quota after 5.6s) leaves no `review.md`, and `dotf spec archive` is where anyone first hears of it. Its refusal said only "no review.md, run /adversarial-review", plus a hint when pi's turn cap was the cause, so a review that died reads as one nobody ran.

## What

When the archive gate finds no new verdict (no `review.md`, or one the run did not rewrite) and the cause is not pi's turn cap, the refusal quotes the reviewer's last output (stderr first, then the transcript) and names the reviewer the request sidecar recorded. A long output keeps 400 bytes from each end, cut on rune boundaries: a startup failure names its cause first, a finished run writes its outcome last. The launcher's startup-death message uses the same helper.

## Out of scope

- A `review.md` stub with `verdict: ERROR` written when a run ends without a verdict (option A on #1511): it changes the verdict contract.
- `dotf spec review --status <id>` (option B on #1511): a new command surface.
- Widening the launcher's three-second startup window.

## Risks / open questions

- Reviewer output reaches the terminal, and the agent transcript with it. Resolved: the launcher has printed the same output for startup deaths since #989, and the reviewer CLIs in the pool print provider errors, not credentials.
- A transcript that is absent (another machine, ignored) adds nothing: the refusal stays as it was. Resolved.

## Acceptance criteria

- [ ] AC1: with no verdict and a transcript holding a provider error, the archive refusal contains that error and the reviewer id.
- [ ] AC2: a long output keeps its first and last bytes, and the quote is valid UTF-8.
- [ ] AC3: a turn-capped run still gets the turn-cap hint, and a written verdict's refusal gets neither.

## References

- Issue #1511; #989 (`confirmLaunched`); #2132 (turn-cap hint).
