---
id: lesson-099-extracting-a-hook-function-into-a-sibling-script-c
type: lesson
status: active
created: "2026-06-16"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 099: Extracting a hook function into a sibling script can flip its exit status and silently kill the hook under `set -e`

**Context**: MEMORY-002 extracted the vault→memory symlink resolver out of `claude-session-start.sh`'s `ensure_memory_symlink` into a standalone agnostic `ensure-memory-symlink.sh`. The hook's function was rewritten to delegate: compute Claude's encoded target, call the script, append any message it printed.

**Problem**: The refactored function ended with `[ -n "$msg" ] && CONTEXT_LINES="…"`. When `$msg` is empty (the common steady state — no new symlink to create), that compound returns 1, so the *function* returns 1. It is invoked bare (`ensure_memory_symlink`) under the hook's `set -euo pipefail`, so a non-zero return **aborts the whole hook before it prints its JSON** — every Claude SessionStart would emit zero `additionalContext`. The original always returned 0 (it ended in an `if … fi`). Two suites disagreed: `session-start-false-positives.bats` copies only the hook, so the sibling is absent and an early `[ -x "$helper" ] || return 0` guard returns 0 — it passed and masked the bug; the `byte-equivalence` test runs from the real `scripts/` with the sibling present and caught zero-output-vs-full-output across 3 CWDs.

**Solution**: End the delegating function with an explicit `return 0` (plus `|| true` on the command substitution and a sibling-presence guard so the hook degrades cleanly when the script is not deployed). The byte-equivalence regression test — diffing the refactored script's stdout against `origin/main`'s across representative CWDs — is the oracle that a behaviour-preserving extraction stays byte-identical.

**Rule**: When extracting logic out of a `set -e` script into a function or sibling, audit the function's *last command's* exit status — a trailing `[ cond ] && action` returns 1 when the condition is false, and a bare call to that function then aborts the parent. Make best-effort helpers end in explicit `return 0`. And trust output-parity (byte-equivalence) tests over fixture tests that copy only a subset of the deploy tree: the subset can hide a newly-introduced sibling dependency that only bites when the full tree is present.
