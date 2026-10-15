---
id: "lesson-438-a-command-typed-into-a-pane-runs-the-panes-world-not-the-callers"
type: lesson
status: active
title: "A command typed into a pane runs the pane's world, not the caller's"
created: "2026-10-14"
owner: manu
tags: [lane, herdr, agents, shell, cli]
---

# A command typed into a pane runs the pane's world, not the caller's

**Context:** `dotf lane next` hands a finished lane to a successor: it makes a worktree through herdr and types a command into the new pane that starts the same kind of agent (CLI-119, #2528). The first manual version, `herdr pane run wG:p1 'claude "$(cat prompt.md)"'`, worked, so the Go version was expected to be a port. Three things the caller took for granted were not there in the pane.

**Problem:** (1) `pi` is a shell function that wraps `dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY`. `lane exec` replaces the process, and an exec never sees functions, so the bare `pi` binary started with no keys. (2) The typed word `dotf` resolved on the pane's PATH to the pinned release, which has no `lane` command: `Error: unknown command "lane" for "dotf"`. (3) `trustedWorkspaces: ["…/worktrees/*/*"]`, which #2504 said keeps agy from asking, did not: agy 1.3.5 stopped at its trust dialog in a fresh worktree of a trusted repo (#2533). None of the three shows in a unit test of the argv, because the argv was right.

**Solution:** The launch table says what each agent needs to start alone (pi goes through `dotf secrets run`, with the keys named). `lane next` types the absolute path of the binary that made the decision, when that path is a word every shell reads alike, so the successor is the same dotf and not the release on PATH. The command in the pane is a plain word list with a file path, never an inline prompt, so zsh, bash, PowerShell and cmd read it the same way. The agy claim was measured instead of trusted and ticketed.

**Why:** A pane is a fresh login shell: it has none of the caller's functions, aliases, exported variables, PATH order or tool versions. Before saying a handed-over command works, run it in a pane that started cold. A claim about another tool's config (agy "trusts" a glob) is a hypothesis until a launch in a new directory shows no prompt. And a test that pins the argv only proves the command line; the place it fails is the environment it runs in.
