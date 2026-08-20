# Guide: Cross-Agent Session Memory Bridge

> Implements ADR-014 (`docs/adr/adr-014-cross-agent-session-memory-bridge.md`) and resolves MEMORY-001 (Issue #1092).

## Overview

The Neural Hive architecture uses the **personal knowledge vault as the single sink of agent memory** (GUARD-001 / ADR-007). To prevent context loss when switching across different AI coding agents (Claude Code, OpenCode, Antigravity, Copilot), the cross-agent session memory bridge unifies session startup context and end-of-session handoffs.

```
+-------------------------------------------------------------------------------+
|                             CROSS-AGENT MEMORY FLOW                           |
+-------------------------------------------------------------------------------+
|                                                                               |
|  1. STARTUP (SessionStart)                                                    |
|     `dotf mem session-start`                                                  |
|      - Generates/reads session brief from vault memory                        |
|      - Injects into active agent context                                      |
|                                                                               |
|  2. EXECUTION                                                                 |
|      - Agent operates on tasks / specs / bitácora board                       |
|      - Memory single-sink invariant strictly enforced                         |
|                                                                               |
|  3. HANDOFF (SessionEnd / Crystallization)                                    |
|      - Extracts ## Session Handoff structured section                         |
|      - Archives record to `<vault>/10_projects/<project>/sessions/<date>.md`  |
|      - Updates latest project MEMORY.md snapshot                              |
|                                                                               |
+-------------------------------------------------------------------------------+
```

---

## Agent Integration Matrix

| Agent | Startup Hook | End-of-Session Trigger | Implementation Mechanism |
|---|---|---|---|
| **Claude Code** | `SessionStart` hook | `SessionEnd` hook | Native hook in `ai/claude/settings.json` invoking `dotf mem session-end` |
| **OpenCode** | Session context / AGENTS.md | `/handoff` slash command | Deployed from vault skills (`~/.config/opencode/commands/handoff.md`) |
| **Antigravity** | System prompt / AGENTS.md | `/handoff` skill | Native skill in `builtin/skills/` & prompt injection |
| **Copilot CLI** | AGENTS.md instructions | Manual `/handoff` | Explicitly excluded from daemon hooks (see Copilot Decision below) |

---

## Agent-by-Agent Details

### 1. Claude Code
Claude Code provides native process lifecycle hooks. These are registered declaratively in `ai/claude/settings.json`:
```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": ".*",
        "hooks": [
          { "type": "command", "command": "$HOME/.local/bin/dotf mem session-start" }
        ]
      }
    ],
    "SessionEnd": [
      {
        "matcher": ".*",
        "hooks": [
          { "type": "command", "command": "$HOME/.local/bin/dotf mem session-end" }
        ]
      }
    ]
  }
}
```
`setup-linux.sh` and `setup-windows.ps1` merge these hooks into `~/.claude/settings.json`.

### 2. OpenCode
OpenCode's configuration schema (`opencode.jsonc`) supports model providers, agents, instructions, formatters, and MCP tools, but does not provide background daemon lifecycle hooks. Continuity in OpenCode is delivered via the `/handoff` command:
- Deployed by `compile-harness.sh --deploy` to `~/.config/opencode/commands/handoff.md`.
- Invoking `/handoff` formats the structured `## Session Handoff` markdown block and writes the session record to the project's vault sessions folder.

### 3. Antigravity (AGY)
Antigravity executes skills from the harness repository (`builtin/skills/` and `$VAULT_PATH/00_meta/skills/`). The `/handoff` skill formats decisions, open threads, and next actions, ensuring seamless handoffs to subsequent agent sessions.

### 4. GitHub Copilot CLI Decision
GitHub Copilot CLI operates as a standalone CLI tool without background event hooks or a plugin execution runtime. 
- **Design Decision**: Copilot CLI is intentionally excluded from automated daemon hooks. Continuity for Copilot sessions relies on manual execution of the `/handoff` routine and adherence to AGENTS.md standing orders.

---

## Tooling Parity: `dotf mem`

Per ADR-020 (Go CLI Convergence), the legacy `session-handoff.sh` and `session-handoff.ps1` twins were replaced by the cross-platform Go binary `dotf mem`:
- `dotf mem session-start`: Prepares and injects context briefs.
- `dotf mem session-end`: Parses the handoff payload, filters out trivial sessions, and writes the persistent markdown session record.
