---
id: "lesson-435-a-claude-call-without-claude-config-dir-writes-another-claude-json"
type: lesson
status: active
title: "A claude call without CLAUDE_CONFIG_DIR writes another .claude.json"
created: "2026-10-15"
owner: manu
tags: [claude-code, setup, mcp, environment, silent-failure]
---

# A claude call without CLAUDE_CONFIG_DIR writes another .claude.json

**Context:** PLAT-001b moved Claude Code's MCP registration into `dotf deploy` (`claude-mcp`) and converge's `agent-extensions`, and left the old loop in `setup-linux.sh` until the pin carried the new path. Before deleting the loop, the from-zero log of run 38671157122 (ubuntu) was read for proof that it had become a no-op.

**Problem:** It was not a no-op. In the same setup run, `dotf deploy` reported `claude-mcp 0 added, 4 already present`, and the shell loop that followed reported `MCP servers: 4 added, 0 already present`. dotf runs `claude` with `CLAUDE_CONFIG_DIR` pinned to the directory its truncation guard watches. The loop inherited setup's environment, where the variable was unset, so Claude Code kept its state in the home-root `.claude.json` instead: a second registration store that dotf never reads, checks or repairs. The loop's own snapshot guard watched `~/.claude/.claude.json`, so it protected the file the call did not write. Each half reported success.

**Solution:** The loop is deleted (setup has no `claude mcp` call left, and a bats guard keeps it that way); `dotf deploy` is the one writer. Any remaining caller of the claude CLI goes through `claudeCLI`, which sets the variable on the child.

**Why:** A tool whose state location depends on an environment variable has as many stores as there are environments that call it. Two writers that agree on the data but not on the location both succeed, and only comparing their reports in the same run shows it. When two code paths hand state to one tool, pin the location on the call, never by inheriting it.
