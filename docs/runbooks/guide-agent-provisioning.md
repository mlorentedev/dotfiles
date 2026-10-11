---
id: guide-agent-provisioning
type: runbook
status: active
created: "2026-10-10"
tags: [runbook, dotfiles, ai, harness]
---

# Agent provisioning

> Reference: what a machine gets for each AI agent, and which file declares it. Replaces `ai-tools-setup.md`, which described a layout dismantled long before (audit-009 D2). Every row here names its source of truth; when this page and that file disagree, the file wins.

## How it is applied

`install.sh` (or `install.ps1`) installs `dotf` and hands off to `dotf converge`, which applies each step in order and reports what changed. `dotf converge --plan` shows the same report without writing. A converged machine reports zero changes.

| Converge step | What it does for agents | Declared in |
|---|---|---|
| `records-harness` | instruction files, skills, Claude subagents (`compile-harness.sh --deploy`) | `harness/manifest.json` (`presence`, `skills.deploy`, `doctrine.deploy`, `agents.deploy`) |
| `tools` | installs the agent CLIs the catalog holds | `packages.json` |
| `configs-deploy` | agent settings, models and MCP config files | `ai/deploy.json` |
| `records-bind` | session and gate hooks | `harness/manifest.json` (`agents.bind`, `agents.bind_named`) |

On Linux and Windows, converge runs the setup script as its last step, and that script still installs the agents the catalog lacks. On macOS no setup script runs (ADR-045).

Claude Code's MCP servers and plugins are not a converge step yet (#1843 B10): a bare `dotf deploy` applies them, so run it once after converge.

## Per agent

| Agent | Installed by | Instruction file | Skills | Configs (`ai/deploy.json`) | Hooks |
|---|---|---|---|---|---|
| Claude Code | `setup-linux.sh` only (#2317, AI-041) | `~/.claude/CLAUDE.md` | `~/.claude/skills`, subagents in `~/.claude/agents` | `claude-settings` (merge) | `SessionStart`/`SessionEnd` (`dotf mem`), `PreToolUse` gate, `UserPromptSubmit` suggest, in `settings.json` |
| opencode | catalog (npm `opencode-ai`) | `~/.config/opencode/AGENTS.md` | `~/.config/opencode/commands` | `opencode` (secrets rendered), `opencode-tui` | plugin `dotfiles-gate.ts`, not written by bind |
| pi | `setup-linux.sh` only (#2317) | `~/.pi/agent/AGENTS.md` | `~/.pi/agent/skills` | `pi` (secrets rendered), `pi-mcp`, `pi-nan-provider`, `pi-compaction` | extension `dotfiles-gate.ts`, not written by bind |
| Copilot CLI | catalog (npm `@github/copilot`) | `~/.copilot/copilot-instructions.md`, when `copilot` is on PATH | `~/.copilot/skills` | `copilot-settings`, `copilot-config`, `copilot-mcp` | — |
| agy (Antigravity) | `setup-linux.sh` only (#2317, AI-041) | compact doctrine in `~/.gemini/GEMINI.md`; `~/.gemini/AGY.md` | `~/.gemini/skills`, `~/.gemini/prompts` | `agy-settings` (merge), `agy-instructions`, `agy-geminiignore` | named hooks in `~/.gemini/config/hooks.json` |
| Codex | not installed by dotfiles | compact doctrine in `~/.codex/AGENTS.md` | — | — | — |

agy and Codex get the compact doctrine because each caps what it reads (agy 12,000 characters per rules file, Codex 32 KiB across the instruction chain); the reasons sit beside each row of `doctrine.deploy`.

A `merge` entry writes only the keys the template names and keeps everything the agent wrote itself. One exception is open: agy removes `search_web(*)` from its grants on every save, so converge re-adds it each run (#2312).

## MCP servers

Claude Code's list is `mcp-servers.json`: `sequential-thinking`, `context7`, `hive` and `pdf-modifier`. A bare `dotf deploy` registers them at user scope, skipping a server whose prerequisite binary (`npx`, `uv`) is absent. pi and Copilot read their own files, `ai/pi/mcp.json` and `ai/copilot/mcp-config.json`, deployed by converge; those are separate lists, kept in step by hand.

## Claude Code plugins

`ai/claude/plugins.json` lists them. A bare `dotf deploy` installs the ones `claude plugin list` lacks and leaves the rest alone. It runs `claude` with `CLAUDE_CONFIG_DIR` pinned, so `.claude.json` lands in the config directory and not in your home root.

## What stays manual

- **Signing in.** Each agent's first run asks for its own login (`claude`, `copilot`, `agy`, …). That is a consent, not configuration.
- **API keys.** They are never exported into the shell. A config that needs one is rendered through `dotf secrets render`, and a one-off command gets it with `dotf secrets run -- <cmd>` (ADR-028).

## Check it

```bash
dotf converge --plan     # every step reports OK on a converged machine
dotf deploy --dry-run    # Claude Code's MCP servers and plugins too
dotf doctor              # instruction files, hooks, configs and MCP per agent
```

## Related

- [`guide-antigravity-cli-migration.md`](guide-antigravity-cli-migration.md): agy setup, permissions and models.
- [`guide-opencode-go-setup.md`](guide-opencode-go-setup.md): opencode and the NaN provider.
- [`guide-cross-agent-memory.md`](guide-cross-agent-memory.md): the session hooks' memory side.
- [Troubleshooting: AI tools](../troubleshooting/ai-tools.md).
