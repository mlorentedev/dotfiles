#!/bin/bash
# ~/.claude/init-project.sh
# Purpose: Initialize a new project with Dual AI Memory (Claude & Gemini)
# Usage: ./init-project.sh [project-name] [stack]

set -euo pipefail

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
mkdir -p "$PROJECT_KB_DIR"

cat > "$PROJECT_KB_DIR/11-tasks.md" << EOF
---
id: "${ACTUAL_PROJECT_NAME}-tasks"
type: project
status: active
tags: []
---
# Project Tasks

## Pending

- [ ] Initial setup complete

## In Progress

## Done

EOF

cat > "$PROJECT_KB_DIR/90-lessons.md" << EOF
---
id: "${ACTUAL_PROJECT_NAME}-lessons"
type: lesson
status: active
tags: []
---
# Lessons Learned

*Record mistakes and prevention rules here. Review at session start.*

## Patterns to Avoid

## Corrections Log

EOF

log_success "Created Knowledge Vault entries in 10_projects/${ACTUAL_PROJECT_NAME}"

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

log_success "Project initialized with Dual AI Configuration"
log_info "Structure:"
echo "  CLAUDE.md / GEMINI.md      - AI Memory files (Dual-Core)"
echo "  .claude/skills/            - Custom skills"
echo "  Knowledge Vault updated    - 11-tasks.md & 90-lessons.md in knowledge/10_projects/"