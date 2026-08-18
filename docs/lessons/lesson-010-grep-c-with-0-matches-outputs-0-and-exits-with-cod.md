---
id: lesson-010-grep-c-with-0-matches-outputs-0-and-exits-with-cod
type: lesson
status: active
created: "2026-02-28"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 010: grep -c with 0 matches outputs "0" AND exits with code 1

**Context**: Writing `dedup_current_date()` in `knowledge-crystallize.sh`. Used `count=$(grep -c 'pattern' "$file" 2>/dev/null || echo 0)` to count matching lines.

**Problem**: `grep -c` with zero matches prints `"0"` to stdout AND exits with code 1 (not found). The `|| echo 0` then fires, appending a SECOND `"0"` to stdout. Result: `count="0\n0"` — a multi-line string. Downstream `[ "$count" -le 1 ]` fails with "integer expression expected".

**Solution**: Use assignment outside the subshell: `count=$(grep -c 'pattern' "$file" 2>/dev/null) || count=0`. The `$()` captures grep's stdout (`"0"`). If grep exits 1, `|| count=0` overwrites the variable. Either way, `count` is a single clean integer.

**Rule**: Never use `|| echo VALUE` to provide a default for a command that BOTH outputs and exits non-zero on "empty" results (grep -c, wc, etc.). Use `command || var=default` (assignment form) to overwrite after the subshell, not append.
