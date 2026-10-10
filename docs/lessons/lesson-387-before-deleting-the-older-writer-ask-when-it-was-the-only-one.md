---
id: "lesson-387-before-deleting-the-older-writer-ask-when-it-was-the-only-one"
type: lesson
status: active
title: "Before deleting the older writer, ask when it was the only one"
created: "2026-10-10"
---

# Before deleting the older writer, ask when it was the only one

## Context
`~/.claude/settings.json` had two writers on every setup. CLI-063 (#1339) ported the merge into
`dotf deploy` (`claude-settings`) and left the jq and PowerShell twins in place on purpose. Both ran:
`dotf deploy` first, then `merge_claude_settings` / `Merge-ClaudeSettings` over its output (#2000).
Deleting the twins looks free, since the Go port has tests. It is free only where the twin was never
the sole writer.

## The Trap
The two writers had different preconditions and different policies, and a test of either one says
nothing about the other.

- **Preconditions.** The Go entry declares `requires: claude`; the twin needed only jq. A box where
  `claude` is not on PATH when `dotf deploy` runs got its settings from the twin alone.
- **Policy.** The twin replaced `attribution` whole, so a stale trailer subkey could not survive.
  The Go merge recurses into objects. The existing Go test started from a box with no `attribution`
  at all, so it could not tell the two apart.
- **Tests that only looked alive.** `setup-windows.ps1 registers SessionStart hook` grepped for
  `SessionStart` and was satisfied by a comment inside the merge function, years after hooks moved
  to `dotf harness bind`.

## The Solution
Enumerate the cases where the old writer was the only one, and prove each one closed before
deleting it:

1. **Order and PATH:** both setups install `claude` (into `~/.local/bin`, already on PATH) before
   `dotf deploy` runs. The integration log shows `deployed claude-settings` on a fresh container.
2. **Policy:** a Go case with a box holding a stale `attribution` trailer. The merge overwrites each
   template scalar, so the trailer clears. That test settles it, so no per-key replace policy is
   needed.
3. **Live equivalence:** `dotf deploy --dry-run claude-settings` from the worktree reported
   `in sync` on msi, where the twin had run last: both writers produce the same file.
4. **The fresh-box guard:** `verify-setup.bats` now asserts the template's values on the deployed
   file, so the single writer cannot silently stop applying.

## Takeaways
- **A second writer is deleted by proving it redundant, case by case, not because the port has
  tests.** The port's tests cover the port. The question is where the old writer was the only one.
- **A dry run that reports "in sync" after the old writer ran is the cheapest equivalence proof**
  a live box offers.
- **When a function goes, re-read every test that names its file.** A loose grep survives the
  deletion by matching a comment, and then pins nothing.
