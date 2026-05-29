---
id: "dotfiles-adr-007-mcp-persistence"
type: adr
adr: "007"
title: MCP Server Persistence and Auto-Memory in Vault
tags: [adr, dotfiles, claude-code, mcp, auto-memory, vault]
status: accepted
created: "2026-02-26"
owner: manu
---

# ADR-007: MCP Server Persistence and Auto-Memory in Vault

## Context

Claude Code MCP servers and auto-memory were configured manually per machine, creating drift:

| Component | Before | Problem |
|-----------|--------|---------|
| drawio, socket | Deployed by setup scripts | OK |
| sequential-thinking | Only in `~/.claude.json` (manual) | Lost on new machine |
| context7 | Only in kubelab project config | Not available globally |
| Auto-memory | `~/.claude/projects/*/memory/` (local) | Lost on machine failure, not portable |

An initial approach stored memory files in the dotfiles repo (`ai/claude/memory/`), but this violated the Neural Hive protocol: **"Code lives in Git. Knowledge lives in the vault."** Auto-memory is knowledge, not code.

## Decision

### MCP Servers: All Global via Setup Scripts

Register all MCP servers with `--scope user` in both setup scripts:

| Server | Transport | Purpose |
|--------|-----------|---------|
| drawio | stdio | Diagram generation |
| socket | http | Dependency security scoring |
| sequential-thinking | stdio | Structured reasoning for complex decisions |
| context7 | http | Version-accurate library documentation |

Context7 moved from per-project (kubelab only) to global scope — it has no project-specific state, only fetches third-party docs.

### Auto-Memory: Lives in the Vault

Memory files live in the knowledge vault at `10_projects/<project>/memory/MEMORY.md`, alongside ADRs, lessons, and tasks for each project.

Setup scripts create symlinks (Linux) or junctions (Windows) from Claude Code's internal path to the vault:

```
~/.claude/projects/<encoded>/memory/  →  ~/Projects/knowledge/<scope>/<project>/memory/
```

- **Linux:** Symlinks (bidirectional — Claude writes, Obsidian syncs)
- **Windows:** Junctions (bidirectional, no admin privileges required)

The setup scripts scan two vault scopes for `memory/` directories:

| Scope | Vault Path | CWD Mapping |
|-------|-----------|-------------|
| `10_projects/` | `10_projects/<name>/memory/` | Convention: `~/Projects/<name>` |
| `50_work/` | `50_work/**/<name>/memory/` | Vault path itself (recursive scan) |

No hardcoded project list needed — auto-discovery via filesystem scan.

### CLAUDE.md: Auto-Invocation Rules

Added MCP usage rules to global CLAUDE.md (`ai/claude/CLAUDE.md`):
- **Context7:** Auto-invoke when writing code with third-party libraries
- **Sequential Thinking:** Auto-invoke when the Socratic Guardrail triggers

## Sync Flow

```
Claude writes MEMORY.md
       ↓ (symlink on Linux, junction on Windows)
~/Projects/knowledge/<scope>/<project>/memory/MEMORY.md
       ↓ (Obsidian git sync every 10 min)
Remote vault repo
       ↓ (pull on other machine)
~/Projects/knowledge/<scope>/<project>/memory/MEMORY.md
       ↓ (symlink/junction created by setup script)
~/.claude/projects/<encoded>/memory/MEMORY.md
       ↓
Claude reads MEMORY.md on next session
```

No manual commits needed — Obsidian's auto-sync handles persistence.
Both Linux and Windows achieve full bidirectional sync.

## Consequences

### Positive

- **Follows Neural Hive:** Knowledge in vault, code in git — no mixing
- **Auto-sync:** Obsidian git sync every 10 min means near-zero friction
- **No dotfiles noise:** Memory changes don't pollute dotfiles commit history
- **Privacy:** Vault is private repo; dotfiles is public
- **Co-location:** Each project's memory sits with its ADRs, lessons, and tasks
- **Auto-discovery:** Setup script finds all projects with `memory/` dirs — no config needed
- **Full MCP portability:** All 4 servers deployed on any new machine

### Negative

- **Two repos required:** Both dotfiles (setup scripts) and vault (memory content) must exist
- **Obsidian dependency:** Sync relies on Obsidian git plugin on each machine
- **Windows junctions local-only:** Junctions only work for local paths (not network drives or OneDrive-synced folders)
- **Path assumption:** Assumes `~/Projects/<name>` layout matches vault `10_projects/<name>`

### Mitigations

- Vault is already cloned on all machines (prerequisite for Neural Hive protocol)
- Obsidian git plugin is part of the standard machine setup
- Windows junctions are bidirectional and require no admin — full parity with Linux
- Setup script is idempotent — re-run after adding new projects to vault
