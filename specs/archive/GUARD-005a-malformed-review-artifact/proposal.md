---
id: "GUARD-005a-malformed-review-artifact"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-29"
issue: "mlorentedev/dotfiles#1157"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# GUARD-005a: Reject malformed review artifacts at launch completion

## Why

<!-- from issue #1157: GUARD: dotf spec review exits OK when the run produces no verdict, leaving the previous review.md in place -->

The original GUARD-005 provenance check rejects a missing or byte-identical
`review.md`, but a reviewer can mutate the file without producing valid
frontmatter. WIN-014 reproduced this on Windows: the foreground command exited
successfully after writing only `reviewer` and `reviewed_sha`, while the body
said FAIL and the required machine-readable `verdict` was absent.

## What

After a foreground review exits, `dotf spec review` parses the newly written
`review.md` using the same parser as the archive gate. A missing or invalid
machine-readable verdict makes the command fail immediately and names both the
malformed artifact and the transcript; a valid fresh verdict remains accepted.

## Out of scope

- Changing detached review semantics; the launcher cannot observe a detached
  run after returning.
- Changing the accepted verdict vocabulary or review frontmatter schema.
- Repairing malformed reviewer output automatically.

## Risks / open questions

- Validation must happen after the digest check so an unchanged stale review
  still reports the more precise "reviewer wrote no verdict" cause.
- The foreground guard must reuse `ParseReview`; maintaining a second schema
  would let launch-time and archive-time validation drift.

## Acceptance criteria

- [x] A foreground run that changes `review.md` but omits a valid `verdict:`
  returns an error that names `review.md` and the review transcript.
- [x] A fresh, parseable review artifact still passes launch-time verification.
- [x] Existing missing-file and unchanged-file failure messages retain their
  current precedence.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Prior spec: `specs/archive/GUARD-005-review-verdict-provenance/`
- Related lesson: `docs/lessons/lesson-215-a-parser-for-one-runner-reads-the-other-runners-re.md`

<!-- archived 2026-09-29 — PR: https://github.com/mlorentedev/dotfiles/pull/1828 -->
