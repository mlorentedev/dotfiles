---
id: "lesson-595-a-binary-that-replaces-itself-must-survive-its-own-previous-copy"
type: lesson
status: active
title: "A binary that replaces itself must survive its own previous copy"
created: "2026-11-12"
---

# A binary that replaces itself must survive its own previous copy

## Context

`dotf update` used to replace `dotf` by running `install.sh` or `pwsh install.ps1`, and only on a checkout
that fast-forwarded. On the owner's Windows box a diverged checkout left the machine on an old release with
nothing but a hint to run the installer by hand (#3433). The fix moved the install into Go, one code path
on every OS, replacing the file the running process was started from (CLI-133).

## Problem

Windows refuses to rename over, or delete, a running `.exe`, but it lets the file be renamed. So the
replacement renames `dotf.exe` to `dotf.exe.old`, renames the new binary in, and deletes `.old`
best-effort. The first draft then removed a stale `.old` before setting the current binary aside, and
returned an error when that removal failed. On Unix it never fails, since unlinking a running binary is
allowed, so every local test passed. On Windows the previous copy is often still running: a
`dotf secrets run -- <agent>` wrapper lives for days. The second update would have failed as
`install-failed` on exactly the OS the change was for. A test that replaces a running copy once does not
see it. The trap is the second replacement while the first set-aside copy still runs.

## Solution

Treat a set-aside copy that cannot be removed as normal, not as an error. When `<dst>.old` cannot be
removed, set the current binary aside under a unique `<dst>.old.<n>`, and after the new file is in place
sweep every `<dst>.old*` best-effort. To make the Windows-only failure testable everywhere, put
`os.Remove` behind a seam that refuses `<dst>.old` the way Windows does, so the test fails on every OS if
the fallback goes. Keep the real-process test too: it is the one the Windows CI leg runs for real.

## Related

- #3433, CLI-133 (`specs/archive/CLI-133-update-self-install/`), BUG-037 (why staging plus rename, not an
  in-place write)
- `cli/internal/tools/install.go` (`replaceFile`, `removeAside`), `cli/internal/tools/place_test.go`
