---
id: "lesson-417-bash-32-misparses-a-case-pattern-inside-command-substitution"
type: lesson
status: active
title: "bash 3.2 misparses a case pattern's ) inside $(...), and only main's macOS job could see it"
created: "2026-10-14"
---

# bash 3.2 misparses a case pattern's ) inside $(...), and only main's macOS job could see it

## Context
#2433 added a bats test that maps model ids to short names with `case "$model" in *"$short"*) … ;; esac` inside a `$(...)` pipeline. Every PR check passed, and the change merged. From then on, `test-macos` failed on every push to main: `syntax error near unexpected token ';;'`, then `setup_file failed`, so the whole file never ran.

## Problem
bash 3.2 (macOS `/bin/bash`) closes the command substitution at the pattern's unbalanced `)`. bash 4+ parses it correctly.

On a pull request, `test-macos` runs only the `os-sensitive` tier. The full suite under bash 3.2 runs only on a push to main. So no PR check ever parsed this file with bash 3.2, and the first run that could fail was a push to main.

## Solution
- Inside `$(...)`, use a test instead of `case`: `[ "${model#*"$short"}" = "$model" ] || echo …`. The POSIX form `(pattern)`, with a leading parenthesis, also balances.
- `test-macos` now has a step that runs on pull requests as well: every `tests/*.bats` must parse under `/bin/bash -n`, after a sed turns each `@test` header into a function header. It takes seconds, and it caught this file before the fix.

## Why
A tier split that saves CI minutes needs a cheap check that covers all files, or the expensive tier becomes the first place a whole class of failure shows up. A syntax check is that cheap check: it reads every file and runs none of them.
