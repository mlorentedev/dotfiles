---
id: lesson-003-dev-null-is-not-posix
type: lesson
status: active
created: "2025-12-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 003: &>/dev/null is not POSIX

**Context**: Scripts used `&>/dev/null` to suppress both stdout and stderr

**Problem**: `&>` is a bash-ism. Not POSIX-compliant and can fail in strict zsh configurations or when scripts are sourced in unexpected contexts.

**Solution**: Replace all `&>/dev/null` with `>/dev/null 2>&1` across every script.

**Rule**: Always use `>/dev/null 2>&1` for output suppression. Never use `&>`.
