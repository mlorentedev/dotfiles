#!/bin/bash
# init-project.sh
# Purpose: Initialize a new project with full vault entry + AI memory
#
# Usage:
#   ./init-project.sh [project-name] [stack]           personal coding project
#   ./init-project.sh --work-sdk <family> <component>  work SDK vault entry only
#
# --work-sdk creates only the vault entry under 50_work/45-development/<family>/<component>/
# Fill in source_path in the generated 00-context.md with the real repo path.

set -euo pipefail

# Handle --work-sdk mode before argument parsing
if [ "${1:-}" = "--work-sdk" ]; then
    FAMILY="${2:?Usage: init-project.sh --work-sdk <family-slug> <component>}"
    COMPONENT="${3:?Usage: init-project.sh --work-sdk <family-slug> <component>}"
    KNOWLEDGE_HOME="${HOME}/Projects/knowledge"
    TODAY=$(date +%Y-%m-%d)
    SDK_DIR="$KNOWLEDGE_HOME/50_work/45-development/$FAMILY/$COMPONENT"

    printf '[INFO] Creating work SDK vault entry: 50_work/45-development/%s/%s\n' "$FAMILY" "$COMPONENT"
    mkdir -p "$SDK_DIR/30-architecture" "$SDK_DIR/40-runbooks" "$SDK_DIR/50-troubleshooting" "$SDK_DIR/memory"

    cat > "$SDK_DIR/00-context.md" << EOF
---
id: "${FAMILY}-${COMPONENT}"
type: project
status: active
owner: manu
source_path:
  code: "\${PROJECTS_PATH}/<ProductFamily>/<component>"
  onedrive: "\${ONEDRIVE_PATH}/Products/<ProductFamily>"
stack: []
tags: [work, sdk, $FAMILY]
created: "$TODAY"
---

# ${COMPONENT}: Work SDK Context

> **Goal:** [Describe this repo's purpose within the $FAMILY product family]

## Technical Stack
- **Language:**
- **Key Tools:**

## Repos in this Family
See [[../00-context|$FAMILY family context]] for all repos.

## Critical Links
- **Real repo path:** Fill in source_path above with actual \${PROJECTS_PATH}/... path

## Knowledge Structure
- [[30-architecture]]: ADRs & Technical Decisions
- [[40-runbooks]]: Operational Guides
- [[50-troubleshooting]]: Known Issues
- [[memory/MEMORY]]: Session memory
- [[90-lessons]]: Lessons Learned

## Development Flow
1. Open Claude in the real repo (see source_path)
2. Session hook links this vault entry as MEMORY context
EOF

    cat > "$SDK_DIR/memory/MEMORY.md" << EOF
# ${COMPONENT} — Work SDK Session Memory

## Session Handoff
> Updated: $TODAY
**Last task:** Vault entry initialized
**Decisions:** None
**Open threads:** Fill source_path in 00-context.md with real repo path
**Next action:** Open Claude in real repo; session hook will create symlink automatically

## User Preferences
- Follow global CLAUDE.md + workflow-protocol.md rules

## Last Crystallized: $TODAY
EOF

    cat > "$SDK_DIR/90-lessons.md" << EOF
---
id: "${FAMILY}-${COMPONENT}-lessons"
type: lesson
status: active
owner: manu
tags: [work, sdk, $FAMILY]
---
# Lessons Learned — ${COMPONENT}

*Record non-trivial bugs and decisions here. Promoted to 00_meta/patterns/ if recurring.*
EOF

    # Create parent family context if it doesn't exist
    FAMILY_CONTEXT="$KNOWLEDGE_HOME/50_work/45-development/$FAMILY/00-context.md"
    if [ ! -f "$FAMILY_CONTEXT" ]; then
        cat > "$FAMILY_CONTEXT" << EOF
---
id: "${FAMILY}"
type: project
status: active
owner: manu
tags: [work, sdk, product-family]
created: "$TODAY"
---

# ${FAMILY}: Product Family Context

> Umbrella context for all repos in the $FAMILY product family.

## Repos

| Component | Vault Path | Real Path |
|-----------|-----------|-----------|
| $COMPONENT | [[${COMPONENT}/00-context]] | Fill in source_path |

## Work Context
- Product knowledge: [[../../20-products/${FAMILY}/00-overview|product overview]] (if exists)

EOF
        printf '[SUCCESS] Created family context: %s\n' "$FAMILY_CONTEXT"
    else
        printf '[INFO] Family context already exists: %s\n' "$FAMILY_CONTEXT"
    fi

    printf '[SUCCESS] Work SDK vault entry created: %s\n' "$SDK_DIR"
    printf '[INFO] Next steps:\n'
    printf '  1. Fill in source_path in %s/00-context.md\n' "$SDK_DIR"
    printf '  2. Open Claude in the real repo — session hook creates symlink automatically\n'
    exit 0
fi

# REFACTOR-004: parse --skip-{agents,standards,github} flags before positional
# args so existing call signatures (project-name + stack) still work.
SKIP_AGENTS=0
SKIP_STANDARDS=0
SKIP_GITHUB=0
_POSITIONAL=()
while [ $# -gt 0 ]; do
    case "$1" in
        --skip-agents) SKIP_AGENTS=1; shift ;;
        --skip-standards) SKIP_STANDARDS=1; shift ;;
        --skip-github) SKIP_GITHUB=1; shift ;;
        --) shift; while [ $# -gt 0 ]; do _POSITIONAL+=("$1"); shift; done ;;
        *) _POSITIONAL+=("$1"); shift ;;
    esac
done
set -- "${_POSITIONAL[@]:-}"

PROJECT_NAME="${1:-.}"
STACK="${2:-python}"
# Using .claude as the central config hub (backward compatibility)
AI_CONFIG_HOME="${HOME}/.claude"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { printf '%b[INFO]%b %s\n' "$BLUE" "$NC" "$1"; }
log_success() { printf '%b[SUCCESS]%b %s\n' "$GREEN" "$NC" "$1"; }
log_error() { printf '%b[ERROR]%b %s\n' "$RED" "$NC" "$1"; }

# Create project directory if not current
if [[ "$PROJECT_NAME" != "." ]]; then
    mkdir -p "$PROJECT_NAME"
    cd "$PROJECT_NAME"
    log_info "Created project directory: $PROJECT_NAME"
fi

# Base structure
log_info "Creating project structure..."
mkdir -p src tests scripts .claude/skills

# --- AI MEMORY INJECTION ---

# Copy CLAUDE.md
if [[ -f "$AI_CONFIG_HOME/CLAUDE.md" ]]; then
    cp "$AI_CONFIG_HOME/CLAUDE.md" .
    log_success "Injected CLAUDE.md"
else
    log_info "CLAUDE.md not found in global config"
fi

# Copy GEMINI.md
if [[ -f "$AI_CONFIG_HOME/GEMINI.md" ]]; then
    cp "$AI_CONFIG_HOME/GEMINI.md" .
    log_success "Injected GEMINI.md"
else
    log_info "GEMINI.md not found in global config"
fi

# Copy skills (Claude Code specific)
if [[ -d "$AI_CONFIG_HOME/skills" ]]; then
    for skill_dir in "$AI_CONFIG_HOME/skills/"*/; do
        if [[ -d "$skill_dir" ]]; then
            skill_name=$(basename "$skill_dir")
            mkdir -p ".claude/skills/$skill_name"
            # Using cp -r with error suppression in case directory is empty
            cp -r "$skill_dir"* ".claude/skills/$skill_name/" 2>/dev/null || true
        fi
    done
    log_success "Injected skills to .claude/skills/"
fi

# --- KNOWLEDGE VAULT INTEGRATION ---
KNOWLEDGE_HOME="${HOME}/Projects/knowledge"
# Ensure we get the actual project name, even if PROJECT_NAME is "."
ACTUAL_PROJECT_NAME=$(basename "$(pwd)")
PROJECT_KB_DIR="${KNOWLEDGE_HOME}/10_projects/${ACTUAL_PROJECT_NAME}"

log_info "Initializing Knowledge Vault structure..."
TODAY=$(date +%Y-%m-%d)
mkdir -p "$PROJECT_KB_DIR/30-architecture" "$PROJECT_KB_DIR/40-runbooks" "$PROJECT_KB_DIR/50-troubleshooting" "$PROJECT_KB_DIR/memory"

# 00-context.md (full project context — fill in after init)
if [ ! -f "$PROJECT_KB_DIR/00-context.md" ]; then
cat > "$PROJECT_KB_DIR/00-context.md" << EOF
---
id: "${ACTUAL_PROJECT_NAME}"
type: project
status: active
owner: manu
repo_url: ""
stack: [$STACK]
tags: []
created: "$TODAY"
---

# ${ACTUAL_PROJECT_NAME}: Project Context

> **Goal:** [Concise, single-sentence project description]

## Technical Stack
- **Language:** $STACK
- **Frameworks:**
- **Infrastructure:**
- **Key Tools:**

## Critical Links
- **Repository:** ~/Projects/${ACTUAL_PROJECT_NAME}
- **Production:**

## Knowledge Structure
- [[10-roadmap]]: Planning & Strategy
- [[11-tasks]]: Active Backlog
- [[30-architecture]]: ADRs & Technical Decisions
- [[40-runbooks]]: Operational Guides
- [[50-troubleshooting]]: Known Issues
- [[90-lessons]]: Lessons Learned

## Development Flow
1. Feature branch from \`main\`
2. PR with tests
3. Merge → deploy
EOF
fi

# 10-roadmap.md
if [ ! -f "$PROJECT_KB_DIR/10-roadmap.md" ]; then
cat > "$PROJECT_KB_DIR/10-roadmap.md" << EOF
---
id: "${ACTUAL_PROJECT_NAME}-roadmap"
type: roadmap
status: active
owner: manu
created: "$TODAY"
---

# ${ACTUAL_PROJECT_NAME}: Roadmap & Vision

> **Vision:** [One-sentence vision]

## Strategic Themes

### 1. [Theme Name]
* **Goal:**
* **Status:** Not started
* **Next:**
EOF
fi

# 11-tasks.md
if [ ! -f "$PROJECT_KB_DIR/11-tasks.md" ]; then
cat > "$PROJECT_KB_DIR/11-tasks.md" << EOF
---
id: "${ACTUAL_PROJECT_NAME}-tasks"
type: project
status: active
owner: manu
tags: []
---
# ${ACTUAL_PROJECT_NAME}: Active Backlog

Progress: [..........] 0%

## Pending

- [ ] Fill in 00-context.md with project details

## In Progress

## Done
EOF
fi

# 90-lessons.md
if [ ! -f "$PROJECT_KB_DIR/90-lessons.md" ]; then
cat > "$PROJECT_KB_DIR/90-lessons.md" << EOF
---
id: "${ACTUAL_PROJECT_NAME}-lessons"
type: lesson
status: active
owner: manu
tags: []
---
# Lessons Learned — ${ACTUAL_PROJECT_NAME}

*Non-trivial bugs and decisions. Promoted to 00_meta/patterns/ if recurring across projects.*
EOF
fi

# memory/MEMORY.md
if [ ! -f "$PROJECT_KB_DIR/memory/MEMORY.md" ]; then
cat > "$PROJECT_KB_DIR/memory/MEMORY.md" << EOF
# ${ACTUAL_PROJECT_NAME}

## Session Handoff
> Updated: $TODAY
**Last task:** Project initialized via init-project.sh
**Decisions:** None
**Open threads:** Fill 00-context.md with actual project details
**Next action:** Run /crystallize after first sprint

## User Preferences
- Follow global CLAUDE.md + workflow-protocol.md rules

## Architecture Map
- See [[00-context]] for stack and entry points

## Last Crystallized: $TODAY
EOF
fi

# Create memory symlink immediately — don't wait for first Claude session
ENCODED_CWD=$(printf '%s' "$(pwd)" | tr '/' '-')
CLAUDE_PROJECT_DIR="$HOME/.claude/projects/$ENCODED_CWD"
mkdir -p "$CLAUDE_PROJECT_DIR"
if [ ! -L "$CLAUDE_PROJECT_DIR/memory" ] && [ ! -d "$CLAUDE_PROJECT_DIR/memory" ]; then
    ln -sfn "$PROJECT_KB_DIR/memory" "$CLAUDE_PROJECT_DIR/memory" 2>/dev/null && \
        log_success "Created memory symlink → vault"
fi

log_success "Created Knowledge Vault entry in 10_projects/${ACTUAL_PROJECT_NAME}"

# --- STACK INITIALIZATION ---

case $STACK in
    python)
        log_info "Initializing Python project..."
        if command -v poetry >/dev/null 2>&1; then
            poetry init -n --name "$PROJECT_NAME" 2>/dev/null || true
            poetry add --group dev typer rich pydantic pytest pytest-cov mypy ruff 2>/dev/null || true
        elif command -v uv >/dev/null 2>&1; then
            uv init 2>/dev/null || true
        fi
        ;;
    go)
        log_info "Initializing Go project..."
        if command -v go >/dev/null 2>&1; then
            go mod init "$PROJECT_NAME" 2>/dev/null || true
        fi
        cat > Makefile << 'EOF'
.PHONY: build test lint run

build:
    go build -o bin/app ./cmd/...

test:
    go test -v -race -cover ./...

lint:
    golangci-lint run

run:
    go run ./cmd/main.go
EOF
        ;;
    node|ts)
        log_info "Initializing Node/TypeScript project..."
        if command -v npm >/dev/null 2>&1; then
            npm init -y 2>/dev/null || true
            npm i -D typescript @types/node tsx vitest 2>/dev/null || true
        fi
        ;;
    *)
        log_info "No stack-specific initialization for: $STACK"
        ;;
esac

# --- GIT & IGNORE ---

# Git initialization
if [[ ! -d .git ]]; then
    git init
    log_success "Initialized git repository"
fi

# Activate pre-commit hooks (gitleaks) if pre-commit is available
if command -v pre-commit >/dev/null 2>&1 && [[ -f .pre-commit-config.yaml ]]; then
    pre-commit install >/dev/null 2>&1 && log_success "Activated pre-commit hooks"
fi

# Create .pre-commit-config.yaml (gitleaks for agent-touched repos)
if [[ ! -f .pre-commit-config.yaml ]]; then
    cat > .pre-commit-config.yaml << 'EOF'
# Threat model: code repo. Agents may paste secrets into test fixtures, env
# samples, or doc snippets. gitleaks blocks them before they reach git history.
#
# Setup (one-time):
#   pre-commit install
#
# Bypass (only with explicit user approval):
#   SKIP=gitleaks git commit ...

repos:
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.30.1
    hooks:
      - id: gitleaks
EOF
    log_success "Created .pre-commit-config.yaml (gitleaks v8.30.1)"
fi

# Create .gitignore if not exists
if [[ ! -f .gitignore ]]; then
    cat > .gitignore << 'EOF'
# Dependencies
node_modules/
__pycache__/
.venv/
vendor/

# Build
dist/
build/
bin/
*.egg-info/

# IDE
.idea/
.vscode/
*.swp

# Environment
.env
.env.local
*.secret

# OS
.DS_Store
Thumbs.db

# Test/Coverage
.coverage
htmlcov/
.pytest_cache/
coverage.out
EOF
    log_success "Created .gitignore"
fi

# --- REPO BOOTSTRAP (REFACTOR-004) ---
# Wire the standalone init-repo-* helpers into the default init-project flow.
# Each invocation is non-fatal: a helper failure logs a warning and continues
# rather than aborting the whole init. github-defaults is auto-skipped when no
# `origin` remote is configured (typical for a brand-new local repo).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

if [ "$SKIP_AGENTS" = "0" ] && [ -x "$SCRIPT_DIR/init-repo-agents.sh" ]; then
    log_info "Bootstrapping AGENTS.md (init-repo-agents)..."
    "$SCRIPT_DIR/init-repo-agents.sh" --repo "$(pwd)" || log_error "init-repo-agents failed (non-fatal); continuing"
fi

if [ "$SKIP_STANDARDS" = "0" ] && [ -x "$SCRIPT_DIR/init-repo-standards.sh" ]; then
    log_info "Bootstrapping docs/standards.md (init-repo-standards)..."
    "$SCRIPT_DIR/init-repo-standards.sh" --repo "$(pwd)" || log_error "init-repo-standards failed (non-fatal); continuing"
fi

if [ "$SKIP_GITHUB" = "0" ] && [ -x "$SCRIPT_DIR/init-repo-github-defaults.sh" ]; then
    if git config --get remote.origin.url >/dev/null 2>&1; then
        log_info "Applying GitHub defaults (init-repo-github-defaults)..."
        "$SCRIPT_DIR/init-repo-github-defaults.sh" || log_error "init-repo-github-defaults failed (non-fatal); continuing"
    else
        log_info "No origin remote yet; skipping init-repo-github-defaults (add a remote then re-run it manually)"
    fi
fi

log_success "Project initialized with Dual AI Configuration"
log_info "Structure:"
echo "  CLAUDE.md / GEMINI.md      - AI Memory files (Dual-Core)"
echo "  .claude/skills/            - Custom skills"
echo "  Knowledge Vault updated    - 11-tasks.md & 90-lessons.md in knowledge/10_projects/"