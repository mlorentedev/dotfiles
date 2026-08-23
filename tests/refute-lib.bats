#!/usr/bin/env bats
# Tests for tests/lib/refute.bash — the negative assertions the whole suite
# leans on (#1034).
#
# A broken assertion helper is worse than no helper: every call site that uses
# it reports green.  The two cases that matter most here are the ones a plain
# `! grep` gets wrong — a grep ERROR and a pattern that starts with `-`.

load 'lib/refute'

setup() {
    TMP="$(mktemp -d)"
    FIXTURE="$TMP/subject.txt"
    printf '%s\n' \
        'alpha' \
        'beta = 2' \
        '--vault knowledge --vault knowledge' \
        'a (parenthesised) phrase' \
        > "$FIXTURE"
}

teardown() { rm -rf "$TMP"; }

@test "refute_grep: passes when the extended regex is absent" {
    refute_grep '^gamma$' "$FIXTURE"
}

@test "refute_grep: fails when the extended regex is present" {
    run refute_grep '^beta = [0-9]+$' "$FIXTURE"
    [ "$status" -eq 1 ]
}

@test "refute_grep: names the file and prints the offending line" {
    run refute_grep '^beta' "$FIXTURE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"expected NOT to find"* ]]
    [[ "$output" == *"subject.txt"* ]]
    [[ "$output" == *"2:beta = 2"* ]]
}

@test "refute_grep: a pattern beginning with a dash is a pattern, not options" {
    run refute_grep '\-\-vault knowledge \-\-vault knowledge' "$FIXTURE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"but it is there"* ]]
}

@test "refute_grep_fixed: a literal dash-leading string is a pattern too" {
    run refute_grep_fixed '--vault knowledge --vault knowledge' "$FIXTURE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"but it is there"* ]]
}

@test "refute_grep_fixed: parentheses are literal, not a group" {
    run refute_grep_fixed '(parenthesised)' "$FIXTURE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"but it is there"* ]]
}

@test "refute_grep: a grep error is a failure, not an absence" {
    # The vacuous pass this whole change exists to kill: an unreadable file
    # exits 2, and `! grep` reads that as "not found".
    run refute_grep 'anything' "$TMP/does-not-exist.txt"
    [ "$status" -eq 1 ]
    [[ "$output" == *"an error is not an absence"* ]]
}

@test "refute_grep: an invalid extended regex is a failure, not an absence" {
    # A BRE pattern moved to -E without thought: `(` is a literal in BRE and an
    # unterminated group in ERE.  grep exits 2, and the assertion must not pass.
    run refute_grep 'a (parenthesised' "$FIXTURE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"an error is not an absence"* ]]
}

@test "refute_grep: a wrong argument count fails instead of half-asserting" {
    run refute_grep '^alpha$'
    [ "$status" -eq 1 ]
    [[ "$output" == *"expected <pattern> <file>"* ]]
}
