---
id: "dotfiles-troubleshoot-ai-tools"
type: troubleshooting
status: active
tags: [troubleshooting, dotfiles, ai, claude, gemini]
created: "2026-02-22"
owner: manu
---

# Troubleshooting: AI Tools

## Skills not loading

### Linux / macOS

```bash
# Verify skills exist
ls -la ~/.claude/skills/

# Re-run install
cd ~/Projects/dotfiles
./setup-linux.sh
```

### Windows

```powershell
# Verify skills exist
Get-ChildItem "$env:USERPROFILE\.claude\skills"

# Re-run install
cd path\to\dotfiles
.\setup-windows.ps1
```

**Common causes:**
- Setup script not run after cloning
- Skills directory missing from dotfiles repo (`ai/skills/`)
- Permissions issue on `~/.claude/skills/`

## CLAUDE.md not being read

```bash
# Verify file exists in project root
ls -la CLAUDE.md

# Or in home directory for global config
ls -la ~/.claude/CLAUDE.md
```

**Common causes:**
- File not in project root (Claude Code reads from CWD)
- File not copied by setup script (re-run `setup-linux.sh`)
- Working directory is wrong when starting `claude`

## Gemini prompts not working

```bash
# Check gp function is defined
type gp

# Verify prompts directory
ls -la ~/.gemini/prompts/
```

**Common causes:**
- `gp` function not defined (shell config not sourced)
- Prompts directory empty (setup script didn't convert skills)
- YAML frontmatter not stripped (conversion failed)

## MCP servers not connecting

```bash
# List registered servers
claude mcp list

# Inside Claude Code session
/mcp
```

**Common causes:**
- `npx` not available (install Node.js)
- Claude Code CLI not installed (`npm install -g @anthropic-ai/claude-code`)
- Server registration failed during setup (re-run manually):

```bash
claude mcp add --transport stdio drawio --scope user -- npx -y @drawio/mcp
claude mcp add --transport http socket --scope user -- https://mcp.socket.dev/
```

## Plugins not installed

```bash
# Check enabled plugins
cat ~/.claude/settings.json | jq '.enabledPlugins'
```

**Common causes:**
- Setup script skipped plugin installation
- `claude` CLI not in PATH when setup ran
- Network issue during plugin download

## project-init not found

```bash
# Check if alias/function exists
type project-init

# Linux: verify init-project.sh was copied
ls -la ~/.claude/init-project.sh

# Windows: verify init-project.ps1 was copied
# Get-ChildItem "$env:USERPROFILE\scripts\init-project.ps1"
```

**Common causes:**
- Shell config not sourced after setup
- Script not copied by setup (re-run setup)
- On Windows: PowerShell profile not loaded (restart PowerShell)

## Related

- [Runbook: AI Tools Setup](../runbooks/ai-tools-setup.md)
- [ADR-001: Custom Skills Over BMAD](../adr/adr-001-skill-based-ai-workflow.md)
