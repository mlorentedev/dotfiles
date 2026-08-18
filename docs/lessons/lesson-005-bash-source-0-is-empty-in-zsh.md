---
id: lesson-005-bash-source-0-is-empty-in-zsh
type: lesson
status: active
created: "2025-12-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 005: ${BASH_SOURCE[0]} is empty in zsh

**Context**: Scripts used `${BASH_SOURCE[0]}` to determine their own file path for relative directory resolution

**Problem**: zsh does not populate `BASH_SOURCE`. Scripts that relied on it for path resolution silently got empty strings, breaking relative path calculations.

**Solution**: Use `${BASH_SOURCE[0]:-$0}` which falls back to `$0` (populated by zsh) when `BASH_SOURCE` is empty.

**Rule**: Always use `${BASH_SOURCE[0]:-$0}` when a script needs to know its own path. Test path resolution in both bash and zsh.
