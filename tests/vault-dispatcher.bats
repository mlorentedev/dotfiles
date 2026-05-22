#!/usr/bin/env bats
# Tests for REFACTOR-005 vault.sh thin dispatcher.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export VAULT_SCRIPT="$DOTFILES_DIR/scripts/vault.sh"
    export HEALTH_SCRIPT="$DOTFILES_DIR/scripts/vault-health.sh"
    export MAINT_SCRIPT="$DOTFILES_DIR/scripts/vault-maintenance-weekly.sh"
    export ESCAPES_SCRIPT="$DOTFILES_DIR/scripts/check-md-escapes.sh"
}

@test "vault.sh exists" {
    [ -f "$VAULT_SCRIPT" ]
}

@test "vault.sh is executable" {
    [ -x "$VAULT_SCRIPT" ]
}

@test "vault.sh passes shellcheck" {
    if command -v shellcheck >/dev/null 2>&1; then
        shellcheck "$VAULT_SCRIPT"
    elif [ -x "$HOME/.local/bin/shellcheck" ]; then
        "$HOME/.local/bin/shellcheck" "$VAULT_SCRIPT"
    else
        skip "shellcheck not available"
    fi
}

@test "vault.sh valid bash syntax" {
    bash -n "$VAULT_SCRIPT"
}

@test "vault.sh valid zsh syntax" {
    if command -v zsh >/dev/null 2>&1; then
        zsh -n "$VAULT_SCRIPT"
    else
        skip "zsh not available"
    fi
}

# --- Dispatch correctness (asserted via grep on the source, not by execing
# the backing scripts which require Obsidian / cron / fixture vaults) ---

@test "vault.sh dispatches 'health' to vault-health.sh" {
    grep -qE '^\s*health\)' "$VAULT_SCRIPT"
    grep -qE 'exec "\$SCRIPT_DIR/vault-health.sh"' "$VAULT_SCRIPT"
}

@test "vault.sh dispatches 'maintenance' (and alias 'weekly') to vault-maintenance-weekly.sh" {
    grep -qE 'maintenance\|weekly\)' "$VAULT_SCRIPT"
    grep -qE 'exec "\$SCRIPT_DIR/vault-maintenance-weekly.sh"' "$VAULT_SCRIPT"
}

@test "vault.sh dispatches 'check-escapes' (and alias 'check') to check-md-escapes.sh" {
    grep -qE 'check-escapes\|check\)' "$VAULT_SCRIPT"
    grep -qE 'exec "\$SCRIPT_DIR/check-md-escapes.sh"' "$VAULT_SCRIPT"
}

# --- Behavior smoke ---

@test "vault.sh with no args prints usage and exits 2" {
    run "$VAULT_SCRIPT"
    [ "$status" -eq 2 ]
    [[ "$output" =~ "Usage: vault" ]]
}

@test "vault.sh help prints usage and exits 0" {
    run "$VAULT_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage: vault" ]]
}

@test "vault.sh --help also prints usage" {
    run "$VAULT_SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage: vault" ]]
}

@test "vault.sh with unknown subcommand exits 2 with error" {
    run "$VAULT_SCRIPT" totally-not-a-real-subcommand
    [ "$status" -eq 2 ]
    [[ "$output" =~ "unknown subcommand" ]]
}

# --- Backing scripts still exist and still executable (dispatcher is additive,
# not a replacement — REFACTOR-005 scope choice). ---

@test "backing script: vault-health.sh still exists and executable" {
    [ -f "$HEALTH_SCRIPT" ] && [ -x "$HEALTH_SCRIPT" ]
}

@test "backing script: vault-maintenance-weekly.sh still exists and executable" {
    [ -f "$MAINT_SCRIPT" ] && [ -x "$MAINT_SCRIPT" ]
}

@test "backing script: check-md-escapes.sh still exists and executable" {
    [ -f "$ESCAPES_SCRIPT" ] && [ -x "$ESCAPES_SCRIPT" ]
}
