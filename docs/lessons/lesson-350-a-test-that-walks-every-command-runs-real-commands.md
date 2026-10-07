---
id: "lesson-350-a-test-that-walks-every-command-runs-real-commands"
type: lesson
status: active
title: "A test that walks every command runs real commands"
created: "2026-10-07"
---

# A test that walks every command runs real commands

## Context
#2093 added a guard that walks the `dotf` command tree and runs every command group twice:
once with an unknown subcommand, once bare. Its first version ran `dotf init nosuch` and
`dotf init` for real, and scaffolded a project, including a `.git` directory, into the test
package's directory inside the checkout.

## The Trap
A tree walk treats every node alike, but the nodes are not alike. `init` has subcommands, so it
looked like a group, yet it also takes a positional `[path]` and does work when run. The guard
was written for groups whose `RunE` only prints help, and the walk handed it one that does not.
`go test` ran in the package directory, so the write landed in the repository. Only an
untracked-files check afterwards showed it.

## The Solution
A guard that executes what a walk finds must do two things:

- **Select by a declared property, not by shape.** Here, a group that declares its own `Args`
  takes positional arguments on purpose and is never run.
- **Run in a sandbox anyway:** `t.Chdir(t.TempDir())` and a scratch `HOME`, so a node that
  does more than expected writes somewhere disposable.

Assert flag-parse failures, which stop before `RunE`, when you can. They never reach the
command's code; #2092's unknown-flag guard is safe for that reason.
