#!/usr/bin/env bats
# Tests for scripts/check-twin-test-retirement.sh — the guard that makes
# ADR-020 §5's "delete their bats/Pester tests IN THE SAME PR" a mechanism
# instead of a sentence.
#
# The guard is fixture-driven through --deleted-from so these run offline and
# without a git history: the deletion list is the input, and the working tree
# supplies "does the twin still exist at HEAD".

setup() {
    GUARD="$BATS_TEST_DIRNAME/../scripts/check-twin-test-retirement.sh"
    WORK="$BATS_TEST_TMPDIR/work"
    mkdir -p "$WORK/scripts" "$WORK/tests"
    cd "$WORK"
}

# deleted <path...>: write the deletion list the guard will read.
deleted() {
    printf '%s\n' "$@" > "$WORK/deleted.txt"
}

@test "a retired pair whose bats twin survives is reported" {
    deleted scripts/obs-cli.sh scripts/obs-cli.ps1
    : > tests/obs-cli.bats

    run "$GUARD" --deleted-from "$WORK/deleted.txt"
    [ "$status" -eq 1 ]
    [[ "$output" == *"tests/obs-cli.bats"* ]]
    [[ "$output" == *"ADR-020"* ]]
}

@test "a retired pair whose Pester twin survives is reported" {
    deleted scripts/obs-cli.sh scripts/obs-cli.ps1
    : > tests/obs-cli.Tests.ps1

    run "$GUARD" --deleted-from "$WORK/deleted.txt"
    [ "$status" -eq 1 ]
    [[ "$output" == *"tests/obs-cli.Tests.ps1"* ]]
}

@test "the -ps1 Pester spelling is covered too" {
    # The repo carries install-dotf-ps1.Tests.ps1 alongside plain
    # windows-defaults.Tests.ps1; matching only one spelling would let the other
    # through, and it is the one attached to the biggest remaining pair.
    deleted scripts/install-dotf.sh scripts/install-dotf.ps1
    : > tests/install-dotf-ps1.Tests.ps1

    run "$GUARD" --deleted-from "$WORK/deleted.txt"
    [ "$status" -eq 1 ]
    [[ "$output" == *"tests/install-dotf-ps1.Tests.ps1"* ]]
}

@test "a retirement that took its tests along passes" {
    deleted scripts/obs-cli.sh scripts/obs-cli.ps1 tests/obs-cli.bats tests/obs-cli.Tests.ps1
    # Nothing left in tests/ — this is what CLI-072 did by hand.

    run "$GUARD" --deleted-from "$WORK/deleted.txt"
    [ "$status" -eq 0 ]
    [[ "$output" == *"[OK]"* ]]
}

@test "deleting only the .ps1 half does not demand the bats file" {
    # A Windows-only cleanup is not a twin retirement. The surviving .sh still
    # needs its tests, and failing here would push someone to delete live
    # coverage to get green.
    deleted scripts/obs-cli.ps1
    : > scripts/obs-cli.sh
    : > tests/obs-cli.bats

    run "$GUARD" --deleted-from "$WORK/deleted.txt"
    [ "$status" -eq 0 ]
}

@test "a deletion outside scripts/ is ignored" {
    deleted docs/lessons/lesson-001-something.md
    : > tests/lesson-001-something.bats

    run "$GUARD" --deleted-from "$WORK/deleted.txt"
    [ "$status" -eq 0 ]
}

@test "an empty deletion list passes without touching the tree" {
    : > "$WORK/deleted.txt"
    : > tests/obs-cli.bats

    run "$GUARD" --deleted-from "$WORK/deleted.txt"
    [ "$status" -eq 0 ]
}

@test "both flags together, or neither, is a usage error not a silent pass" {
    # Exit 2, distinct from the exit 1 that means "a real offender". A guard
    # whose misuse looks like success is the failure mode this repo keeps
    # meeting.
    run "$GUARD"
    [ "$status" -eq 2 ]

    run "$GUARD" --base main --deleted-from "$WORK/deleted.txt"
    [ "$status" -eq 2 ]
}

@test "a missing deletion-list file is an error, never an empty list" {
    run "$GUARD" --deleted-from "$WORK/does-not-exist.txt"
    [ "$status" -eq 2 ]
}
