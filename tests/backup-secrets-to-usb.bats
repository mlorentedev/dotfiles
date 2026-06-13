#!/usr/bin/env bats
# Tests for scripts/backup-secrets-to-usb.sh (TEST-001 / #128)
#
# NOTE: basic syntax + usage coverage for this script already lives in
# tests/age-scripts.bats (it shares the age/secrets backup family). This file
# adds the *failure-mode guard* coverage that age-scripts.bats does not have:
# the three validation gates that exit non-zero BEFORE any copy happens, so
# they are exercisable with no real USB, no real key, and no secret reads.
#
# We re-assert syntax here too so this file is self-contained as the canonical
# per-script suite, but the load-bearing additions are the guard tests.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export BACKUP_SCRIPT="$SCRIPTS_DIR/backup-secrets-to-usb.sh"
    TMP="$(mktemp -d)"
}

teardown() {
    [ -z "${TMP:-}" ] || rm -rf "$TMP"
}

# --- Syntax (1 & 2) ---

@test "backup-secrets-to-usb.sh valid bash syntax" {
    bash -n "$BACKUP_SCRIPT"
}

@test "backup-secrets-to-usb.sh valid zsh syntax" {
    if command -v zsh >/dev/null 2>&1; then
        zsh -n "$BACKUP_SCRIPT"
    else
        skip "zsh not available"
    fi
}

# --- Usage (3) ---

@test "backup-secrets-to-usb.sh with no args prints usage and exits 1" {
    run bash "$BACKUP_SCRIPT"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Usage:"* ]]
}

@test "backup-secrets-to-usb.sh usage documents the key.txt prerequisite" {
    run bash "$BACKUP_SCRIPT"
    [[ "$output" == *"key.txt must already exist in USB root"* ]]
}

# --- Failure-mode guards: the three validations, in order (4) ---
# Each gate exits non-zero before any cp/mkdir, so no real secrets, USB, or
# key material are touched. These are the seams age-scripts.bats omits.

@test "backup-secrets-to-usb.sh exits with error when USB path does not exist" {
    run bash "$BACKUP_SCRIPT" "$TMP/no-such-usb"
    [ "$status" -ne 0 ]
    [[ "$output" == *"USB path not found"* ]]
}

@test "backup-secrets-to-usb.sh exits with error when key.txt is missing on the USB" {
    # USB path exists but has no key.txt => second gate fires.
    usb="$TMP/usb"
    mkdir -p "$usb"
    run bash "$BACKUP_SCRIPT" "$usb"
    [ "$status" -ne 0 ]
    [[ "$output" == *"key.txt not found in USB"* ]]
}

@test "backup-secrets-to-usb.sh does not write anything to the USB when key.txt is missing" {
    usb="$TMP/usb"
    mkdir -p "$usb"
    run bash "$BACKUP_SCRIPT" "$usb"
    [ "$status" -ne 0 ]
    # The guard must fire before the secrets/ dir is created on the USB.
    [ ! -e "$usb/secrets" ]
    [ ! -e "$usb/age-standalone.sh" ]
}

@test "backup-secrets-to-usb.sh key.txt guard also fires under zsh" {
    if ! command -v zsh >/dev/null 2>&1; then
        skip "zsh not available"
    fi
    usb="$TMP/usb"
    mkdir -p "$usb"
    run zsh "$BACKUP_SCRIPT" "$usb"
    [ "$status" -ne 0 ]
    [[ "$output" == *"key.txt not found in USB"* ]]
}
