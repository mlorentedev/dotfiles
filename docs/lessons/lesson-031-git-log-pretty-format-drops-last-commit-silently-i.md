---
id: lesson-031-git-log-pretty-format-drops-last-commit-silently-i
type: lesson
status: active
created: "2026-05-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 031: git log --pretty=format: drops last commit silently in while-read pipelines

**Context:** Building scripts/changelog-gen.sh to bucket commits by Conventional Commit type. Read `git log --no-merges --pretty=format:'%cs|%h|%s'` via `while IFS='|' read -r date hash subject; do ...; done < <(git log ...)`. All commits except the very oldest one made it into the output. Tests caught it: the assertion that "feat(core): initial commit" appeared under "## Features" failed.</context>
<parameter name="problem">`git log --pretty=format:'...'` writes the format string between commits but does NOT terminate the LAST commit with a newline (matches the docstring of `format:` in `git log` man page). When piped into a `while read` loop, `read` returns non-zero on the unterminated final line, so the loop exits BEFORE processing it. Result: the oldest commit is silently dropped. Easy to miss — the output looks correct, just one entry shorter.</problem>
<parameter name="solution">Use `--pretty=tformat:'...'` instead of `--pretty=format:'...'`. The `t` prefix means "terminator" — git appends a newline after every commit, including the last. The `while read` loop then sees the newline and processes the final entry. Alternatively, use the canonical idiom `while read || [[ -n "$line" ]]; do ...; done` to consume the unterminated line, but `tformat:` is cleaner because it fixes the data, not the consumer. Generalizes: any `while read` loop fed by a tool with optional trailing newlines (printf, awk, custom scripts) needs either the `|| [[ -n "$var" ]]` guard or the producer to always terminate. When in doubt, write a test that asserts the FIRST and LAST records make it through.</solution>
<parameter name="tags">["bash", "git", "shell-pipelines", "off-by-one", "test-discovery"]
