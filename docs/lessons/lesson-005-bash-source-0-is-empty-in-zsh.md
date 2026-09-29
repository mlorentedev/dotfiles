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

**Solution**: For file-backed scripts, use `${BASH_SOURCE[0]:-$0}` so zsh can fall back to `$0`. For scripts that may be streamed through stdin, do not use `$0` as a file path: it names the shell executable and can make the script source files from an attacker-controlled working directory. Treat an empty `BASH_SOURCE` as "no script directory" and use self-contained fallbacks.

**Rule**: Use `${BASH_SOURCE[0]:-$0}` only when the entry point is guaranteed to be file-backed. Stdin-capable entry points must use `${BASH_SOURCE[0]:-}` and explicitly handle the empty case. Test path resolution in bash, zsh, and raw-stdin execution.
