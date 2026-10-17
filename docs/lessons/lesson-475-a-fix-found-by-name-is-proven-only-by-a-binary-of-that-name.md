---
id: "lesson-475-a-fix-found-by-name-is-proven-only-by-a-binary-of-that-name"
type: lesson
status: active
title: "A fix found by name is proven only by a binary of that name"
created: "2026-10-17"
---

# A fix found by name is proven only by a binary of that name


**Context:** #2688: `dotf converge` now puts its own directory first on PATH, because its children (`compile-harness.sh`'s `type -P dotf`) look the binary up by name. The live proof on msi was to copy a dev build to `/tmp/dotf-2688` and run it on `PATH=/usr/bin:/bin`.

**Problem:** That proof could not pass. The fix prepends `/tmp`, and `/tmp` holds no file named `dotf`: the children still find nothing. Worse, a stray `/tmp/dotf` left by anyone would have made it pass while proving nothing, and it put a world-writable directory at the head of every child's PATH. The same holds for every renamed build the repo uses to keep off `~/.local/bin/dotf` (`$TMPDIR/dotf-2136`, `dotf-dev`).

**Solution:** The fix only prepends when the running binary is named `dotf` (`dotf.exe`), a test row holds that, and a dev build under test goes into a directory of its own: `/tmp/dotf-2688/dotf`. A counter-control with the same tree minus the fix, in the same layout, failed. That separates the fix from everything else the dev build changes.

**Why:** A test copy of a binary keeps whatever the proof depends on. When the code under test finds itself by name, the name is part of the input, and renaming the copy changes the experiment.
