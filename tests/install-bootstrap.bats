#!/usr/bin/env bats
# Tests for IDEAS-005: curl bootstrap one-liner (Issue #1094)

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export INSTALL_SH="$DOTFILES_DIR/install.sh"
    TEST_TMPDIR="$(mktemp -d)"
}

teardown() {
    rm -rf "$TEST_TMPDIR"
}

@test "IDEAS-005: install.sh exists and is executable" {
    [ -f "$INSTALL_SH" ]
    [ -x "$INSTALL_SH" ]
}

@test "IDEAS-005: install.sh passes shellcheck" {
    if ! command -v shellcheck >/dev/null 2>&1; then
        skip "shellcheck not installed"
    fi
    shellcheck "$INSTALL_SH"
}

@test "IDEAS-005: install.sh has valid bash syntax" {
    bash -n "$INSTALL_SH"
}

@test "IDEAS-005: install.sh honors DOTFILES_DIR for fresh clone" {
    target="$TEST_TMPDIR/fresh-dotfiles"
    DOTFILES_DIR="$target" \
    DOTFILES_REPO="$BATS_TEST_DIRNAME/.." \
    DOTFILES_SKIP_SETUP=1 \
        bash "$INSTALL_SH"
    [ -d "$target/.git" ]
}

@test "IDEAS-005: install.sh updates existing clone (idempotent)" {
    target="$TEST_TMPDIR/existing-dotfiles"
    git clone "$DOTFILES_DIR" "$target"
    DOTFILES_DIR="$target" \
    DOTFILES_SKIP_SETUP=1 \
        bash "$INSTALL_SH"
    [ -d "$target/.git" ]
}

@test "IDEAS-005: install.sh fails if target exists but is not a git repo" {
    target="$TEST_TMPDIR/not-a-repo"
    mkdir -p "$target"
    run bash -c "DOTFILES_DIR='$target' DOTFILES_SKIP_SETUP=1 bash '$INSTALL_SH'"
    [ "$status" -ne 0 ]
    [[ "$output" == *"not a git repository"* ]]
}

@test "IDEAS-005: install.sh fails if git is missing" {
    # Create a minimal PATH with only bash (no git)
    stub="$TEST_TMPDIR/stub-bin"
    mkdir -p "$stub"
    ln -s "$(command -v bash)" "$stub/bash"
    run env PATH="$stub" DOTFILES_SKIP_SETUP=1 bash "$INSTALL_SH"
    [ "$status" -ne 0 ]
    [[ "$output" == *"git"* ]]
}

@test "IDEAS-005: install.sh skips setup when DOTFILES_SKIP_SETUP=1" {
    target="$TEST_TMPDIR/skip-setup"
    run bash -c "DOTFILES_DIR='$target' DOTFILES_REPO='$DOTFILES_DIR' DOTFILES_SKIP_SETUP=1 bash '$INSTALL_SH'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"DOTFILES_SKIP_SETUP=1"* ]]
}
