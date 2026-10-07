---
id: "lesson-352-a-mid-test-double-bracket-assertion-is-a-no-op-under-bash-3-2"
type: lesson
status: active
title: "A mid-test [[ ]] assertion is a no-op under bash 3.2"
created: "2026-10-07"
---

# A mid-test [[ ]] assertion is a no-op under bash 3.2

## Context
#2055 added bats tests that source the real `.zshrc` in a scratch HOME. Mutating the rc to drop
the `*_HOME` directory guard, or the compinit fallback, should have failed the first test, and
on this Mac it did not.

## The Trap
`bats` resolves `/usr/bin/env bash`, which on a Mac without Homebrew bash is `/bin/bash` 3.2.
Before bash 4.1 a failing `[[ ]]` does not trip `set -e`, so only the last command of a test
decides it. Any `[[ ]]` before that is evaluated and ignored, while `[ ]` still fails. The test
reads as five assertions and checks one. Linux CI runs bash 5, so the same file is strict there
and lenient on the Mac. A green local run therefore says less than it appears to, and so does a
CI leg pinned to bash 3.2. #2096 counts 255 such assertions in 50 files.

## The Solution
End every `[[ ]]` that is not a test's last line with `|| false`, or use `[ ]` where no pattern
match is needed. When a mutation survives on one OS only, check which bash ran the test before
doubting the mutation. The class needs a guard that fails on the bare form, not a convention
(#2096).
