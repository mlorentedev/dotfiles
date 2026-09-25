---
id: lesson-297
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, go, filesystem, permissions, atomic-write]
---

# 297 — A temp-file rename narrows the mode of the file it replaces

## What happened

`dotf mem handoff-write` replaces `MEMORY.md` atomically: it writes to a temp file in the same directory and renames it over the original. `os.CreateTemp` creates the temp file with mode 0600, and the rename carries that mode onto the replaced file. Nine of the vault's 21 `MEMORY.md` files were 0600, exactly the ones this command had written. The rest kept the umask's 0664.

## The rule

An atomic replace of an existing file copies the old file's permission bits onto the temp file before the rename. A new file can start private. `harness bind` already did this for settings files; `handoff-write` now does it too, and `TestMemHandoffWriteKeepsTheFileMode` pins it. Look for the same shape anywhere a write goes through `CreateTemp` and a rename.

Refs: MEMORY-008 slice 1 (#1716).
