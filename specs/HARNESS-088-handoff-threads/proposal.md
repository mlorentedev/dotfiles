---
id: "HARNESS-088-handoff-threads"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-27"
issue: "mlorentedev/dotfiles#1278"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-088-handoff-threads

## Why

`MEMORY.md` carries **one** `## Session Handoff` block, and this machine runs
five worktrees. On 2026-08-26/27 three sessions wrote it in sequence and each
**replaced** the previous one's block. None noticed.

The skill already forbids it — `harness/skills/handoff/SKILL.md`, HARNESS-028:

> *"If concurrent writes occurred, merge threads."*

**The rule keeps losing because the failure is invisible to the writer.**
Last-writer-wins produces a well-formed file and a successful edit; nothing looks
wrong until a later session follows a pointer into a block that no longer exists.
That is exactly what happened: session 4's block said *"per the prior handoff
below"* and there was no handoff below.

One shared mutable slot with N concurrent writers is a **data-structure**
problem. Editing it more carefully was never going to fix it.

## What

**One sub-block per thread**, keyed by worktree (`### wt-pi-harness`), written by
`dotf mem handoff-write`. A session replaces only its own; every other is foreign
and untouchable.

That is `MergeHooks` from #1272 one content-type over — find-by-marker, replace
in place, append if absent, never reorder or delete a foreign entry — and the
same posture this repository already takes toward Orca's hooks.

**It shares `extractHandoffBlock`'s section boundary** rather than re-deriving
it. Verified: that parser stops at the next `## `, so `###` sub-blocks fall
inside it and archival keeps working. A second boundary rule would be the fifth
silently-divergent parser found here in a week.

**Journals gain the thread**: `<date>-<project>-<agent>-<thread>.md`, so "my
journal" is derivable from the working directory. `dotf mem thread` prints both.

## Out of scope

- **A lock file, a daemon, per-session directories.** The shared surfaces are a
  short knowable list, and one command owning each write is cheaper and more
  honest than a scheme. The list is written into the skill.
- **Migrating the existing `-2`/`-3` journals.** They are append-only history;
  renaming them would rewrite a record to match a convention that postdates it.

## Risks / open questions

- **A subdirectory must resolve to its worktree.** The first `ThreadKey` read
  only `filepath.Base(cwd)`, so a session in `cli/` — where most work here
  happens — resolved to `main`, and every such session would have shared one key
  and clobbered the others exactly as before. **Resolved**: it walks up. Found by
  running `dotf mem thread` from `cli/`; the tests passed because they all passed
  worktree roots.
- **A blank body must not blank a thread.** Resolved: empty stdin is refused.
- **Open**: whether `dotf doctor` should fail when the section lost a thread
  whose worktree still exists. Mechanically detectable (`git worktree list` vs
  thread keys) and not built here — the writer removes the cause, the check would
  catch a hand-edit that reintroduces it.

## Acceptance criteria

- [x] **AC1** — writing one thread leaves every other byte-identical, proven
      against a fixture with two live threads and against a copy of the real
      `MEMORY.md`.
- [x] **AC2** — writing the same thread twice is idempotent; writing new content
      replaces rather than appends.
- [x] **AC3** — a new thread is appended without reordering existing ones.
- [x] **AC4** — the writer and `extractHandoffBlock` agree on the section
      boundary, asserted rather than assumed.
- [x] **AC5** — a missing `## Session Handoff` section is a loud error, never a
      silent append.
- [x] **AC6** — the thread key resolves from any subdirectory of a worktree.
- [x] **AC7** — a journal filename is derivable from the working directory and
      distinct per worktree.
- [ ] **AC8** — `dotf doctor` reports a handoff section that lost a thread whose
      worktree still exists. **Not built** — see Out of scope reasoning above;
      recorded so it is visible rather than forgotten.

## References

- Bitácora: `mlorentedev/dotfiles#1278`.
- The algorithm this ports: #1272's `MergeHooks`.
- The rule it replaces with a mechanism: HARNESS-028.
- The same "a rule nothing enforces" shape: #1197, #1229, #1238.
