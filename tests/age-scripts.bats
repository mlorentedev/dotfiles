#!/usr/bin/env bats
# Tests for age-encrypt-decrypt.sh and age-standalone.sh

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
    [[ "$output" == *"Usage"* ]]
}

@test "age-encrypt-decrypt.sh shows usage with invalid arg" {
    run bash "$SCRIPTS_DIR/age-encrypt-decrypt.sh" invalid 2>&1
    [[ "$output" == *"Usage"* ]]
}

@test "age-encrypt-decrypt.sh shows usage under zsh" {
    run zsh "$SCRIPTS_DIR/age-encrypt-decrypt.sh" 2>&1
    [[ "$output" == *"Usage"* ]]
}

# --- age-standalone.sh ---

@test "age-standalone.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/age-standalone.sh"
}

@test "age-standalone.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/age-standalone.sh"
}

@test "age-standalone.sh shows usage with no args" {
    run bash "$SCRIPTS_DIR/age-standalone.sh" 2>&1
    [[ "$output" == *"Usage"* ]]
}

@test "age-standalone.sh shows usage with invalid arg" {
    run bash "$SCRIPTS_DIR/age-standalone.sh" invalid 2>&1
    [[ "$output" == *"Usage"* ]]
}

@test "age-standalone.sh shows usage under zsh" {
    run zsh "$SCRIPTS_DIR/age-standalone.sh" 2>&1
    [[ "$output" == *"Usage"* ]]
}

# --- backup-secrets-to-usb.sh ---

@test "backup-secrets-to-usb.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/backup-secrets-to-usb.sh"
}

@test "backup-secrets-to-usb.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/backup-secrets-to-usb.sh"
}

@test "backup-secrets-to-usb.sh shows usage with no args" {
    run bash "$SCRIPTS_DIR/backup-secrets-to-usb.sh" 2>&1
    [[ "$output" == *"Usage"* ]]
}

@test "backup-secrets-to-usb.sh shows usage under zsh" {
    run zsh "$SCRIPTS_DIR/backup-secrets-to-usb.sh" 2>&1
    [[ "$output" == *"Usage"* ]]
}
