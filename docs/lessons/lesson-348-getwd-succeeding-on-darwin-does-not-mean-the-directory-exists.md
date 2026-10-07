---
id: "lesson-348-getwd-succeeding-on-darwin-does-not-mean-the-directory-exists"
type: lesson
status: active
title: "Getwd succeeding on darwin does not mean the directory exists"
created: "2026-10-07"
---

# Getwd succeeding on darwin does not mean the directory exists

## Context
`TestThreadKeyForCwdFailsWhenTheWorkingDirectoryIsGone` passed on Linux CI and failed on a
macOS laptop: a process whose working directory had been removed still got the thread key
`main` (#2085).

## The Trap
`os.Getwd` is not an existence check. Measured on macOS arm64 (Go, 2026-10-07), after
`Chdir(d); Remove(d)`:

- `os.Getwd()` returns the old path with no error. `$PWD` is stale, so Go falls back to the
  kernel, and darwin answers with the last path it knew for the directory.
- `os.Stat(".")` succeeds, since the process still holds the removed inode.
- `os.Stat(<the path Getwd returned>)` fails with ENOENT.

Linux `getcwd(2)` fails with ENOENT for a removed working directory, so the same code looks
right there. A test that only runs on Linux CI never sees the difference.

## The Solution
Stat the path `Getwd` returned, never `.`. A stat of the path asks whether the directory is
still reachable by name, which is what a thread key derived from it needs, and it behaves the
same on every OS. `ThreadKeyForCwd` does this, so no `runtime.GOOS` branch exists.
