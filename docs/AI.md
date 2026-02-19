# AI Tools Setup Guide

Complete guide for setting up and using AI coding assistants (Claude Code and Gemini CLI) with this dotfiles repository.

## Architecture Overview

```
~/.claude/                      # Claude Code home
├── CLAUDE.md                   # Master instructions (copied to projects)
├── init-project.sh             # Project initialization script
└── skills/                     # Custom skills for Claude Code
    ├── audit/SKILL.md          # /audit - Security review
    ├── refactor/SKILL.md       # /refactor - Code cleanup
    ├── test/SKILL.md           # /test - Test generation
    ├── doc/SKILL.md            # /doc - Documentation
    └── docker/SKILL.md         # /docker - Containerization

~/.gemini/                      # Gemini CLI home
├── GEMINI.md                   # Master instructions
└── prompts/                    # Skills (YAML frontmatter stripped)
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

# Option 1: One-time bypass
powershell -ExecutionPolicy Bypass -File .\setup-windows.ps1

# Option 2: Set policy for current user (recommended)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
.\setup-windows.ps1

# Restart PowerShell after setup
```

This automatically:
- Copies `ai/claude/*` to `~/.claude/` (or `~\.claude\` on Windows)
- Copies `ai/gemini/*` to `~/.gemini/` (or `~\.gemini\` on Windows)
- Copies `scripts/init-project.sh` to `~/.claude/` (Linux/macOS)
- Copies `scripts/init-project.ps1` to `~/scripts/` (Windows)
- Copies `ai/skills/*` to `~/.claude/skills/` (full SKILL.md with YAML frontmatter)
- Converts skills to `~/.gemini/prompts/` (YAML frontmatter stripped, flat markdown)
- Registers MCP servers (user scope, available in all projects)
- Creates the `project-init` alias/function

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

## Project Setup Workflow

### Option A: Use project-init (Recommended)

```bash
# Create new project with full configuration
project-init my-project python

# Or initialize current directory
cd existing-project
project-init . go
```

On Windows (PowerShell):
```powershell
project-init my-project python
project-init . node
```

This creates:
```
my-project/
├── CLAUDE.md              # Project instructions
├── .claude/
│   └── skills/            # All skills copied
├── tasks/
│   ├── todo.md            # Task tracking
│   └── lessons.md         # Learning log
├── src/
├── tests/
├── docs/
└── scripts/
```

### Option B: Manual Setup

```bash
cd your-project

# Copy master instructions
cp ~/.claude/CLAUDE.md .

# Copy skills (optional, for project-specific customization)
mkdir -p .claude/skills
cp -r ~/.claude/skills/* .claude/skills/
```

## Using Skills

### Claude Code Skills

Skills are specialized prompts with YAML frontmatter that Claude Code loads automatically. Each skill lives in its own directory with a `SKILL.md` file.

**Skill File Format:**

```markdown
---
name: skill-name
description: When to use this skill (helps Claude auto-load it)
---

# Instructions for Claude when this skill is invoked
```

**Available Skills:**

| Skill | Slash Command | Purpose |
|-------|---------------|---------|
| `audit` | `/audit` | Security audit, find vulnerabilities |
| `refactor` | `/refactor` | Clean up code following SOLID/DRY/KISS |
| `test` | `/test` | Generate comprehensive test suite |
| `doc` | `/doc` | Create technical documentation |
| `docker` | `/docker` | Generate Dockerfile + docker-compose |

**Invoke in Claude Code:**

```bash
# Start Claude Code in your project
claude

# Use slash commands (skills are auto-loaded based on description)
> /audit src/auth.py
> /refactor this function
> /test generate tests for the user service
> /doc document this module
> /docker create containers for this Python app
```

### Gemini CLI Skills

For Gemini, skills work as prompts via the `gp` function:

```bash
# Use the gp function (gemini prompt)
gp audit "$(cat src/auth.py)"
gp refactor "$(cat src/utils.py)"
gp test "$(cat src/service.py)"
gp doc "$(cat src/api.py)"
gp docker "Python FastAPI app with PostgreSQL"
```

## Customizing Skills

### Add a New Skill

1. Create the skill directory and file:

```bash
mkdir -p ~/.claude/skills/api
cat > ~/.claude/skills/api/SKILL.md << 'EOF'
---
name: api
description: Design RESTful API endpoints following OpenAPI 3.0 spec.
---

# ROLE: API DESIGN SPECIALIST

## TASK

Design RESTful API endpoints following OpenAPI 3.0 specification.

## STANDARDS

1. Use proper HTTP methods (GET, POST, PUT, DELETE, PATCH)
2. Include request/response schemas with examples
3. Document error responses (400, 401, 404, 500)
4. Follow naming conventions (plural nouns, kebab-case)
5. Version the API (/v1/, /v2/)

## OUTPUT

OpenAPI 3.0 YAML specification ready to use.
EOF
```

2. Sync to repo:

```bash
mkdir -p ~/Projects/dotfiles/ai/skills/api
cp ~/.claude/skills/api/SKILL.md ~/Projects/dotfiles/ai/skills/api/
dotfiles-sync
```

### Modify Existing Skills

Edit directly in `~/.claude/skills/<skill>/SKILL.md` then sync:

```bash
vim ~/.claude/skills/audit/SKILL.md
dotfiles-sync  # Push changes to repo
```

### Skill Structure Reference

Each skill must have:
- A directory named after the skill (e.g., `audit/`)
- A `SKILL.md` file inside with:
  - YAML frontmatter between `---` markers
  - `name`: becomes the `/slash-command`
  - `description`: helps Claude auto-load the skill
  - Markdown content with instructions

## Task Management Integration

Claude Code uses `tasks/todo.md` and `tasks/lessons.md` for persistent context.

### todo.md

Track project tasks:

```markdown
# Project Tasks

## Pending
- [ ] Implement user authentication
- [ ] Add rate limiting

## In Progress
- [ ] Database migration script

## Done
- [x] Project setup
```

### lessons.md

Record mistakes and prevention rules:

```markdown
# Lessons Learned

## Patterns to Avoid
- [2025-02-01] Don't use `localhost` in Docker containers - use service names

## Corrections Log
- [2025-02-01] Fixed N+1 query in user list endpoint
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GEMINI_API_KEY` | (none) | Gemini API key |
| `ANTHROPIC_API_KEY` | (none) | Claude API key (if not using OAuth) |
| `CLAUDE_HOME` | `~/.claude` | Claude Code config directory |

## Shell Aliases

After installation, these aliases are available:

### Linux / macOS (Bash/Zsh)

```bash
c              # claude (Claude Code CLI)
g              # gemini (Gemini CLI)
project-init   # Initialize project with dual AI config

# Gemini prompt function
gp <skill> <args>   # Gemini with skill (e.g., gp audit "code here")

# Claude uses slash commands inside session:
#   claude
#   > /audit src/auth.py
```

### Windows (PowerShell)

```powershell
c              # claude (Claude Code CLI)
g              # gemini (Gemini CLI)
k              # kubectl
project-init   # Initialize project with dual AI config (function)

# Example:
project-init my-project python
```

## Workflow: Daily Development

1. **Start session:**
   ```bash
   cd my-project
   claude
   ```

2. **Review context:**
   - Claude reads `CLAUDE.md` automatically
   - Review `tasks/todo.md` for pending work
   - Check `tasks/lessons.md` for project-specific rules

3. **Work on tasks:**
   ```
   > Review the pending tasks and pick the highest priority one
   > /audit the authentication module
   > /test generate tests for the changes
   ```

4. **Update tracking:**
   - Mark completed items in `tasks/todo.md`
   - Record any lessons in `tasks/lessons.md`

## Workflow: Syncing Across Machines

### Linux / macOS

```bash
# On machine A - after customizing skills
dotfiles-sync

# On machine B - pull updates
cd ~/.dotfiles
git pull
./setup-linux.sh
```

### Windows

```powershell
# Pull updates
cd path\to\dotfiles
git pull
.\setup-windows.ps1
# Restart PowerShell
```

## Troubleshooting

### Skills not loading

#### Linux / macOS

```bash
# Verify skills exist
ls -la ~/.claude/skills/

# Re-run install
cd ~/Projects/dotfiles
./setup-linux.sh
```

#### Windows

```powershell
# Verify skills exist
Get-ChildItem "$env:USERPROFILE\.claude\skills"

# Re-run install
cd path\to\dotfiles
.\setup-windows.ps1
```

### CLAUDE.md not being read

```bash
# Verify file exists in project root
ls -la CLAUDE.md

# Or in home directory for global config
ls -la ~/.claude/CLAUDE.md
```

### Gemini prompts not working

```bash
# Check gp function is defined
type gp

# Verify prompts directory
ls -la ~/.gemini/prompts/
```

## MCP Servers

MCP (Model Context Protocol) servers extend Claude Code with external tool integrations. The setup scripts register these automatically at user scope (available in all projects).

### Registered Servers

| Server | Type | Package/URL | Purpose |
|--------|------|-------------|---------|
| `drawio` | stdio | `@drawio/mcp` | Generate draw.io diagrams (XML, CSV, Mermaid) |
| `socket` | http | `https://mcp.socket.dev/` | Dependency security analysis |

### Prerequisites

- Claude Code CLI installed (`claude`)
- Node.js/npm installed (`npx`)

### Manual Registration

If the setup script skipped MCP registration (tools not installed yet), run manually:

```bash
claude mcp add --transport stdio drawio --scope user -- npx -y @drawio/mcp
claude mcp add --transport http socket --scope user -- https://mcp.socket.dev/
```

### Verify MCP Servers

```bash
# List registered servers and check connectivity
claude mcp list

# Inside Claude Code session
/mcp
```

### Adding New MCP Servers

1. Add the `claude mcp add` command to both setup scripts:
   - `setup-linux.sh` (in the MCP servers section)
   - `setup-windows.ps1` (in the MCP servers section)
2. Update this table
3. Commit and sync

## Claude Code Plugins

Plugins extend Claude Code with specialized capabilities. Install once per machine to maintain consistent environment.

### Essential Plugins (Recommended)

Install these on every new machine for a complete setup:

```bash
# Memory System (Long-term context across sessions)
claude /install claude-mem@thedotmack

# Code Quality & Security
claude /install code-simplifier@claude-plugins-official
claude /install security-guidance@claude-plugins-official
claude /install code-review@claude-plugins-official

# Git Workflow
claude /install github@claude-plugins-official
claude /install commit-commands@claude-plugins-official
claude /install pr-review-toolkit@claude-plugins-official

# LSP Support (install based on your languages)
claude /install gopls-lsp@claude-plugins-official      # Go
claude /install pyright-lsp@claude-plugins-official    # Python
claude /install typescript-lsp@claude-plugins-official # TypeScript
claude /install rust-analyzer-lsp@claude-plugins-official # Rust
```

### Plugin Reference

| Plugin | Source | Purpose |
|--------|--------|---------|
| `claude-mem` | thedotmack | Long-term memory across sessions |
| `code-simplifier` | official | Simplify complex code |
| `security-guidance` | official | Security best practices |
| `code-review` | official | Automated code review |
| `github` | official | GitHub API integration |
| `commit-commands` | official | Git commit helpers |
| `pr-review-toolkit` | official | PR review workflow |
| `gopls-lsp` | official | Go language server |
| `pyright-lsp` | official | Python language server |
| `typescript-lsp` | official | TypeScript language server |

### Optional Plugins (Project-Specific)

```bash
# Frontend Development
claude /install frontend-design@claude-plugins-official

# Feature Development Workflow
claude /install feature-dev@claude-plugins-official

# External Integrations
claude /install linear@claude-plugins-official    # Project management
claude /install playwright@claude-plugins-official # E2E testing
claude /install supabase@claude-plugins-official  # Supabase backend
claude /install slack@claude-plugins-official     # Slack notifications
```

### Verify Installation

```bash
# List installed plugins
cat ~/.claude/plugins/installed_plugins.json | jq '.plugins | keys'

# Check enabled plugins
cat ~/.claude/settings.json | jq '.enabledPlugins'
```

## Claude-Mem Plugin (Long-Term Memory)

The `claude-mem` plugin provides persistent memory across sessions. Essential for maintaining context in ongoing projects.

### How It Works

**Automatic Hooks** (no configuration needed):

| Hook | Event | Action |
|------|-------|--------|
| `SessionStart` | Session begins | Loads relevant context from previous sessions |
| `PostToolUse` | After each tool use | Saves observations indexed for search |
| `Stop` | Session ends | Summarizes and persists session data |

### Memory Search (MCP Tools)

Three-step workflow for efficient token usage:

```
1. search(query, limit, project) → Returns IDs (~50-100 tokens/result)
2. timeline(anchor=ID)           → Context around results
3. get_observations([IDs])       → Full details for filtered IDs
```

### Planning Skills

| Skill | Command | Use Case |
|-------|---------|----------|
| `make-plan` | `/claude-mem:make-plan` | Create plan with documentation discovery |
| `do` | `/claude-mem:do` | Execute plan with subagent orchestration |

**When to use make-plan + do:**
- Complex tasks with unfamiliar APIs
- Large migrations or refactors
- When you need to copy from documentation, not invent

### make-plan vs Native Plan Mode

| Aspect | Native `EnterPlanMode` | `claude-mem:make-plan` |
|--------|------------------------|------------------------|
| Memory | Current session only | Persists across sessions |
| Documentation | Manual exploration | Auto-discovery (Phase 0) |
| Verification | Optional | Mandatory between phases |
| Anti-patterns | Not tracked | Explicit guards |

### do Execution Protocol

Each phase follows:
1. **Implementation Subagent** → Copy from docs, cite sources
2. **Verification Subagent** → Prove the phase worked
3. **Anti-pattern Subagent** → Grep for known bad patterns
4. **Code Quality Subagent** → Review changes
5. **Commit Subagent** → Only if verification passes

### Example Workflow

```bash
# 1. Create a plan
claude
> /claude-mem:make-plan Add OAuth2 authentication with Google provider

# (Creates phased plan with doc references, saved to memory)

# 2. Execute the plan
> /claude-mem:do #plan-id

# (Orchestrates subagents, verifies each phase, commits incrementally)
```

### Memory Configuration

Plugin enabled in `~/.claude/settings.json`:

```json
{
  "enabledPlugins": {
    "claude-mem@thedotmack": true
  }
}
```

Data stored by background worker service. No additional configuration required.
