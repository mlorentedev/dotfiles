---
id: lesson-160-a-test-suite-must-test-the-tree-it-ships-in-not-th
type: lesson
status: active
created: "2026-08-07"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 160: A test suite must test the tree it ships in, not the tree it deployed to

**Context**: `scripts/test.sh` runs as the `dotfiles-test` pre-commit hook. It resolved its own location through an ambient `$DOTFILES_DIR` — the same variable the shell exports to point at the **deploy mirror** (`~/.dotfiles`).

**Problem**: The suite therefore asserted against the deployed copy rather than the tree being committed, which is the one thing a pre-commit hook exists to check. On Windows the mirror carries only a subset of `scripts/` (a single file, `load-secrets.ps1`), so the run died at step 2/15 with `utils.sh not found` and **blocked every commit on that machine**. CI never saw it, because `$DOTFILES_DIR` is unset there and the fallback happened to resolve correctly — which is exactly why it survived so long. Two different concepts had been sharing one variable: the tree under test, and the environment that was deployed to.

**Solution**: Split them. `REPO_DIR` is the tree under test and is always derived from the script's own location; `DOTFILES_DIR` is the deploy environment, read but never overwritten. `test.sh` is not itself a deployed artifact, so running it at all means running it from a checkout. Separately, POSIX-only assertions (symlink semantics, the `600` mode check) became explicit skips on Windows carrying the same reason the setup prints, rather than unconditional failures.

**Rule**: When a script can run both from a repo and from a deployment of that repo, "where am I" and "where did this get installed" are two questions, and one variable cannot answer both. A test that resolves its subject through ambient environment is testing whatever the environment last pointed at. Derive the tree under test from the script's own path, and treat any deploy-location variable as read-only input. Watch for the CI blind spot this creates: an unset variable in CI makes the fallback path the only one ever exercised, so the bug lives exclusively on developer machines.
