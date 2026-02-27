# Dotfiles Project Memory

## User Preferences
- **NEVER commit directly.** Only provide the commit message. The user handles `git add` and `git commit` themselves.

## Shell Script Patterns
- All scripts use `#!/bin/bash` or `#!/usr/bin/env bash` but are now hardened for zsh compat
- `echo -e` → `printf '%b'` for portable color output
- `&>/dev/null` → `>/dev/null 2>&1` everywhere
- `source` → `.` for POSIX sourcing
- `declare -g` → `eval` for global variable assignment
- `declare -a` unnecessary; `VAR=()` works in both bash and zsh
- `${BASH_SOURCE[0]}` → `${BASH_SOURCE[0]:-$0}` for zsh fallback
- `((count++))` is unsafe with `set -e` when count=0 → use `count=$((count + 1))`
- `${!var}` (bash indirect) already has zsh `${(P)var}` branches in utils.sh

## Test Infrastructure
- bats-core installed at `~/.local/bin/bats`
- shellcheck installed at `~/.local/bin/shellcheck`
- Test files: `tests/*.bats` (95 tests total, all pass bash+zsh)
- Run: `~/.local/bin/bats tests/*.bats`

## Key Files
- `scripts/utils.sh` - Foundation library, sourced by everything
- `scripts/load-secrets.sh` - Sourced in .zshrc/.bashrc, must work in both
- `SHELL-HARDENING.md` - Full before/after report of all changes
