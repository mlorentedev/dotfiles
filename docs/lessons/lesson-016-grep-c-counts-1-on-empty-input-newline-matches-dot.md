---
id: lesson-016-grep-c-counts-1-on-empty-input-newline-matches-dot
type: lesson
status: active
created: "2026-03-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 016: grep -c '.' counts 1 on empty input (newline matches dot)

**Context**: `vault-health.sh` counted orphan/dead-end links with `echo "$output" | grep -c '.'`. When output was empty, the count should be 0.

**Problem**: `echo ""` emits a newline character. The regex `.` matches a newline in this pipeline context, so `grep -c '.'` returns 1 instead of 0. Every vault health check reported at least 1 orphan even on a clean vault.

**Solution**: Changed all `grep -c '.'` to `grep -c '[^[:space:]]'` which only counts lines with visible characters.

**Rule**: When counting non-empty lines from command output, use `grep -c '[^[:space:]]'` not `grep -c '.'`. The dot regex matches newlines from `echo` even when the content is empty.
