---
id: lesson-004-count-exits-with-code-1-when-count-is-0
type: lesson
status: active
created: "2025-12-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 004: ((count++)) exits with code 1 when count is 0

**Context**: Counter variables used `((count++))` inside scripts with `set -e`

**Problem**: In bash, `((0))` evaluates to false and returns exit code 1. When `count=0`, `((count++))` evaluates the pre-increment value (0), triggering `set -e` to abort the script.

**Solution**: Replace `((count++))` with `count=$((count + 1))`. The `$((...))` form always returns exit code 0.

**Rule**: Never use `((...))` for arithmetic that might evaluate to 0 under `set -e`. Use `var=$((expr))` assignment form instead.
