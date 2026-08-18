---
id: lesson-118-in-bats-a-grep-q-guard-is-exempt-from-errexit-it-w
type: lesson
status: active
created: "2026-06-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 118: In bats, a `! grep -q` guard is exempt from errexit — it won't fail the test when the pattern is found

**Context**: A guard test asserting a retired token (`diff-check`) no longer appears in the production caller files.

**Problem**: Written as `! grep -qF 'diff-check' "$file"`, the line does NOT fail the test when the token IS present. POSIX `set -e` (which bats applies inside a test body) explicitly ignores the failure of a command/pipeline negated with `!`, so the negated grep's status never trips errexit and the test passes regardless — a guard that silently never guards.

**Solution**: Write the negative assertion explicitly: `if grep -qF 'diff-check' "$file"; then echo "still referenced in $file" >&2; return 1; fi`.

**Rule**: Never rely on a bare `! cmd` line as a failing assertion under errexit (bats, or any `set -e` script) — `!`-negated commands are exempt from errexit. Use an explicit `if cmd; then return 1; fi`, or `run cmd` + an `$status` check.
