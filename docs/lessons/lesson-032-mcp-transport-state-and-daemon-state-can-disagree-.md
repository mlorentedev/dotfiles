---
id: lesson-032-mcp-transport-state-and-daemon-state-can-disagree-
type: lesson
status: active
created: "2026-05-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 032: MCP transport state and daemon state can disagree per-conversation

**Context:** Mid-session on a fresh Windows 11 laptop, after rejecting the very first `mcp__hive__session_briefing` call in the permission prompt. Every subsequent Hive tool call returned `MCP error -32000: Connection closed`, then `No such tool available`. Spent a few minutes assuming the Hive server had crashed.

**Problem:** `claude mcp list` in a separate terminal reported `hive: uvx hive-vault - ✓ Connected`. The daemon process was healthy; only the current conversation's handle to it was dead. There is no in-session command to re-attach. Filed as [mlorentedev/hive#75](https://github.com/mlorentedev/hive/issues/75); see also [`troubleshooting/hive-mcp-rejection-disconnect.md`](troubleshooting/hive-mcp-rejection-disconnect.md).

**Solution:** Two layers. Operationally: always accept the first MCP tool call in a fresh conversation; you can deny later ones safely. If the transport is already poisoned, restart the conversation — that is the only recovery. Diagnostically: when an MCP-backed tool stops working mid-session, the very first signal to capture is `claude mcp list` in a separate terminal. If the daemon is `✓ Connected` but the conversation cannot reach the tools, you have a session-state vs daemon-state divergence — not a server crash, not a config error, just a per-conversation transport handle that Claude Code does not recover on its own. The fallback is filesystem reads of the vault for the rest of the session.

**Rule:** Treat "MCP server appears dead" and "MCP server actually died" as two different failures and check them separately. The daemon-side check (`claude mcp list`) is free, takes seconds, and decides between "restart conversation" and "actually investigate the server". Doing this check first reframes the rest of the debugging — and prevents 20 minutes of poking at a healthy server.

**Tags:** `#mcp` `#claude-code` `#hive` `#transport-state` `#diagnostic-first-move`
