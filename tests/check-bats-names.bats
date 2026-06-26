#!/usr/bin/env bats
# Tests for scripts/check-bats-names.sh -- guards against bats @test names that
# the runner silently fails to register (non-ASCII chars, or duplicate names),
# which yields a "green" test that never ran. Detected by CURATOR-001 (#615).

setup() {
    SCRIPT="$BATS_TEST_DIRNAME/../scripts/check-bats-names.sh"
    TMP="$(mktemp -d)"
}
teardown() { rm -rf "$TMP"; }

@test "check-bats-names: exits 2 on no args" {
    run "$SCRIPT"
    [ "$status" -eq 2 ]
}

@test "check-bats-names: passes a clean bats file (exit 0)" {
    printf '@test "plain ascii name" {\n  true\n}\n' > "$TMP/clean.bats"
    run "$SCRIPT" "$TMP/clean.bats"
    [ "$status" -eq 0 ]
}

@test "check-bats-names: flags a non-ASCII @test name with file:line" {
    printf '@test "bad \xe2\x80\x94 em-dash" {\n  true\n}\n' > "$TMP/bad.bats"
    run "$SCRIPT" "$TMP/bad.bats"
    [ "$status" -eq 1 ]
    [[ "$output" == *"bad.bats:1"* ]]
    [[ "$output" == *"non-ASCII"* ]]
}

@test "check-bats-names: flags the <= glyph too (not just em-dash)" {
    printf '@test "pointer is \xe2\x89\xa4 100 lines" {\n  true\n}\n' > "$TMP/le.bats"
    run "$SCRIPT" "$TMP/le.bats"
    [ "$status" -eq 1 ]
    [[ "$output" == *"non-ASCII"* ]]
}

@test "check-bats-names: flags duplicate @test names within a file" {
    printf '@test "dup name" {\n  true\n}\n@test "dup name" {\n  false\n}\n' > "$TMP/dup.bats"
    run "$SCRIPT" "$TMP/dup.bats"
    [ "$status" -eq 1 ]
    [[ "$output" == *"duplicate"* ]]
}

@test "check-bats-names: scans a directory recursively for *.bats" {
    mkdir -p "$TMP/sub"
    printf '@test "ok" {\n  true\n}\n' > "$TMP/sub/ok.bats"
    printf '@test "bad \xe2\x80\x94 dash" {\n  true\n}\n' > "$TMP/sub/bad.bats"
    run "$SCRIPT" "$TMP"
    [ "$status" -eq 1 ]
    [[ "$output" == *"bad.bats"* ]]
}

@test "check-bats-names: the repo's own tests/ pass (no silent-skip names remain)" {
    run "$SCRIPT" "$BATS_TEST_DIRNAME"
    [ "$status" -eq 0 ]
}
