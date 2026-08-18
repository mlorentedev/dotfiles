---
id: lesson-006-claude-code-auto-memory-path-encoding
type: lesson
status: active
created: "2026-02-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 006: Claude Code auto-memory path encoding

**Context**: Needed to symlink auto-memory directories from dotfiles repo to `~/.claude/projects/*/memory/`

**Problem**: Claude Code encodes project paths by replacing all `/` with `-`. The path `/home/manu/Projects/dotfiles` becomes `-home-manu-Projects-dotfiles`. This encoding is deterministic but machine-specific (depends on username and directory structure).

**Solution**: Compute the encoded path dynamically in setup scripts: `printf '%s' "$project_path" | sed 's|/|-|g'`. Symlink from encoded path to vault directory (`~/Projects/knowledge/10_projects/<project>/memory/`).

**Rule**: When symlinking Claude Code internal paths, always compute the encoding at setup time — never hardcode the encoded path. Memory lives in the vault (not in the dotfiles repo) following the Neural Hive protocol: code in git, knowledge in vault.
