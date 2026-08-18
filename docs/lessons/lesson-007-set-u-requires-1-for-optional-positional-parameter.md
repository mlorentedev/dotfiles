---
id: lesson-007-set-u-requires-1-for-optional-positional-parameter
type: lesson
status: active
created: "2026-02-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 007: set -u requires ${1:-} for optional positional parameters

**Context**: Adding `set -euo pipefail` to all standalone scripts. Several scripts used bare `$1` in argument parsing (e.g., `case "$1" in`, `[[ -z "$1" ]]`).

**Problem**: `set -u` treats unset variables as errors. When a script is invoked with no arguments, `$1` is unbound and the script aborts before reaching its usage message. Tests that checked "shows usage with no args" all failed.

**Solution**: Use `${1:-}` (default to empty string) wherever a positional parameter might be unset. For scripts with a `case "$1"` dispatch, assign to a local variable first: `ACTION="${1:-}"` then `case "$ACTION" in`.

**Rule**: When using `set -u`, always guard optional positional parameters with `${1:-}` or `${1:-default}`. The `$#` check (`[ $# -ne 2 ]`) before access is also safe. Only bare `$1` inside a `while [[ $# -gt 0 ]]` loop is inherently safe.
