#!/usr/bin/env bats
# Tests for .github/hooks/validate-commit-msg.sh (already POSIX)

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export HOOK="$DOTFILES_DIR/.github/hooks/validate-commit-msg.sh"
}

@test "validate-commit-msg.sh valid sh syntax" {
    sh -n "$HOOK"
}

@test "validate-commit-msg.sh valid bash syntax" {
    bash -n "$HOOK"
}

@test "validate-commit-msg.sh valid zsh syntax" {
    zsh -n "$HOOK"
}

@test "accepts valid commit message" {
    local msg_file="/tmp/bats_commit_msg_$$"
    echo "feat: add new feature" > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    rm -f "$msg_file"
}

@test "rejects invalid commit message" {
    local msg_file="/tmp/bats_commit_msg_$$"
    echo "Invalid commit message" > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 1 ]]
    [[ "$output" == *"ERROR"* ]]
    rm -f "$msg_file"
}

@test "accepts valid message under zsh" {
    local msg_file="/tmp/bats_commit_msg_zsh_$$"
    echo "fix: resolve bug" > "$msg_file"
    run zsh "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    rm -f "$msg_file"
}

@test "rejects invalid message under zsh" {
    local msg_file="/tmp/bats_commit_msg_zsh_$$"
    echo "Bad message" > "$msg_file"
    run zsh "$HOOK" "$msg_file"
    [[ $status -eq 1 ]]
    rm -f "$msg_file"
}
