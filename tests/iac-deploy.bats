#!/usr/bin/env bats
# SDD-007: Tests for deploy_file + check_deployed (IaC deploy strategy).
# Replaces ln -sf for config files. Asserts: atomic, idempotent, drift recovery.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    source "$SCRIPTS_DIR/utils.sh"

    # Per-test sandbox
    TEST_TMPDIR="$(mktemp -d "${BATS_TMPDIR:-/tmp}/iac-deploy.XXXXXX")"
    export SRC="$TEST_TMPDIR/src"
    export DST="$TEST_TMPDIR/dst"
    echo "original-content-v1" > "$SRC"
}

teardown() {
    rm -rf "$TEST_TMPDIR"
}

# --- deploy_file ---

@test "deploy_file: fresh deploy creates target with source content" {
    run deploy_file "$SRC" "$DST"
    [ "$status" -eq 0 ]
    [ -f "$DST" ]
    [ ! -L "$DST" ]
    cmp -s "$SRC" "$DST"
}

@test "deploy_file: second run re-deploys unconditionally (declarative always-overwrite)" {
    deploy_file "$SRC" "$DST"
    run deploy_file "$SRC" "$DST"
    [ "$status" -eq 0 ]
    # Aligned with the Windows always-overwrite strategy: every run guarantees
    # convergence and logs the deploy — there is no silent "already in sync"
    # skip path that could mask drift. Idempotency is of *outcome*, not output.
    [[ "$output" == *"Deployed"* ]]
    [ -f "$DST" ]
    [ ! -L "$DST" ]
    cmp -s "$SRC" "$DST"
}

@test "deploy_file: source change triggers re-deploy" {
    deploy_file "$SRC" "$DST"
    echo "modified-content-v2" > "$SRC"
    run deploy_file "$SRC" "$DST"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Deployed"* ]]
    cmp -s "$SRC" "$DST"
}

@test "deploy_file: missing source returns error" {
    run deploy_file "$TEST_TMPDIR/does-not-exist" "$DST"
    [ "$status" -eq 1 ]
    [[ "$output" == *"source missing"* ]]
    [ ! -f "$DST" ]
}

@test "deploy_file: drift recovery — removes pre-existing symlink at target" {
    # Simulate legacy state: ln -sf strategy left a symlink
    other="$TEST_TMPDIR/somewhere-else"
    echo "stale" > "$other"
    ln -sf "$other" "$DST"
    [ -L "$DST" ]

    run deploy_file "$SRC" "$DST"
    [ "$status" -eq 0 ]
    [ -f "$DST" ]
    [ ! -L "$DST" ]
    cmp -s "$SRC" "$DST"
}

@test "deploy_file: creates parent directory if missing" {
    nested="$TEST_TMPDIR/a/b/c/file"
    run deploy_file "$SRC" "$nested"
    [ "$status" -eq 0 ]
    [ -f "$nested" ]
}

# --- check_deployed ---

@test "check_deployed: matching content returns 0 (success)" {
    deploy_file "$SRC" "$DST"
    run check_deployed "$SRC" "$DST" "test-file"
    [ "$status" -eq 0 ]
    [[ "$output" == *"deployed (matches repo)"* ]]
}

@test "check_deployed: missing target returns 1 (drift)" {
    run check_deployed "$SRC" "$TEST_TMPDIR/never-deployed" "test-file"
    [ "$status" -eq 1 ]
    [[ "$output" == *"missing"* ]]
}

@test "check_deployed: symlinked target returns 1 (legacy-strategy drift)" {
    ln -sf "$SRC" "$DST"
    run check_deployed "$SRC" "$DST" "test-file"
    [ "$status" -eq 1 ]
    [[ "$output" == *"symlink"* ]]
}

@test "check_deployed: drift between repo and home returns 1" {
    deploy_file "$SRC" "$DST"
    echo "user-edited-in-home" > "$DST"
    run check_deployed "$SRC" "$DST" "test-file"
    [ "$status" -eq 1 ]
    [[ "$output" == *"drifted"* ]]
}
