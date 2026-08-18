---
id: lesson-021-shellcheck-treats-shellcheck-comments-as-directive
type: lesson
status: active
created: "2026-03-16"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 021: ShellCheck treats "# shellcheck" comments as directives

**Context**: `setup-linux.sh` had a comment `# shellcheck (shell script linter)` describing the tool being installed.

**Problem**: ShellCheck parses any comment starting with `# shellcheck` as a directive (like `# shellcheck disable=SC2012`). The parenthesized description text is not valid directive syntax, causing SC1073/SC1072 parse errors that halt further checking of the entire file.

**Solution**: Capitalize or rephrase the comment to avoid the `# shellcheck` prefix, e.g., `# ShellCheck (shell script linter)` or `# Install shellcheck`.

**Rule**: Never start a comment with the literal text `# shellcheck` unless it is an actual ShellCheck directive. The tool intercepts any comment matching that prefix. Use capitalization (`# ShellCheck`) or different phrasing to describe the tool in human-readable comments.
