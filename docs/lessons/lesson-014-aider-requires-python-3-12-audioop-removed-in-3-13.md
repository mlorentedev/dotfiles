---
id: lesson-014-aider-requires-python-3-12-audioop-removed-in-3-13
type: lesson
status: active
created: "2026-03-10"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 014: Aider requires Python 3.12 — audioop removed in 3.13

**Context**: Installing aider-chat via `uv tool install` on a system with Python 3.13.

**Problem**: Python 3.13 removed the `audioop` stdlib module. Aider depends on `pydub` which imports `audioop` at module load time. The error manifests as `ModuleNotFoundError: No module named 'audioop'` on any aider command, even `--version`.

**Solution**: Pin Python 3.12 in the install command: `uv tool install --python 3.12 aider-chat`. Both `setup-linux.sh` and `setup-windows.ps1` use this pinned version.

**Rule**: When installing Python tools that depend on deprecated/removed stdlib modules, always pin the Python version explicitly in `uv tool install --python X.Y`. Check release notes for stdlib removals before upgrading the pinned version.
