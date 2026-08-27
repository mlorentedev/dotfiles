# Lesson 239 — Re-measure a filed bug on the current toolchain before implementing its fix

**Date:** 2026-08-27
**Context:** #912 (BUG-069, `core.hooksPath` in `C:/` form breaks hook execution) and #914 (WIN-006, hooks resolve WSL's bash) — both open, both claiming `git commit` fails on Windows.
**Category:** git, windows, diagnosis, tickets

## What happened

Both issues were filed with care. #912 carried a 2×2 evidence table (EOL ×
path form) measured on git-for-windows 2.53.0, isolating the `C:/` hooksPath
value as the cause: MSYS bash could not open the hook at a drive path. #914
proposed a different mechanism — `#!/usr/bin/env bash` resolving to WSL's
`bash.exe` because `C:\Windows\System32` precedes Git on PATH.

On the work box today, git is 2.55.0.windows.5, and `git commit` from
PowerShell exits 0 with the same `C:/…/git-hooks` value, System32 still first on
PATH. `GIT_TRACE=1` shows git starting all three hooks by their `C:/` path with
its bundled bash, ignoring the shebang; a repo-local `commit-msg` exiting 1
blocks the commit through the global→local chain. An isolated probe shows MSYS
bash on 2.55 opening a `C:/` path with rc=0 — the exact call that returned rc=1
on 2.53.

So #912's diagnosis was correct and its fix was shipped upstream, by
git-for-windows, between the two measurements. #914's mechanism never applied:
git does not consult the shebang when it invokes a hook.

## The lesson

A correct diagnosis has a toolchain version attached to it, whether or not the
ticket says so. Before implementing a filed bug's fix, reproduce it on what is
installed *now*; an upstream fix invalidates a correct ticket silently, and the
workaround it would have earned (`/c/` MSYS-form rewriting) becomes a
maintenance liability with no remaining cause.

What survives such a bug is not the workaround but the **floor**: the minimum
toolchain version that carries the upstream fix, declared where the repo
declares versions, and a guard that asserts the behaviour by effect rather than
by string-comparing a config value.

## Disposition

#914 closed as a misdiagnosed duplicate. #912 re-scoped to its acceptance
criterion 3 — a declared git-for-windows version floor plus the by-effect guard
— and explicitly not to the `/c/` rewrite.
