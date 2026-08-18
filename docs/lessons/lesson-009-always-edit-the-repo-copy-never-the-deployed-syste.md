---
id: lesson-009-always-edit-the-repo-copy-never-the-deployed-syste
type: lesson
status: active
created: "2026-02-27"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 009: Always edit the repo copy, never the deployed system copy

**Context**: Adding an `obsidian --no-sandbox` alias to `.zshrc` and `.bashrc`

**Problem**: Edited `~/.dotfiles/.zshrc` and `~/.dotfiles/.bashrc` (the deployed system copy) first, instead of `~/Projects/dotfiles/.zshrc` and `~/Projects/dotfiles/.bashrc` (the canonical repo). Changes in `~/.dotfiles/` are not tracked by git and will be overwritten on next sync.

**Solution**: Always edit files in `~/Projects/dotfiles/` (the repo), then commit there. The sync/install script deploys repo → `~/.dotfiles/`.

**Rule**: The canonical source of truth for dotfiles is `~/Projects/dotfiles/`. The `~/.dotfiles/` directory is a deployment target — never edit it directly. Flow: edit repo → commit → sync to system.
