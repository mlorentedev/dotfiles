---
id: "lesson-313-a-smoke-test-that-runs-the-real-command-cannot-fail-and-can-do-damage"
type: lesson
status: active
title: "A smoke test that runs the real command cannot fail, and can do damage"
created: "2026-09-27"
owner: manu
tags: [lesson, testing, pre-commit, sync, secrets, blast-radius]
---

# A smoke test that runs the real command cannot fail, and can do damage

## What happened

CLI-036 (#1776) removed the 31 legacy `sensitive/*.secret.age` blobs from git. Minutes later, all 31 were back in the main checkout as untracked files, byte-identical to the retired ones.

Nobody restored them. The pre-commit hook "Run dotfiles tests" runs `scripts/test.sh`, and section 13 of that script executed the real `dotfiles-sync.sh --secrets-only` with no isolation. It used the real `$HOME`, so the real `~/.dotfiles` and the real `~/Projects/dotfiles` were in play. The sync copied `sensitive/` both ways, newest file wins, and the retired blobs still lived in `~/.dotfiles/sensitive/`: the deploy never prunes. Every commit, in any worktree, copied them back into the repo. The timestamps matched a peer session's two commits to the second.

The step could not fail either. Both branches of its `grep -q … && pass || pass` called `pass`. A check that passes on every outcome carries no information. This one had a side effect on the developer's real files instead.

## The rule

A test never executes a command against the real environment. Point `HOME` and every path the command defaults to at temporary directories, the way `tests/dotfiles-sync.bats` now does. A smoke test that needs the real environment is not a test. It is an operation, and it belongs to the operator.

A test with two branches must be able to take the failing one. If both branches pass, delete it: it hides a side effect behind a green line.

A two-way sync resurrects whatever one side has not pruned. Retiring a file means removing it from every copy a sync can reach, or removing the sync. #1795 removed the secrets half of `dotfiles-sync`: Bitwarden is the store (ADR-028), and git already tracks the one blob left.

Refs: BUG-106 (#1795), CLI-036 (#1776), BUG-105 (#1793), lesson 310.
