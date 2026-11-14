---
id: "MEMORY-019-handoff-archive-finished"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-13"
issue: "mlorentedev/dotfiles#3393"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# MEMORY-019: Handoff archive finished

## Why

<!-- from issue #3393: handoff-write: archive a finished thread when its writer writes it -->

A finished handoff thread stays in MEMORY.md until someone archives it by hand. That happened on 2026-11-06 (six threads) and on 2026-11-12 (ten threads, vault `10f29c46`), each time with the file already past the 200 lines a session start loads, so the newest threads were the ones no session saw. The writer knows when its line of work is finished; nothing lets it say so.

## What

`dotf mem handoff-write --finished` writes the thread as today and then archives it in the same locked write:

- The thread file `threads/<key>.md` is written as on any write and stays where it is.
- The block this write would have placed in MEMORY.md (its `### thread:` heading and the entry derived from the new body: Updated, Next action, Awaiting Manu, Journal, `[full thread](...)` pointer) is appended byte-identical to `memory/archive/superseded-handoffs-<date>.md`. It is the new entry, never the old span re-read, so a thread an older release wrote inline archives in the same shape. A new file opens with `# Superseded handoffs <date>`, the shape of the 11-02, 11-05 and 11-12 files written by hand. Entries are separated by one blank line.
- The thread's entry is removed from MEMORY.md, and stderr says `archived   thread "<key>" to archive/superseded-handoffs-<date>.md`.

The marker is the flag, never the body: a Next action that reads "None" is not taken to mean finished. `--finished` is refused, with nothing written, when the body's Awaiting Manu has an item, because archiving it removes the owner's open decisions from the one surface a session start loads.

## Out of scope

- `dotf mem prune`, the explicit sweep the issue names: it does not exist, and archiving threads other sessions wrote is the move this spec keeps away from. Filed as a follow-up if the flag proves insufficient.
- The hand-written `**Archived <date> (...)**` lines in MEMORY.md's live-items section, and any skill text (`harness/**`, owner-only). The handoff skill learns `--finished` in a follow-up ticket, after a release carries the flag.
- Inferring "finished" from the text of the Next action.
- `mem.ArchiveAndPruneThread`, the lane-park archive (HARNESS-186), which moves the thread file to `archive/<key>.md` instead. Converging the two shapes, and its discarded errors, is #3487. This spec reuses its MEMORY.md prune with an exact-key mode and adds no third shape.

## Risks / open questions

- A crash mid-write. Order is archive append, then thread file, then MEMORY.md, under the existing lock: an interrupted write leaves the entry in MEMORY.md (and at worst also in the archive), never in neither place.
- Idempotence. A `--finished` write for a key with no entry in MEMORY.md and an archive file that already holds that exact entry appends nothing and reports `unchanged`.
- The 11-12 hand archive moved two threads with `Awaiting Manu: 1`. The refusal is loud for that reason: a writer that wants to archive one first answers or moves the decision.
- Ownership after the archive. With the entry gone, the next write to that key is a new thread and is stamped by whoever writes it. `dotf mem resume` finds a thread through its entry (`ThreadText`), so the thread file left in `threads/` never resumes as live.
- Forks. Agent B writing `--finished` under agent A's key forks to `<key>+B` as today; only the fork is archived, and A's thread stays.

## Acceptance criteria

- [ ] AC1: `handoff-write --finished` moves the thread's entry out of MEMORY.md into `archive/superseded-handoffs-<date>.md` byte-identical, keeps every other thread byte-identical, writes the thread file, and reports the move on stderr.
- [ ] AC2: `--finished` with an Awaiting Manu item is refused and writes nothing; a thread without the flag whose Next action reads "None until X, then Y" stays in MEMORY.md.
- [ ] AC3: another agent's `--finished` write under the same key archives only its fork, and the first agent's thread stays in MEMORY.md.
- [ ] AC4: a repeated `--finished` write of an archived thread appends nothing; `--dry-run --finished` writes nothing and names the archive file on stderr.

## References

- Bitácora board: mlorentedev/dotfiles#3393 (split from #3227)
- Related ADR: `docs/adr/adr-056-a-handoff-thread-is-stored-in-its-own-file.md`
- Related specs: MEMORY-008 (per-thread store), MEMORY-009 (writer stamps)
