---
id: "lesson-363-an-assertion-sound-on-one-bash-is-vacuous-on-another"
type: lesson
status: active
title: "An assertion sound on one bash is vacuous on another"
created: "2026-10-07"
---

# An assertion sound on one bash is vacuous on another

## Context
bats fails a test on its first false assertion only through `set -e`. The suite wrote most of its
string checks as a bare `[[ $output == *x* ]]` on its own line, 668 of them. Linux CI runs bats
under bash 5, and the `test-macos` job runs the `os-sensitive` tier under macOS `/bin/bash` 3.2 on
purpose (#2146).

## The Trap
Before bash 4.1, a failing `[[ ]]` does not trip `set -e`. So on the macOS leg, every bare `[[ ]]`
that was not the last line of its `@test` could not fail it. The tier had 54 of them, and the job
reported 257 passing tests, more than it actually checked. Nothing looked wrong: the same files are
sound on Linux, and a vacuous pass is indistinguishable from a real one. It surfaced only because a
new test passed against the unfixed file it was written to catch.

## The Solution
Append `|| false` to every bare `[[ ]]`. That is the form the bats documentation gives for bash 3.2,
and a codemod did all 668 lines. `tests/guard-bats-dbracket.bats` rejects the bare form, pins its
detector to a hand-counted fixture, and runs the premise as a probe on whatever bash runs bats.
The tier still passed 257/257 afterwards, so the assertions had been true all along. The guard is
what keeps that from being luck.

A test leg exists to run under a different interpreter, and the assertion idiom has to hold there
too. When a leg deliberately changes the shell, measure the suite's failure path on that shell, not
just its pass count (#2164).
