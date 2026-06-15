---
id: "dotfiles-runbook-ai-tools-setup"
type: runbook
status: active
tags: [runbook, dotfiles, ai, claude, gemini]
created: "2026-02-22"
owner: manu
---

# AI Tools Setup

Guide for setting up and using Claude Code and Gemini CLI with the dotfiles repository.

## Architecture

```
~/.claude/                      # Claude Code home
├── CLAUDE.md                   # Master instructions
└── skills/                     # Custom skills (21 total)
    ├── audit/SKILL.md
    ├── refactor/SKILL.md
    ├── test/SKILL.md
    └── .../SKILL.md

~/.gemini/                      # Gemini CLI home
├── GEMINI.md                   # Master instructions
└── prompts/                    # Skills (YAML frontmatter stripped)
    ├── audit.md
    └── .../
```

## Initial Setup (New Machine)

### 1. Install Dotfiles

#### Linux / macOS

```bash
git clone https://github.com/mlorentedev/dotfiles.git ~/.dotfiles
cd ~/.dotfiles
./setup-linux.sh
source ~/.zshrc
```

#### Windows (PowerShell)

```powershell
git clone https://github.com/mlorentedev/dotfiles.git
cd dotfiles
powershell -ExecutionPolicy Bypass -File .\setup-windows.ps1
# Restart PowerShell after setup
```

This automatically:
- Copies `ai/claude/*` to `~/.claude/` (or `~\.claude\` on Windows)
- Copies `ai/gemini/*` to `~/.gemini/` (or `~\.gemini\` on Windows)
- Copies `scripts/init-project.ps1` to `~/scripts/` (Windows project scaffolder)
- Copies `ai/skills/*` to `~/.claude/skills/` (full SKILL.md with YAML frontmatter)
- Converts skills to `~/.gemini/prompts/` (YAML frontmatter stripped, flat markdown)
- Registers MCP servers (user scope, available in all projects)
- Installs Claude Code plugins
- On Linux/macOS, repo scaffolding is `dotf init` (on PATH); Windows keeps the `project-init` function → `init-project.ps1` until #380

### 2. Install AI Tools

```bash
# Claude Code (npm)
npm install -g @anthropic-ai/claude-code

# Gemini CLI (pip)
pip install google-generativeai
```

### 3. Authenticate

```bash
# Claude Code - follows browser auth flow
claude

# Gemini CLI - set API key
export GEMINI_API_KEY="your-api-key"
```

## Project Setup

### Option A: Use `dotf init` (Recommended)

```bash
dotf init my-project --stack python
dotf init . --stack go         # Initialize current directory
```

On Windows (PowerShell), until a Windows `dotf` install path exists (#380):
```powershell
project-init my-project python
```

### Option B: Manual Setup

```bash
cd your-project
cp ~/.claude/CLAUDE.md .
mkdir -p .claude/skills
cp -r ~/.claude/skills/* .claude/skills/
```

## Using Skills

### Claude Code

Skills are `SKILL.md` files with YAML frontmatter that Claude Code loads automatically via slash commands.

```bash
claude
> /audit src/auth.py
> /refactor this function
> /test generate tests for the user service
> /doc document this module
> /docker create containers for this Python app
```

### Gemini CLI

Skills work as prompts via the `gp` function:

```bash
gp audit "$(cat src/auth.py)"
gp refactor "$(cat src/utils.py)"
```

## Adding a New Skill

```bash
# 1. Create the skill directory and file
mkdir -p ai/skills/api
cat > ai/skills/api/SKILL.md << 'EOF'
---
name: api
description: Design RESTful API endpoints following OpenAPI 3.0 spec.
---

# ROLE: API DESIGN SPECIALIST
...
EOF

# 2. Deploy
./setup-linux.sh    # Or setup-windows.ps1

# 3. Sync to dotfiles repo
dotfiles-sync
```

### Skill Structure

Each skill requires:
- A directory named after the skill (e.g., `audit/`)
- A `SKILL.md` file inside with:
  - YAML frontmatter between `---` markers
  - `name`: becomes the `/slash-command`
  - `description`: helps Claude auto-load the skill
  - Markdown content with instructions

## MCP Servers

MCP servers extend Claude Code with external tool integrations. Registered automatically at user scope.

| Server | Transport | Package/URL | Purpose |
|--------|-----------|-------------|---------|
| `drawio` | stdio | `@drawio/mcp` | Generate draw.io diagrams |
| `socket` | http | `https://mcp.socket.dev/` | Dependency security analysis |

### Manual Registration

```bash
claude mcp add --transport stdio drawio --scope user -- npx -y @drawio/mcp
claude mcp add --transport http socket --scope user -- https://mcp.socket.dev/
```

### Verify

```bash
claude mcp list
# Or inside Claude Code session: /mcp
```

### Adding New MCP Servers

1. Add the `claude mcp add` command to both setup scripts (`setup-linux.sh` and `setup-windows.ps1`)
2. Commit and sync

## Plugins

### Essential (Automated via setup scripts)

| Plugin | Purpose |
|--------|---------|
| `claude-mem@thedotmack` | Long-term memory across sessions |
| `code-simplifier` | Simplify complex code |
| `github` | GitHub API integration |
| `security-guidance` | Security best practices |
| `claude-md-management` | CLAUDE.md management |
| `claude-code-setup` | Project setup assistance |
| `frontend-design` | Frontend design |
| `ralph-loop` | Iterative development loop |
| `code-review` | Automated code review |
| `commit-commands` | Git commit helpers |
| `pr-review-toolkit` | PR review workflow |

### Language-Specific (Optional, per-machine)

```bash
claude /install gopls-lsp@claude-plugins-official        # Go
claude /install pyright-lsp@claude-plugins-official       # Python
claude /install typescript-lsp@claude-plugins-official    # TypeScript
```

## Claude-Mem Plugin

The `claude-mem` plugin provides persistent memory across sessions.

### Memory Search (MCP Tools)

```
1. search(query, limit, project) → Returns IDs
2. timeline(anchor=ID)           → Context around results
3. get_observations([IDs])       → Full details for filtered IDs
```

### Planning Skills

| Skill | Command | Use Case |
|-------|---------|----------|
| `make-plan` | `/claude-mem:make-plan` | Create plan with documentation discovery |
| `do` | `/claude-mem:do` | Execute plan with subagent orchestration |

## Shell Aliases

| Alias | Command | Platform |
|-------|---------|----------|
| `c` | `claude` | All |
| `g` | `gemini` | All |
| `dotf init` | Scaffold a new fully-practiced repo | Linux/macOS |
| `project-init` | Windows scaffolder (-> init-project.ps1) | Windows |
| `gp <skill> <args>` | Gemini prompt function | Linux/macOS |

## Daily Development Workflow

1. **Start session:** `cd my-project && claude`
2. **Review context:** Claude reads `CLAUDE.md` + `tasks/todo.md` + `tasks/lessons.md`
3. **Work:** Use slash commands (`/audit`, `/test`, `/refactor`)
4. **Update tracking:** Mark completed items, record lessons

## Syncing Across Machines

```bash
# Machine A - after customizing skills
dotfiles-sync

# Machine B - pull updates
cd ~/.dotfiles && git pull && ./setup-linux.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GEMINI_API_KEY` | (none) | Gemini API key |
| `ANTHROPIC_API_KEY` | (none) | Claude API key (if not using OAuth) |
| `CLAUDE_HOME` | `~/.claude` | Claude Code config directory |

## Related

- [Troubleshooting: AI Tools](../troubleshooting/ai-tools.md) — Common AI tool issues
- [ADR-001](../adr/adr-001-skill-based-ai-workflow.md) — Custom skills over BMAD
- Project overview — see the repo `README.md` (strategic context lives in the maintainer's knowledge store)
