---
id: "lesson-598-a-review-anchor-moves-when-you-push"
type: lesson
status: active
title: "A review anchor moves when you push"
created: "2026-11-13"
---

# A review anchor moves when you push

## Context

`dotf pr triage-queue` keeps a PR in the queue until every REAL finding has a triage row with a
reproduction, and it finds the row by the finding's `path:line`. On #3421 the author wrote the row
for a REAL finding at `verification.md:39`, the line PR-Agent had anchored it to, then pushed two
more commits that changed that file. GitHub marked the inline comment outdated and started
answering `"line": null, "original_line": 39` for it. The queue read the null as line 0, `rowFor`
returns no row for a finding at line 0, and the PR was held with "the triage record has no row for
it". No row the author could write would have matched (#3455).

## Problem

A finding's current line is not a stable key. It is the line as of the latest head, and the review
loop exists to produce new heads: every push after a reviewer speaks can move an anchor out of the
diff. A gate keyed on the current line therefore fails closed on exactly the PRs doing the loop
right, and the failure reads like an author who never triaged.

## Solution

Key the finding by the line the reviewer anchored, `original_line`, which never moves: a row written
when the reviewer spoke matches for the life of the PR, and the queue prints that line once the
current one is gone. The current line is not a second key. The first cut accepted either, and the
review of the fix showed why that fails open: a push that removes lines above finding A moves it
onto the line finding B was anchored to, and B's row then clears A. `Comment.Anchors()` answers the
reviewer's line, with the current line only for a comment GitHub reports no original for, and
three tests hold the shapes: a null-line finding against a row under the original, a shifted
finding whose row matches on the original and not on the current line, and two findings sharing a
current line where one row clears one finding. A file-level comment has neither line and no row
keyed by a line can name it; that case stays open on #3455.

## Related

- #3455 (TOOL-037), the defect; #3421, the PR it held.
- Lesson 268: a refused question and a negative answer print the same thing. This was the same shape, a gate's own defect reported as the author's omission.
