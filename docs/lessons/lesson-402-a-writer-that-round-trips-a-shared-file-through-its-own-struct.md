---
id: "lesson-402-a-writer-that-round-trips-a-shared-file-through-its-own-struct"
type: lesson
status: active
title: "A writer that round-trips a shared file through its own struct deletes what it does not know"
created: "2026-10-12"
---

# A writer that round-trips a shared file through its own struct deletes what it does not know

## Context
`~/.config/dotfiles/machine.json` holds three sections with three readers. `paths` is read by
the path cascade, `machine` and `pools` by `LoadMachinePolicy`, and `roles` by the converge
steps a machine opts into (#2356). Only `dotf env set` wrote the file.

## The Trap
`SetMachinePath` loaded the file into `struct{ Paths map[string]string }`, set one key and
marshalled that struct back. Any section the struct lacked was gone after the write: the
machine id, the pool deny list, and any role. The policy reader treats an absent id as an
unidentified machine, which denies every pool. Losing the section therefore did not fail.
It changed the dispatch policy, and nothing reported the change. Each reader had its own
struct, which was deliberate (different absence semantics). That design is fine for reads.
For writes it only works while one writer owns the whole file.

## The Solution
The writers share `editMachine`. It decodes the file into `map[string]json.RawMessage`, lets
the caller replace only its own section, and writes the map back atomically. Sections it does
not touch keep their content. A test writes a path into a file that holds all three sections
and reads the identity and the roles back through their own readers.

## Takeaways
- When several readers share a file, a writer must edit at the level of top-level sections.
  It must never decode into one reader's typed view and write that view back.
- The test for a shared-file writer belongs on the readers it does not own. Asserting the
  writer's own field shows that the write worked. It does not show that the write destroyed
  nothing.
