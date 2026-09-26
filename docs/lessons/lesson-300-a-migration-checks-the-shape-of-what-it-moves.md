---
id: lesson-300
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, migration, memory, handoff, data-loss]
---

# 300 — A migration checks the shape of what it moves, not only where it sits

## What happened

#1703 made `handoff-write` move a handoff written before threads existed into a `legacy-*` thread. It identified that handoff by position: any text between `## Session Handoff` and the first thread. The dotfiles `MEMORY.md` keeps a standing preamble there: the pointers to superseded blocks, the size rule, and a list of live items every thread points at. The first write moved all of it under `legacy-undated`. The live items read as superseded, and "only LIVE items below" pointed at nothing. A peer session caught it before committing.

## The rule

Before moving or rewriting content, recognise it by its own shape, here a `Last task` or `Next action` field. Position alone is not enough, because other content can sit in the same place. Test the migration against a real file that holds content it must leave alone, not only against a fixture made of the content it should move.

Refs: MEMORY-010 (#1725, #1726), #1651, #1703.
