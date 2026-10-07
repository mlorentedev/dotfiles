---
id: "lesson-002-echo-e-breaks-in-zsh"
type: lesson
status: active
title: "echo -e breaks in zsh"
created: "2025-12-15"
owner: manu
tags: [lesson, dotfiles]
---

# echo -e breaks in zsh

**Context**: Shell scripts used `echo -e "\033[32mDone\033[0m"` for colored output

**Problem**: zsh does not support the `-e` flag on `echo`. Color codes printed as literal text instead of being interpreted.

**Solution**: Replace all `echo -e` with `printf '%b'` which handles escape sequences portably.

**Rule**: Never use `echo -e`. Always use `printf '%b' "..."` for colored or escaped output.
