---
id: lesson-008-var-fallback-pattern-for-sourced-config-files
type: lesson
status: active
created: "2026-02-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 008: ${VAR:-fallback} pattern for sourced config files

**Context**: Tool versions (Java 21.0.4, Go 1.26.0, etc.) were hardcoded in both `.zshrc` and `.bashrc` — 12 duplicated strings with no single source of truth.

**Problem**: Updating a tool version required editing two files. A missed edit caused silent path mismatches. Also, on a fresh machine before running setup, `versions.conf` might not exist yet, and the shell would fail to set up PATH correctly.

**Solution**: Created `versions.conf` at repo root (KEY=VALUE, no export, no quotes — simultaneously sourceable by bash/zsh and parseable by PowerShell). Shell RCs source it with a guard (`[[ -f ... ]] && . ...`) then use `${JAVA_VERSION:-21.0.4}` to construct paths, providing a safe fallback if the file is missing.

**Rule**: When sourcing external config files that may not exist, always: (1) guard the source with a file-existence check, (2) use `${VAR:-default}` for every variable consumed from the config. This ensures the shell works on fresh machines before setup runs. The config file should use bare `KEY=VALUE` format (no `export`, no quotes) for maximum cross-tool compatibility.
