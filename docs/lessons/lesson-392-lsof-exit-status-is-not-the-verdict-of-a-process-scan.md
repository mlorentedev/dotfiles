---
id: "lesson-392-lsof-exit-status-is-not-the-verdict-of-a-process-scan"
type: lesson
status: active
title: "lsof's exit status is not the verdict of a process scan"
created: "2026-10-10"
---

# lsof's exit status is not the verdict of a process scan

## Context
`dotf worktree sweep` deletes a worktree only when Gate f finds no process working inside it, and
`dotf worktree done` refuses to remove the worktree its own session runs in. Both read process
working directories from `/proc`, so on macOS the sweep never reaped anything and `done` had no
ancestor guard (#2013 F-022). Darwin has no `/proc`, and libproc needs cgo, so the darwin port
asks `ps -axo pid=,ppid=,comm=` and `lsof -d cwd -F pn`, once each (about 0.1s for the machine).

## The Trap
- `lsof` exits 1 when any process could not be listed, and an ordinary user's whole-machine scan
  always hits one (581 of 886 processes readable, measured on this Mac). It still prints the rest.
  Failing on the exit status makes the gate refuse on every run; trusting it would be the opposite
  mistake, as a missing `lsof` also exits non-zero with an empty listing.
- `lsof` reports the physical path. `$TMPDIR` is `/var/folders/…`, a symlink to
  `/private/var/folders/…`, so an unresolved target never compares equal: the gate would answer
  "nobody is inside" while a shell sits there.

## The Solution
- The verdict is the listing, not the status: an empty listing from either tool is a failed read.
  Gate f answers `Inside` on it (the caller deletes on a false); `done` answers "not inside", so a
  broken table never makes worktree removal impossible.
- The target goes through `filepath.EvalSymlinks` before the comparison.
- One process table feeds both gates, and the caller-walk tests that already ran on Linux now run on
  darwin unchanged.

## Takeaways
- Before treating a tool's exit status as the answer, measure what it returns on the partial
  result it will always produce in practice.
- A scan's failure must not look like an empty result. Decide which way each caller fails and test
  that path: here, `PATH` holding `ps` alone.
