---
id: "lesson-384-a-sandbox-home-does-not-isolate-claude-code"
type: lesson
status: active
title: "A sandbox HOME does not isolate Claude Code"
created: "2026-10-10"
---

# A sandbox HOME does not isolate Claude Code

## Context
Triaging #2259 raised a question: does `claude plugin marketplace remove` also uninstall the
marketplace's plugins? To answer it with an experiment rather than a guess, I set `HOME` to a
scratch directory, added a local marketplace, installed a dummy plugin from it, and removed the
marketplace.

## The Trap
On this machine `CLAUDE_CONFIG_DIR=~/.claude`. Claude Code reads its config from that directory, and
the variable overrides `HOME`. The "sandboxed" commands therefore registered, installed and removed
the test marketplace in the real config. The tell was `claude plugin list` in the scratch HOME
listing the real plugins. Afterwards I checked the real config: marketplace list, a grep for the
test name, `.claude.json` size and MCP servers against the floor. The only residue was a cache
directory. Claude had already marked it `.orphaned_at`, and I deleted it.

## The Solution
To isolate Claude Code, set `CLAUDE_CONFIG_DIR` to the scratch directory as well as `HOME`, and then
confirm the isolation: `claude plugin list` must show nothing you did not install there. The
experiment's answer stands: removing a marketplace uninstalls its plugins.

## Takeaways
- **Isolation is a property you verify, not one you set.** Before a mutating experiment, run the
  read command once and check that it sees only the sandbox.
- **Look for the tool's own config variable before trusting `HOME`.** Claude Code, git
  (`GIT_CONFIG_GLOBAL`), XDG tools and gh (`GH_CONFIG_DIR`) all honour one that wins over `HOME`.
