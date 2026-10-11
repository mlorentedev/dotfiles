---
id: "lesson-375-a-ratio-reads-its-numerator-and-denominator-from-different-sources"
type: lesson
status: active
title: "A ratio reads its numerator and denominator from different sources"
created: "2026-10-09"
---

# A ratio reads its numerator and denominator from different sources

## Context
`dotf vault health` reported `Dead-ends: 1936/2184 (88%)` on every session start (#2197). The
ticket's diagnosis was genre: session journals, archived notes and course notes have no outgoing
links by design, and the owner decided to leave them out.

## The Trap
With those zones excluded the count was still 62%, and part of it was not a genre at all. The
numerator came from `obsidian deadends`, which lists every file in the vault, images and PDFs
included: 469 of the 1936 entries were attachments. The denominator came from a walk of `*.md`
files. Each half was correct for its own source, so neither looked wrong. The orphans count had
the same split: 249 of its 408 were attachments, which turned a 15% PASS into a 39% WARN.

## The Solution
One tally per check reads the CLI's listing and the markdown population through the same filter:
attachments are set apart (and reported as their own line for orphans, where an unlinked image is
worth knowing), and the exempt zones come from one table that also writes the report's
"Not counted" lines. Dead-ends now read 671/1182 (56%), and the FAIL names the folders that hold
them. Golden case `deadends-expected-noise` lists an attachment, so dropping the filter turns it red.

## Takeaways
- **Before tuning a percentage, check that both halves count the same kind of thing.** A ratio
  whose terms come from two tools inherits each tool's idea of a "file".
- **Derive the disclosure from the rule.** A report that says what it left out, in a string written
  next to the rule, drifts the first time the rule changes.
