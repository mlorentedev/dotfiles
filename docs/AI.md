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

```bash
git clone https://github.com/mlorentedev/dotfiles.git ~/.dotfiles
cd ~/.dotfiles
./install.sh
source ~/.zshrc
```

This automatically:
- Copies `ai/claude/*` to `~/.claude/`
- Copies `ai/gemini/*` to `~/.gemini/`
- Copies `scripts/init-project.sh` to `~/.claude/`
- Copies `ai/skills/*` to `~/.claude/skills/` (full SKILL.md with YAML frontmatter)
- Converts skills to `~/.gemini/prompts/` (YAML frontmatter stripped, flat markdown)
- Creates the `claude-init` alias

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

### Option A: Use claude-init (Recommended)

```bash
# Create new project with full configuration
claude-init my-project python

# Or initialize current directory
cd existing-project
claude-init . go
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

```bash
c              # claude (Claude Code CLI)
g              # gemini (Gemini CLI)
claude-init    # Initialize project with Claude config

# Gemini prompt function
gp <skill> <args>   # Gemini with skill (e.g., gp audit "code here")

# Claude uses slash commands inside session:
#   claude
#   > /audit src/auth.py
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

```bash
# On machine A - after customizing skills
dotfiles-sync

# On machine B - pull updates
cd ~/.dotfiles
git pull
./install.sh
```

## Troubleshooting

### Skills not loading

```bash
# Verify skills exist
ls -la ~/.claude/skills/

# Re-run install
cd ~/Projects/dotfiles
./install.sh
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
