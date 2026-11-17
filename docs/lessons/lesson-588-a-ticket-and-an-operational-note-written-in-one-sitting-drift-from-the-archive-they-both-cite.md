---
id: "lesson-588-a-ticket-and-an-operational-note-written-in-one-sitting-drift-from-the-archive-they-both-cite"
type: lesson
status: active
title: "A ticket and an operational note written in one sitting drift from the archive they both cite"
created: "2026-11-12"
---

# A ticket and an operational note written in one sitting drift from the archive they both cite

## Context

Spec AI-055 (the pr-agent budget guard) drew two stalls from `agy/gemini-3.8-flash-high` while it was being reviewed. `nan/glm5.3-flash` took the other two rounds. The spec archived that accounting in `specs/archive/AI-055-budget-guard-readable-reserve/verification.md`, and a later session wrote an `operational_note` on the pool entry (#3359) plus a bitacora ticket (#3361) describing the same event.

## Problem

The note and the ticket contradicted each other. The note read "rounds 1 and 3" and called the draws "consecutive", which is what the archive says; the ticket attributed round 1 to `nan/glm5.3-flash`, which the archive does not. A reviewer compared the two, found the conflict, and - because the note was the artifact inside the diff - reported the note as the wrong one.

It was not. The archive was right, the note mirrored it, and the ticket was the divergent copy. Applying the finding as written would have made the durable operational record disagree with the frozen record of the event it describes, and the only way to settle it is to open the archived paragraph and read the round numbers out of it.

Two smaller versions of the same failure sat in the same sentence. The note claimed the stalled runs "drafted nothing" while citing the deletion of the `review-round3.md` they left behind - if they had drafted nothing there would have been no file to delete. And the ticket carried "~760 s" for one `nan` round, a figure that appears in no file in the repository; it had been invented in conversation and would have outlived every session able to check it.

**Why it happens:** the archive is written while the work is happening, by someone reading the artifact. The ticket and the note are written afterwards, from memory, by someone who has already compressed the event into a story. The story is more fluent than the record, so it wins any disagreement on readability and loses on truth - and the next session learns the wrong version from the more readable source.

## Solution

Diff against the record **at write time**, not at review time. Open the archived paragraph and read the values out of it instead of restating them from memory; a reconstruction is only as good as the check that produced it.

When two artifacts describe one event and disagree, the one written from the record is the one to trust, and the other is the bug - regardless of which one is in the diff. A reviewer citing the archive and then arguing against it is the case to verify, not to apply.

Do not introduce a figure that appears in no record at all. The `~760 s` was removed rather than reconciled, because a number nobody can trace is worse in a durable operational note than an admission that only the round completed is known.

The correction landed as #3369, with the ticket body edited to match the archive and the pool note rewritten in both copies.