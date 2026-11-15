---
id: "MEMORY-021-shared-thread-file-report"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-14"
issue: "mlorentedev/dotfiles#3521"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# MEMORY-021-shared-thread-file-report

> **Naming**: file lives at `<repo>/specs/MEMORY-021-shared-thread-file-report/proposal.md`. `MEMORY-021-shared-thread-file-report` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3521: MEMORY-020 follow-ups: archive function size and the key-collision park report -->

MEMORY-020 (#3487, PR #3518) gave a parked or finished thread one archive shape. Its independent review left two Minors, filed as #3521. First, `fix/x` and `fix-x` name one thread file. A park of one archives its entry and correctly leaves the other's file in place, but nothing says the archived `[full thread]` pointer now leads to no body of its own. Second, `ArchiveParkedThread` (40 lines, about CC 13) and `takeThreads` (about CC 12) exceed AGENTS.md's limits of 40 lines and CC 10. Lint cannot see this, because `internal/mem` is excluded from gocyclo.

## What

- A park whose archived thread's file is held by another key says so: `lane park` and `worktree done` print `archived   thread "<key>": its thread file is another thread's and stays in threads/, so the archived entry's link does not lead to it`. `worktree sweep` prints `  - archived thread: <key> (its thread file is another thread's and stays in threads/)`, and its JSON report gains `shared_threads`.
- `mem.ParkArchive` gains `Shared []string`. `moveThreadFile` returns whether it left a file because another key holds it.
- `ArchiveParkedThread` is split into `parkTarget`, the lock, and `archiveParked`. `takeThreads` moves its key match to `takesThread` and its block scan to `blockEnd`. The behaviour is unchanged.

## Out of scope

- Renaming either thread or rewriting the archived pointer. The collision predates the park, and the file belongs to the other thread.
- Making the collision impossible, for example by changing how `sanitizeThread` maps `/`.
- Any change to what the archive writes.

## Risks / open questions

- `shared_threads` is a new optional key in `worktree sweep --json`. It is `omitempty`, so a report without the case is byte-identical. Resolved: additive only.

## Acceptance criteria

- [ ] AC1: a park of a key whose `threads/` file another key holds archives the entry, leaves the file, records the key in `ParkArchive.Shared`, and `lane park`, `worktree done` and `worktree sweep` (text and `shared_threads` in JSON) each name it.
- [ ] AC2: every function in `cli/internal/mem/thread_archive.go` is at most 40 lines and under CC 10, with every existing archive test unchanged and passing.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `<repo>/docs/adr/adr-XXX.md` (if any)
- Related patterns: `00_meta/patterns/<pattern>.md` (if any)
