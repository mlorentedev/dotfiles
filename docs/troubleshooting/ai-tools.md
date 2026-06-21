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

## `dotf init` not found (Linux/macOS)

```bash
# dotf must be on PATH (installed by setup + install-dotf)
command -v dotf
dotf init --help
```

**Common causes:**
- Shell config not sourced after setup (open a new shell or `source ~/.zshrc`)
- `dotf` not installed / not on PATH — re-run `./setup-linux.sh`

## `project-init` not found (Windows)

```powershell
# project-init is a profile function that calls `dotf init`; verify both
Get-Command dotf
type project-init
```

**Common causes:**
- PowerShell profile not loaded (restart PowerShell)
- `dotf` not installed / not on PATH (re-run setup-windows.ps1)

## Related

- [Runbook: AI Tools Setup](../runbooks/ai-tools-setup.md)
- [ADR-001: Custom Skills Over BMAD](../adr/adr-001-skill-based-ai-workflow.md)
