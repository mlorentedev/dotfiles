---
id: lesson-089-a-bats-teardown-s-last-command-classifies-even-ski
type: lesson
status: active
created: "2026-06-13"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 089: A bats teardown's last command classifies even *skipped* tests — never end it with a bare `[ cond ] && cmd`

**Context**: Six `tests/claude-mem-heal-ps1.bats` tests reported as `not ok N … # skip` locally (no `pwsh` installed), which reads as six failures. CI was green because the runner has `pwsh`, so the tests actually run instead of skipping. This failure-shaped skip had previously derailed reasoning about local suite health.

**Problem**: `setup()` calls `skip "pwsh not available"` *before* it assigns `TMP=$(mktemp -d)`. bats still runs `teardown()` after a skipped test, and the teardown's only line was `[ -n "${TMP:-}" ] && rm -rf "$TMP"`. With `TMP` unset, `[ -n "" ]` exits 1, the `&&` short-circuits, and teardown's *last* command therefore exits non-zero — so bats reclassifies the clean skip as `not ok # skip`. The same skip-fragile last-line pattern (`[ -n "$VAR" ] && rm -rf "$VAR"` as the final teardown statement) existed in ~8 test files; it only bites where a test can `skip` before the guarded var is set.

**Solution**: Invert the guard so both branches exit 0: `[ -z "${TMP:-}" ] || rm -rf "$TMP"` (empty → `[ -z ]` true, short-circuits at exit 0; set → runs `rm`, exit 0). Applied across the affected teardowns; the six heal-ps1 entries flip from `not ok # skip` to `ok # skip`.

**Rule**: A teardown's final command determines the test's exit classification — for passing, failing, *and* skipped tests alike. Never end a teardown with a bare `[ cond ] && cmd`; invert to `[ ! cond ] || cmd` or append an explicit `return 0`. A `skip` that fires before `setup()` finishes leaves cleanup vars unset, so the cleanup guard must not itself be able to fail.
