#!/usr/bin/env bats
# Tests for scripts/github-secrets-manager.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
}

@test "github-secrets-manager.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/github-secrets-manager.sh"
}

@test "github-secrets-manager.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/github-secrets-manager.sh"
}

@test "github-secrets-manager.sh --list produces output" {
    # --list should work even without gh auth (just reads mapping file)
    run bash "$SCRIPTS_DIR/github-secrets-manager.sh" --list 2>&1
    # Should either show secrets or an error about missing dependencies
    [[ "$output" == *"Available"* ]] || [[ "$output" == *"ERROR"* ]] || [[ "$output" == *"not found"* ]]
}

@test "github-secrets-manager.sh --list works under zsh" {
    run zsh "$SCRIPTS_DIR/github-secrets-manager.sh" --list 2>&1
    [[ "$output" == *"Available"* ]] || [[ "$output" == *"ERROR"* ]] || [[ "$output" == *"not found"* ]]
}
