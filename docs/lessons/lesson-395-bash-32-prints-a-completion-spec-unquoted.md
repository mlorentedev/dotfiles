---
id: "lesson-395-bash-32-prints-a-completion-spec-unquoted"
type: lesson
status: active
title: "bash 3.2 prints a completion spec unquoted, and an untagged test only meets it on main"
created: "2026-10-10"
---

# bash 3.2 prints a completion spec unquoted, and an untagged test only meets it on main

## Context
#2290 made `.bashrc` register terraform completion on whichever terraform PATH resolves, and
`tests/bashrc-guards.bats` asserted the result through `complete -p terraform`. The assertion
matched bash 5's output, `complete -o nospace -C '/path/terraform' terraform`.

## Problem
bash 3.2, which macOS ships as `/bin/bash` and the macOS CI leg runs bats under, prints the same
spec without the quotes: `-C /path/terraform`. The test failed on the macOS leg of every push to
main for 20 merges and nobody saw it. The pull request's own macOS run had passed, because a PR runs
only the files tagged `os-sensitive` and this file had no tag. The full suite runs only on a push
to main, and `test-macos` is not a required check yet (W10), so the red run blocked nothing.

## Solution
The assertion strips single quotes before it compares, so both shells' forms match. The file now
carries `# bats file_tags=os-sensitive`: it sources the rc under the host's bash, so whatever the rc
prints depends on the bash version. A change to it now meets bash 3.2 on its own pull request.

## Why
- **The output of a shell builtin is part of the shell version.** `complete -p`, `declare -p`,
  `alias` and `set` all quote differently across bash releases. An assertion on that output must
  normalise the quoting or pin the shell.
- **A tier filter turns an untagged test into a main-only test.** A non-required check that runs
  only on main reports to nobody. While `test-macos` is non-required, check its last main run
  before you call a macOS-facing change done.
