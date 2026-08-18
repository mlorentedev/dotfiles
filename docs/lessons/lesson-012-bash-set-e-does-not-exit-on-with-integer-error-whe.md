---
id: lesson-012-bash-set-e-does-not-exit-on-with-integer-error-whe
type: lesson
status: active
created: "2026-02-28"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 012: bash set -e does not exit on [ with integer error when in && chain

**Context**: In `knowledge-crystallize.sh`, `[ "$count" -le 1 ] && return 0` was at the top of `dedup_current_date`. When `count="0\n0"` (bug above), `[` exited with code 2 (error).

**Problem**: Expected `set -euo pipefail` to abort the script on the `[` error. Instead, the script continued. Bash's `set -e` doesn't always trigger on `[` exits with code 2 inside `&&` chains — the `&&` compound absorbs the error differently than a simple command.

**Solution**: Fixed the root cause (`grep -c` bug). Also added `|| log_warning "..."` in `run_all` around `process_project` so one bad project can't kill the batch run.

**Rule**: Don't rely on `set -e` to catch every `[` evaluation error — especially inside `&&`/`||` compounds. Validate that numeric variables are actually integers before using them in arithmetic comparisons. Wrap batch loops with per-item `|| warn` to isolate failures.
