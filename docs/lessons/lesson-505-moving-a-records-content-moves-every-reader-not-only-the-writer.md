---
id: "lesson-505-moving-a-records-content-moves-every-reader-not-only-the-writer"
type: lesson
status: active
title: "Moving a record's content moves every reader, not only the writer"
created: "2026-10-21"
---

# Moving a record's content moves every reader, not only the writer

## Context

MEMORY-008 slice 2 moves a handoff thread's body out of `MEMORY.md`, into
`memory/threads/<key>.md`, and leaves a derived entry in its place. The spec
named one writer (`handoff-write`) and one future reader (`dotf mem resume`).

## Problem

`session-end` reads the thread too. Its fallback journal archives the thread's
block from `MEMORY.md` (`threadHandoffBlock`). Nothing in the spec mentioned
it, and its tests build `MEMORY.md` by hand, so they would have stayed green
while every fallback journal archived a 300-byte summary instead of the
thread. A second reader was out of sight for the same reason: machines on an
older pinned release keep writing the full block inline. Without a rule for
that, the new file would be stale with nothing to say so.

## Solution

Before moving content, find every reader of the old location, not only the
writer: `grep` for the parser's entry points (`handoffSection`,
`threadHandoffBlock`), not for the field names. Then give the new location one
resolver that every reader calls (`threadText`), and make the format say
which kind of record it is: here, the pointer line marks an entry as derived,
and a block without one wins. Older writers then degrade to the old
behaviour, and nothing breaks. Finally, add a test that fails when the reader
gets the summary instead of the thread
(`TestSessionEndArchivesTheStoredThreadNotItsEntry`).
