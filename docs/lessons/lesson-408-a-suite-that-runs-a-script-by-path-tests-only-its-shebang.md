---
id: "lesson-408-a-suite-that-runs-a-script-by-path-tests-only-its-shebang"
type: lesson
status: active
title: "A suite that runs a script by path tests only its shebang"
created: "2026-10-13"
---

# A suite that runs a script by path tests only its shebang

## Context
Every script in the repo must run under bash and zsh (`.claude/CLAUDE.md`). `check-spec-gate.sh`
has three bats suites: `check-spec-gate.bats`, `spec-gate-archive.bats` and
`spec-gate-adjacency.bats`. All three called it as `run "$SCRIPTS_DIR/check-spec-gate.sh"`.

## The Trap
A script called by its path runs under its shebang, `#!/usr/bin/env bash`. So no case in those
suites ever ran it under zsh, and they were green while the script was broken there. Its code-span
stripping reads `BASH_REMATCH`, which zsh fills only under `setopt BASH_REMATCH`. Under
`set -u`, three archive-gate cases aborted with `BASH_REMATCH[0]: parameter not set`. Issue #1738
had measured 8 failures in September. #2169 fixed five of them, with nothing to say three were
left.

The knowledge gate and the PR-Agent push gate had already learned this. Each of their suites
carried its own inline copy of a run-under-both-shells helper. Copies are how the next suite
misses it: there was nothing to load, so the third suite never did.

## The Solution
- `tests/lib/both-shells.bash` provides `run_both_shells <script> [args...]`. It runs the script
  under zsh and then under bash, and fails when the two disagree on status or output. Seven suites
  load it instead of calling the script by path.
- `check-spec-gate.sh` gets the same zsh shim the knowledge gate has:
  `setopt KSH_ARRAYS BASH_REMATCH`.
- `tests/guard-zsh-bash-rematch.bats` fails any tracked `*.sh` or `*.bash` that reads
  `BASH_REMATCH` without that shim. `zsh -n` and shellcheck both pass such a file.

## Takeaways
- For a dual-shell script, `run "$script"` says nothing about zsh. Use `run_both_shells`.
- When a fix applies to every script, put it in a shared helper and a guard, not in one suite's
  local function.
