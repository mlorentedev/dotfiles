---
id: "lesson-603-nothing-reads-it-is-a-claim-about-every-package-that-imports-the-name"
type: lesson
status: active
title: "\"Nothing reads it\" is a claim about every package that imports the name"
created: "2026-11-14"
---

# "Nothing reads it" is a claim about every package that imports the name

## Context

MEMORY-020 (#3487) had to choose where an archived handoff thread's file lives. The cheapest design
left it in `memory/threads/` and only rewrote the archived entry's pointer. It rested on one fact:
nothing enumerates `threads/`, so a finished file there is harmless.

## Problem

That fact came from `grep ReadDir internal/mem`, the package that owns the directory. The reader was
in another package: `internal/doctor/checks_handoff_threads.go` reaches the directory through the
exported `mem.ThreadsDir` and warns on any thread file whose entry has left MEMORY.md. Its comment
even says the archive step moves the file. With the design as first built, every `--finished`
write would have been reported as a lost handoff on each `dotf doctor` run until the writer's
worktree was removed. `--finished` as already merged in #3488 had the same defect. The issue's
premise was also wrong: it said the coded archive shape had never run, and the vault held one file
in that shape. Both claims came from searching the wrong scope.

## Solution

Before resting a design on "nothing reads X", grep the repository for the exported name too, not
only for the operation inside the owning package: `grep -rn 'mem.ThreadsDir\|threadsDir'
cli/internal`. A package exports a constant so that other packages can use it. Check a premise
about data against the data itself (`ls` the vault) before writing it into an issue. The archive
moved the file to `archive/threads/`, where the entry's unchanged pointer also resolves.

## Applies to

- Any change that leaves files behind on the assumption that no check or reader enumerates them.
- Issues and specs that state "X has never happened": check that against the data.
