---
id: lesson-023-cp-fails-when-source-and-destination-resolve-to-th
type: lesson
status: active
created: "2026-03-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 023: cp fails when source and destination resolve to the same file via symlink

**Context**: `setup-linux.sh` used `safe_copy` (which wraps `cp`) to deploy `.gitconfig` from `$DOTFILES_DIR/.gitconfig` to `$HOME/.gitconfig`.

**Problem**: `~/.gitconfig` was already a symlink pointing to `~/.dotfiles/.gitconfig` (the same file). `cp` cannot copy a file over a symlink that resolves to the same source — it fails silently or with an error. This broke the "Setting up Git configuration" step on re-runs.

**Solution**: Replaced `safe_copy` with `ln -sf` for `.gitconfig`, consistent with how `.zshrc` and `.profile` are already deployed. `ln -sf` silently replaces any existing symlink. Updated `verify-setup.bats` to assert symlink behavior instead of regular file.

**Rule**: When deploying dotfiles, prefer `ln -sf` over `cp` for idempotency. A symlink can always be replaced atomically, while `cp` fails when the destination is a symlink to the source. Use the same deployment mechanism (symlink) for all managed dotfiles to avoid inconsistent behavior on re-runs.
