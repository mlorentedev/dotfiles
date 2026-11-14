---
id: "MEMORY-020-one-archive-shape"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-14"
issue: "mlorentedev/dotfiles#3487"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# MEMORY-020-one-archive-shape

> **Naming**: file lives at `<repo>/specs/MEMORY-020-one-archive-shape/proposal.md`. `MEMORY-020-one-archive-shape` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3487: mem: two archive shapes for a handoff thread, and the coded one discards its errors -->

A finished handoff thread leaves MEMORY.md in three different shapes today. `handoff-write --finished` (MEMORY-019) and every hand archive append the entry to `archive/superseded-handoffs-<date>.md`. A lane park (HARNESS-186) moves the thread file to `archive/<key>.md` and drops the entry, and its two callers discard its error, so a park that fails to archive says nothing. In both shapes the archived `[full thread](threads/<key>.md)` pointer is relative to MEMORY.md, so read from `archive/` it names a file that does not exist. A reader of the archive cannot follow a finished thread back to its record, and a park that silently leaves a stale entry eats the session-start budget MEMORY.md exists to protect.

## What

- One function in `internal/mem` takes a thread's entry out of MEMORY.md and appends it, byte-identical, to `archive/superseded-handoffs-<date>.md`, and moves the thread file from `threads/<key>.md` to `archive/threads/<key>.md`. Both `handoff-write --finished` and a lane park (`dotf lane park`, `dotf worktree done`, `dotf worktree sweep`, `dotf lane apply`) go through it.
- With the file under `archive/threads/`, the entry's own `[full thread](threads/<key>.md)` pointer resolves from the dated archive, so nothing rewrites it.
- A finished thread no longer leaves a file in `threads/` without its entry: `dotf doctor` reports exactly that as a lost handoff, and `--finished` as merged in #3488 tripped it for as long as the writer's worktree lived.
- A lane park archives the lane's thread and its `<key>+<agent>` forks: a thread is a branch, and a parked branch's forks describe the same finished work. A block whose thread still has Awaiting Manu items is never archived, by either path. It stays in MEMORY.md and `threads/`, and the park names it.
- A park whose archive fails still parks, then prints the failure in ERROR / WHY / FIX form and exits non-zero.
- The budget refusal and the doctor warning name the same shape.

## Out of scope

- Moving existing vault data. The thread files the existing dated archives point at, and the one `archive/feat-3307-lease-in-git-dir.md` file, move by a vault commit after this merges.
- A verb that archives an arbitrary thread by key: the budget refusal names the hand step for a thread the caller does not own.
- Changing what `dotf mem resume` reads.

## Risks / open questions

- Resolved: what enumerates `threads/`. `internal/doctor/checks_handoff_threads.go` does, per live worktree, and warns on a file with no entry; its contract already said "the archive step moves the file". That decides the file moves, and to `archive/threads/` rather than `archive/<key>.md`, which shares a directory with archived auto-memory files (`adr-023-*`, `feedback_*`).
- Resolved: whether park should take another agent's fork. Yes, except a fork with Awaiting Manu items: the same rule `--finished` applies to its own thread (MEMORY-019).
- A key reused after an earlier archive replaces the archived file of the same name (`os.Rename` replaces on every platform). The dated entries keep both records.
- Write order on a park: dated archive, then MEMORY.md, then the file moves. Interrupted, it leaves an entry in both places, or a file in `threads/` that doctor reports. It never leaves an entry in neither.

## Acceptance criteria

- [ ] AC1: One archive shape, written by one function. `--finished` and a lane park both append the entry to `archive/superseded-handoffs-<date>.md`, move the thread file to `archive/threads/<key>.md`, and write no `archive/<key>.md`.
- [ ] AC2: The archived entry is byte-identical to the one MEMORY.md held, and its pointer resolves from the archive file.
- [ ] AC3: Parking a lane archives its forks, and keeps in MEMORY.md any block (the key or a fork) whose thread has Awaiting Manu items, naming it.
- [ ] AC4: A failed archive at park time is reported in ERROR / WHY / FIX form with a non-zero exit, and the lane is still parked.

## References

- Bitácora board: mlorentedev/dotfiles#3487
- MEMORY-019 (`specs/archive/MEMORY-019-handoff-archive-finished/`): the dated shape and the Awaiting Manu refusal
- HARNESS-186 (#3259): the lane park that archives
