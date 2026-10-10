---
id: "lesson-383-a-sweep-that-only-adds-links-cannot-converge-them"
type: lesson
status: active
title: "A sweep that only adds links cannot converge them"
created: "2026-10-10"
---

# A sweep that only adds links cannot converge them

## Context
Both setup scripts linked Claude's auto-memory dir for every vault project. Each link went from
`~/.claude/projects/<key>/memory` to `10_projects/<name>/memory`, and the sweep guessed the repo
lived at `~/Projects/<name>`. #1843 B13 asked whether `dotf` should port the sweep, or whether
`dotf doctor --fix` already covered it.

## The Trap
The sweep looked like convergence because it ran on every setup and printed a count. Measured on
msi on 2026-10-10:

- 48 links, 14 of them dangling. All 14 were projects since moved to `90_archive/` or `50_work/`.
  The sweep creates links for the projects it finds and never revisits a link whose project is gone.
- 6 `memory.bak.*` directories. The sweep moved a real memory dir aside, and nothing ever reconciled
  that data back.
- 0 orphan-migration candidates.

`memlink`, the Go primitive the session-start hook and doctor share, had the mirror defect. It
treated any link as healthy, so a dangling one read as `StateLinked`, doctor said PASS, and writes
through it failed.

## The Solution
The sweep is deleted on both OSes. Linking is per project, done when a session opens it (the
session-start hook calls `memlink.Ensure`) and by `dotf doctor --fix` for the current project. Both
know the real working directory, so neither guesses a path. `memlink` now treats a dangling link as
not linked. It replaces the link when a current source resolves, and reports `StateDangling` (a
doctor WARN) when nothing does. A real memory dir is still never moved.

## Takeaways
- **Converging a set means removing what no longer belongs, not only adding what is missing.** An
  additive loop over today's sources leaves yesterday's targets behind, and its success count
  cannot show them.
- **"Is it a link?" is not "does it resolve?"** Check the end state (`Stat` follows the link) and not
  the shape (`Lstat` sees one).
- **Measure the machine before porting a migration.** Here the orphan migration had nothing left to
  migrate, so it was deleted rather than ported.
