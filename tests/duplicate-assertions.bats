#!/usr/bin/env bats
# CI-012 (#2059) N6: one behaviour, one test. Two tests grepping the same tracked
# file for the same pattern assert the same thing twice: a change to that line
# breaks both, and neither says more than the other. The 2026-10-07 audit found
# 63 such assertions; this keeps the count at zero.
#
# The scan lives in tests/lib/check-duplicate-assertions.py, which states the
# definition. Cases below run it on a fixture tree, so each one is about the rule
# and not about whatever the repo holds today.

setup() {
    REPO_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    SCAN="$REPO_ROOT/tests/lib/check-duplicate-assertions.py"
    FX="$BATS_TEST_TMPDIR/fx"
    mkdir -p "$FX/tests" "$FX/scripts"
    printf 'alpha\nbeta\n' > "$FX/scripts/a.sh"
    printf 'alpha\n' > "$FX/scripts/b.sh"
}

# _test <file> <name> <body>: append one @test to a fixture bats file.
_test() {
    printf '@test "%s" {\n%s\n}\n\n' "$2" "$3" >> "$FX/tests/$1.bats"
}

_head() {
    # The variable shapes the real files use to name what they read.
    printf 'REPO="$BATS_TEST_DIRNAME/.."\nSCRIPT="$REPO/scripts/a.sh"\n\n' > "$FX/tests/$1.bats"
}

@test "the repo has no duplicate static assertion" {
    run python3 "$SCAN" "$REPO_ROOT"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
    # A parser that matched nothing would also report no duplicates. The repo
    # holds hundreds of static assertions; reading fewer means the scan broke.
    n="$(printf '%s\n' "$output" | sed -n 's/^scanned \([0-9]*\) static assertions.*/\1/p')"
    [ -n "$n" ] && [ "$n" -ge 500 ] || { printf 'scanned too few: %s\n' "$output"; false; }
}

@test "two tests grepping one tracked file for one pattern fail, naming both" {
    _head one
    _test one "first" "    grep -q 'alpha' \"\$SCRIPT\""
    _test one "second" "    grep -qE 'alpha' \"\$REPO/scripts/a.sh\""
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 1 ]
    [[ "$output" == *"tests/one.bats:5"* ]] || false
    [[ "$output" == *"tests/one.bats:9"* ]] || false
    [[ "$output" == *"scripts/a.sh"* ]] || false
}

@test "a duplicate across two files is found" {
    _head one
    _head two
    _test one "first" "    grep -q 'beta' \"\$SCRIPT\""
    _test two "second" "    grep -q 'beta' \"\$BATS_TEST_DIRNAME/../scripts/a.sh\""
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 1 ]
    [[ "$output" == *"tests/one.bats"* ]] || false
    [[ "$output" == *"tests/two.bats"* ]] || false
}

@test "the same pattern on a file a test writes is not a duplicate" {
    _head one
    _test one "first" "    grep -q 'alpha' \"\$BATS_TEST_TMPDIR/out\""
    _test one "second" "    grep -q 'alpha' \"\$BATS_TEST_TMPDIR/out\""
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
}

@test "the same pattern on two different tracked files is not a duplicate" {
    _head one
    _test one "first" "    grep -q 'alpha' \"\$REPO/scripts/a.sh\""
    _test one "second" "    grep -q 'alpha' \"\$REPO/scripts/b.sh\""
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
}

@test "an assertion and its refutation are two different claims" {
    _head one
    _test one "present" "    grep -q 'alpha' \"\$SCRIPT\""
    _test one "absent" "    refute_grep 'alpha' \"\$SCRIPT\""
    _test one "absent again" "    ! grep -q 'beta' \"\$SCRIPT\""
    _test one "present again" "    grep -q 'beta' \"\$SCRIPT\""
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
}

@test "a refutation repeated in two tests is a duplicate" {
    _head one
    _test one "first" "    refute_grep 'gamma' \"\$SCRIPT\""
    _test one "second" "    ! grep -q 'gamma' \"\$SCRIPT\""
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 1 ]
}

@test "one test asserting a line twice is not a duplicate across tests" {
    _head one
    _test one "only" "    grep -q 'alpha' \"\$SCRIPT\"
    grep -q 'alpha' \"\$SCRIPT\""
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
}

@test "a dup-ok marker with a reason exempts the line" {
    _head one
    _test one "first" "    grep -q 'alpha' \"\$SCRIPT\""
    _test one "second" "    grep -q 'alpha' \"\$SCRIPT\"  # dup-ok: the second entry point reads the same line"
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
}

@test "a dup-ok marker without a reason fails" {
    _head one
    _test one "first" "    grep -q 'alpha' \"\$SCRIPT\""
    _test one "second" "    grep -q 'alpha' \"\$SCRIPT\"  # dup-ok:"
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 1 ]
    [[ "$output" == *"dup-ok"*"reason"* ]] || false
}

@test "a root without a tests directory is a usage error, not a clean result" {
    run python3 "$SCAN" "$BATS_TEST_TMPDIR/nowhere"
    [ "$status" -eq 2 ]
}

@test "a pipeline is one claim: the same pipeline twice is a duplicate" {
    _head one
    _test one "first" "    grep -B1 'beta' \"\$SCRIPT\" | grep -qx 'alpha'"
    _test one "second" "    grep -B1 'beta' \"\$SCRIPT\" | grep -qx 'alpha'"
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 1 ]
    [[ "$output" == *"| grep -qx alpha"* ]] || false
}

@test "pipelines that select the same line for different checks are different claims" {
    _head one
    _test one "before" "    grep -B1 'beta' \"\$SCRIPT\" | grep -qx 'alpha'"
    _test one "after" "    grep -A1 'beta' \"\$SCRIPT\" | grep -qx 'alpha'"
    _test one "plain" "    grep -q 'beta' \"\$SCRIPT\""
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
}

@test "a grep that extracts a value, filters or decides a branch is not an assertion" {
    _head one
    _test one "extract one" "    n=\$(grep -n 'alpha' \"\$SCRIPT\" | cut -d: -f1)
    [ \"\$n\" -eq 1 ]"
    _test one "extract two" "    n=\$(grep -n 'alpha' \"\$SCRIPT\" | cut -d: -f1)
    [ \"\$n\" -lt 2 ]"
    _test one "filter one" "    grep -v 'beta' \"\$SCRIPT\""
    _test one "filter two" "    grep -v 'beta' \"\$SCRIPT\""
    _test one "branch one" "    if grep -q 'beta' \"\$SCRIPT\"; then false; fi"
    _test one "branch two" "    if grep -q 'beta' \"\$SCRIPT\"; then false; fi"
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
}

@test "a variable a test reassigns resolves to that test's value" {
    _head one
    _test one "reads b" "    SCRIPT=\"\$REPO/scripts/b.sh\"
    grep -q 'alpha' \"\$SCRIPT\""
    _test one "reads a" "    grep -q 'alpha' \"\$SCRIPT\""
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
    _test one "reads b again" "    local SCRIPT=\"\$REPO/scripts/b.sh\"
    grep -q 'alpha' \"\$SCRIPT\""
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 1 ]
    [[ "$output" == *"scripts/b.sh: 'alpha' asserted present by 2 tests"* ]] || false
}

@test "a pattern after -- is read as the pattern, not a flag" {
    _head one
    _test one "first" "    grep -q -- '--flag' \"\$SCRIPT\""
    _test one "second" "    grep -qF -- '--flag' \"\$SCRIPT\""
    run python3 "$SCAN" "$FX"
    [ "$status" -eq 1 ]
    [[ "$output" == *"'--flag' asserted present by 2 tests"* ]] || false
}
