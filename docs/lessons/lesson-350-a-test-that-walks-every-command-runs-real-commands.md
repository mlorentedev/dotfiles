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

## Addendum: a scratch HOME does not sandbox a variable that is already set

The same first run also wrote outside the checkout: `10_projects/dotf/` and
`10_projects/nosuch/` appeared in the real vault and stayed there until the #2121 backfill.
`env.ResolvePath` returns a set variable before it ever looks at HOME, so the developer
shell's `VAULT_PATH` sent the scaffold to the real vault. That happened even inside the
scratch HOME. The untracked-files check could not see it, because the vault is another
repository.

The walk now points every variable `env-contract.json` declares, plus the XDG roots, at
scratch. The test reads those names from the contract, so a path variable added later is
covered too. A sandbox defined as "a scratch HOME" covers only the defaults derived from
HOME, not the overrides a real shell carries.
