#!/usr/bin/env bats
# Tests for age-encrypt-decrypt.sh

# bats file_tags=os-sensitive

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
}

# --- age-encrypt-decrypt.sh ---

@test "age-encrypt-decrypt.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/age-encrypt-decrypt.sh"
}

@test "age-encrypt-decrypt.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/age-encrypt-decrypt.sh"
}

@test "age-encrypt-decrypt.sh shows usage with no args" {
    run bash "$SCRIPTS_DIR/age-encrypt-decrypt.sh" 2>&1
    [[ "$output" == *"Usage"* ]] || false
}

@test "age-encrypt-decrypt.sh shows usage with invalid arg" {
    run bash "$SCRIPTS_DIR/age-encrypt-decrypt.sh" invalid 2>&1
    [[ "$output" == *"Usage"* ]] || false
}

@test "age-encrypt-decrypt.sh shows usage under zsh" {
    run zsh "$SCRIPTS_DIR/age-encrypt-decrypt.sh" 2>&1
    [[ "$output" == *"Usage"* ]] || false
}
