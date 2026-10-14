---
id: "lesson-423-every-zsh-reads-zshenv-so-a-test-path-is-not-the-script-path"
type: lesson
status: active
title: "Every zsh reads ~/.zshenv, so a test's PATH is not the script's PATH"
created: "2026-10-14"
---

# Every zsh reads ~/.zshenv, so a test's PATH is not the script's PATH

## Context
`run_both_shells` (tests/lib/both-shells.bash) runs a script under zsh and bash and fails when they disagree. The `check-doc-paths` test for #1043 sets `PATH=/usr/bin:/bin` so that `dotf` cannot be found and the vault checks are skipped.

## Problem
On the owner's Mac the test failed with `bash (0) and zsh (1) disagree`, on main and on every branch, while CI stayed green. zsh reads `~/.zshenv` for every script it runs, interactive or not. The deployed `.zshenv` sources `path.sh`, which puts mise's shims and `~/.local/bin` ahead of PATH, so zsh found `dotf` anyway, resolved the real vault and reported a missing path. CI runners have no `~/.zshenv`, so only a configured machine saw it. The comparison measured the user's startup files, not the script.

## Solution
`run_both_shells` runs `zsh -f` (NO_RCS), so zsh reads no user startup file and both shells get the environment the test set. `tests/both-shells-lib.bats` holds it: a scratch HOME whose `.zshenv` prepends to PATH must not change the PATH zsh reports.

## Why
A test that controls a variable through the environment controls it only for the shells that do not rewrite it at startup. bash reads nothing for a script unless BASH_ENV is set. zsh always reads `$ZDOTDIR/.zshenv`, and a dotfiles repo deploys exactly that file. A suite that passes in CI and fails on the developer's machine is often reading the machine, not the code.
