#!/usr/bin/env bats
# Regression guard for BUG-049 (#794): scripts/test.sh must assert against the
# tree it ships in, never the deploy mirror.
#
# Red-before-green: on the pre-fix script these tests fail, because an exported
# DOTFILES_DIR won and the suite reported "Testing from: <mirror>". That is the
# exact state that made the dotfiles-test pre-commit hook unpassable on Windows,
# where the mirror holds only a subset of scripts/.
#
# Only the resolution header is exercised, not the whole suite: the header is
# emitted before any assertion runs, so these stay fast and do not inherit the
# suite's environment-dependent results.

setup() {
    REPO_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    TMP="$(mktemp -d "${TMPDIR:-/tmp}/bats_testsh_XXXXXX")"

    # A decoy that looks like a deploy mirror but holds none of the sources the
    # suite asserts on -- i.e. exactly the Windows ~/.dotfiles shape.
    DECOY="$TMP/decoy-dotfiles"
    mkdir -p "$DECOY/scripts"
    : > "$DECOY/scripts/load-secrets.ps1"
}

teardown() {
    [ -z "${TMP:-}" ] || rm -rf "$TMP"
}

# Runs the suite only far enough to emit its resolution header.
header() {
    bash "$REPO_ROOT/scripts/test.sh" 2>&1 | grep -m1 '^Testing from:'
}

@test "resolves the repo root even when DOTFILES_DIR points elsewhere" {
    run env DOTFILES_DIR="$DECOY" bash -c "$(declare -f header); REPO_ROOT='$REPO_ROOT'; header"
    [ "$status" -eq 0 ]
    [[ "$output" == *"$REPO_ROOT"* ]]
}

@test "never reports the deploy mirror as the tree under test" {
    run env DOTFILES_DIR="$DECOY" bash -c "$(declare -f header); REPO_ROOT='$REPO_ROOT'; header"
    [[ "$output" != *"$DECOY"* ]]
}

@test "resolves the repo root when DOTFILES_DIR is unset" {
    run env -u DOTFILES_DIR bash -c "$(declare -f header); REPO_ROOT='$REPO_ROOT'; header"
    [ "$status" -eq 0 ]
    [[ "$output" == *"$REPO_ROOT"* ]]
}

@test "does not announce a custom DOTFILES_DIR as the test root" {
    run env DOTFILES_DIR="$DECOY" bash -c "bash '$REPO_ROOT/scripts/test.sh' 2>&1 | head -8"
    [[ "$output" != *"Using custom DOTFILES_DIR"* ]]
}
