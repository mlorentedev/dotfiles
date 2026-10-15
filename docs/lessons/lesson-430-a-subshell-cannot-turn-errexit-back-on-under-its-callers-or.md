---
id: "lesson-430-a-subshell-cannot-turn-errexit-back-on-under-its-callers-or"
type: lesson
status: active
title: "A subshell cannot turn errexit back on under its caller's ||"
created: "2026-10-14"
owner: manu
tags: [bash, errexit, bats, testing]
---

# A subshell cannot turn errexit back on under its caller's ||

**Context:** CI-012 N3 moved five bats files from a `git init` per test to a fixture built once in `setup_file` (`tests/lib/fixture-template.bash`). The helper runs a builder function and must fail when any step of it fails, since a half-built template would be copied into every test.

**Problem:** `( set -e; builder ) || rc=$?` does not stop at a failing step in the middle of the builder. Bash ignores errexit in everything under the left side of `||`, `&&`, `if` or `!`, and an explicit `set -e` inside the subshell does not undo that. Backgrounding the subshell and reading `wait $!` worked when the helper was called directly, but not when its caller was itself under `||`: the background subshell inherited the ignore. Bash 3.2 and 5.3 also disagree on which subshells inherit it. Under 3.2 a command substitution did not and a background job did, so no probe of the calling context gave the same answer on both.

**Solution:** run the builder in a new process: `"$BASH" -euc "$(declare -f "$2")"'; cd "$1"; "$2"' _ "$dir" "$2"`. A new bash cannot inherit the caller's context, so `-e` holds. `-u` holds too, so a variable the caller did not export fails instead of expanding to nothing. The builder gets only its own definition, which makes it a self-contained recipe. `tests/fixture-template-helper.bats` calls the build under `||`, with the failure placed mid-builder, and the subshell form of the helper fails that test.

**Why:** errexit is caller state, not a property of the code that sets it. Where a step must stop on its first failure whatever its caller looks like, a new process is the only boundary both bash versions respect. The test has to put the failure mid-function, because a failing last command is reported under any context.
