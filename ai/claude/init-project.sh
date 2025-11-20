#!/bin/bash
# ~/.claude/init-project.sh

PROJECT_NAME=$1
STACK=${2:-python}

mkdir -p $PROJECT_NAME && cd $PROJECT_NAME

# Estructura base
mkdir -p src tests docs scripts .github/workflows

# Archivos esenciales
cp ~/.claude/CLAUDE.md .
cp ~/.claude/settings.json .claude/

# Stack-specific
case $STACK in
  python)
    poetry init -n
    poetry add typer rich pydantic pytest pytest-cov mypy ruff
    ;;
  go)
    go mod init $PROJECT_NAME
    touch Makefile
    ;;
  node)
    npm init -y
    npm i -D typescript @types/node tsx vitest
    ;;
esac

# Git
git init
git add .
git commit -m "feat: initial commit with Claude optimization"

claude-code analyze . --suggest-structure