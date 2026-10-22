---
id: "ADR-056-a-handoff-thread-is-stored-in-its-own-file"
type: adr
status: accepted
owner: manu
date: "2026-10-21"
supersedes: []
extends: []
issue: mlorentedev/dotfiles#1689
tags: [architecture, decision, memory, handoff]
created: "2026-10-21"
---

# ADR-056: A handoff thread is stored in its own file

## Context

`dotf mem handoff-write` keeps one sub-block per thread under `## Session Handoff` in a project's `MEMORY.md` (HARNESS-088). A session start reads 25,000 bytes of that file and drops the rest. The handoff section is the last one, so the newest thread is the first to go, and the write that caused it reports success (#1477).

Measured 2026-10-20 on the vault: dotfiles' `MEMORY.md` is 23,423 B with eight threads of 1.0 to 2.6 KB each; kubelab is 22,197 B and ts-bridge 22,881 B. MEMORY-008 adds two fields, `Verify` and `Awaiting Manu`. With them, a copy of dotfiles' file is 24,488 B inline, 512 B short of the budget.

## Decision

1. **The thread file is the thread.** `handoff-write` stores the body as `memory/threads/<key>.md`, beside `MEMORY.md`, in the block shape a thread always had (`### thread: <key> [(writer: <agent>)]`, then the body). `ParseThread` reads it unchanged.
2. **The `MEMORY.md` entry is derived from it** and holds only what a session needs before it opens the file: the `Updated` date, the first line of `Next action`, how many items await Manu, the `Journal` line, and a pointer line `[full thread](threads/<file>.md)`. Each quoted field is cut to a fixed number of characters: `Updated` to 24, `Next action` to 160 and `Journal` to 72. A typical entry is under 350 B. With every field at its cap, ASCII text and a key of up to 30 characters, it is at most 450 B. The same copy of dotfiles' file, with every thread stored, is 14,516 B.
3. **The pointer line is structure.** Its presence is what says an entry is derived. A reader that sees one reads the file. `handoff-write` refuses a body that carries one, the same way it refuses a thread marker.
4. **An inline block wins where it has no pointer.** Machines on a release from before the store keep writing full inline blocks, so a block without a pointer is the current thread, and any file under its key is stale. The new writer replaces the thread file on every write, so a stale file only ever sits under an inline block, where no reader looks for it.
5. **The thread file is written first, then `MEMORY.md`, both under the existing handoff lock and through a temp file and rename.** A new thread file takes `MEMORY.md`'s mode.
6. **A key maps to a file the way a branch does.** `/`, `\`, space and `:` become `-`. A name that could leave `threads/` (a leading dot, or a character outside `[A-Za-z0-9._@+-]`) is refused. Two keys that map to one file (`fix/x` and `fix-x`) are told apart by the file's own heading, and the second is refused.
7. **A write that grows `MEMORY.md` past 25,000 B is refused** and names the three least recently updated threads to archive, with how to archive each: a stored thread by its file and its entry, an inline one by its block. A thread's date is its `Updated` line, else the date in its `Journal` path. A thread with neither is listed apart and never suggested, since no date is not evidence of age: the pre-thread block that holds the owner's parked items has none (#1477). A write that shrinks or holds a file already over the budget passes, so such a file can be brought back under.

## Consequences

- Every project's `memory/` gains a `threads/` directory. `vault health` already exempted `memory/` from the orphan check. A thread file has no outgoing links, so `memory/` is now exempt from the dead-end check too, on the same ground: memory files are read by agents, not linked.
- Readers that need the full thread go through one resolver (`threadText` in `cli/internal/mem`). `session-end` archives the thread file, not the entry. `dotf mem resume` (MEMORY-008 slice 3) will use the same resolver.
- An agent reading `MEMORY.md` by hand, such as `/catchup` before slice 5, sees the entry and has to follow the pointer. The pointer is a plain markdown link so that it can.
- `WriteThreadAs` still writes inline blocks. `handoff-write` no longer calls it. Its ownership rule now lives in `threadOwner`, which both writers share.

## Alternatives rejected

- **Inline fields with a per-block byte cap.** It fixes nothing structural, and it put dotfiles' file at 103% of the budget when measured on 2026-09-24 (research note `10_projects/dotfiles/research/2026-09-24-handoff-resume-ritual.md`).
- **Refusing any write over the budget.** Three projects were within 2.9 KB of it on 2026-10-20. An absolute rule would block a handoff that brings a file back under.
