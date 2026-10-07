#!/usr/bin/env bats
# scripts/run-bats.sh when its environment is broken. The runner is what turns a
# missing dependency into an annotated failure instead of an empty or a bats-level
# error, so each case here builds the broken environment and asserts the message.
# The working-environment cases drive the real tools in tests/run-bats-real.bats.

# bats file_tags=os-sensitive

load 'lib/refute'

setup() {
    RUN="$BATS_TEST_DIRNAME/../scripts/run-bats.sh"
    BIN="$BATS_TEST_TMPDIR/bin"
    mkdir -p "$BIN"
}

# Links the real tools the script itself needs into $BIN, so a PATH of only $BIN
# is a working shell environment with exactly one thing missing.
link_basics() {
    local t
    for t in bash dirname head grep getconf; do
        ln -s "$(command -v "$t")" "$BIN/$t"
    done
}

@test "run-bats: a missing GNU parallel fails with the CI annotation, not a bats error" {
    link_basics
    run env PATH="$BIN" "$BIN/bash" "$RUN" --print-jobs
    [ "$status" -eq 1 ]
    [[ "$output" == "::error::run-bats: GNU parallel is missing"* ]]
}

@test "run-bats: a parallel that is not GNU parallel (moreutils) is refused by name" {
    link_basics
    printf '#!/bin/sh\necho "parallel from moreutils"\n' > "$BIN/parallel"
    chmod +x "$BIN/parallel"
    run env PATH="$BIN" "$BIN/bash" "$RUN" --print-jobs
    [ "$status" -eq 1 ]
    [[ "$output" == *"is not GNU parallel"* ]]
}

@test "run-bats: an unknown option is a usage error" {
    run "$RUN" --no-such-flag
    [ "$status" -eq 2 ]
}

@test "ci.yml counts CPUs through run-bats.sh, not nproc, which macOS does not have" {
    refute_grep '(\$\(|`)[[:space:]]*nproc' "$BATS_TEST_DIRNAME/../.github/workflows/ci.yml"
}
