---
id: lesson-095-a-non-runnable-cobra-parent-with-no-subcommands-is
type: lesson
status: active
created: "2026-06-14"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 095: A non-runnable cobra parent with no subcommands is demoted to "Additional help topics"

**Context**: Building `dotf init` (CLI-014) incrementally — Step 1 wired the `init` parent into `root.go` before its `agents`/`github` subcommands or an orchestrator `RunE` existed.

**Problem**: A cobra command that is neither `Runnable()` (no `Run`/`RunE`) nor `HasSubCommands()` renders under **"Additional help topics"** in `dotf --help`, not **"Available Commands"** — and the default help template suppresses the whole `Usage:`/`Flags:` block (`{{if or .Runnable .HasSubCommands}}`). A user scanning `dotf --help` wouldn't see `init` as a real command. A test that asserted `Usage:` was present caught the demotion.

**Solution**: Give the parent `RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }` — the same idiom `root.go` already uses for the root. That makes it `Runnable()`, so cobra lists it under "Available Commands" and renders the Usage block, while bare `dotf init` just prints help until the real orchestrator action lands. (Adding a subcommand also flips `HasSubCommands()` true, but the parent should be first-class from its first commit.)

**Rule**: When scaffolding a cobra subcommand tree incrementally, an under-construction parent namespace must be `Runnable()` or already have a subcommand to be a first-class "Available Command". Default it to `RunE: cmd.Help` (the repo idiom) rather than leaving it bare — a bare parent silently degrades to a help topic with no Usage block.
