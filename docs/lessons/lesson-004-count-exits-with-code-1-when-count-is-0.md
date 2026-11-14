---
id: "lesson-004-count-exits-with-code-1-when-count-is-0"
type: lesson
status: active
title: "((count++)) exits with code 1 when count is 0"
created: "2025-12-15"
owner: manu
tags: [lesson, dotfiles]
---

# ((count++)) exits with code 1 when count is 0

**Context**: Counter variables used `((count++))` inside scripts with `set -e`

**Problem**: In bash, `((0))` evaluates to false and returns exit code 1. When `count=0`, `((count++))` evaluates the pre-increment value (0), triggering `set -e` to abort the script.

**Solution**: Replace `((count++))` with `count=$((count + 1))`. The `$((...))` form always returns exit code 0.

**Rule**: Never use `((...))` for arithmetic that might evaluate to 0 under `set -e`. Use `var=$((expr))` assignment form instead.

**Only from bash 4.1 (measured 2026-11-14).** macOS `/bin/bash` 3.2.57 does not apply `set -e` to a failing `(( ))`: the same script prints its count and exits 0 there, and exits 1 under bash 5.3. A Mac-only check therefore passes a script that dies on Linux. The planted eval case `harness/evals/planted/bash-count-increment/` records both results.
