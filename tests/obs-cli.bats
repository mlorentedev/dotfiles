#!/usr/bin/env bats
# Tests for scripts/obs-cli.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
}

@test "obs-cli.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/obs-cli.sh"
}

@test "obs-cli.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/obs-cli.sh"
}

@test "obs-cli.sh is executable" {
    [[ -x "$SCRIPTS_DIR/obs-cli.sh" ]]
}

@test "obs-cli.sh sources utils.sh" {
    grep -q 'utils.sh' "$SCRIPTS_DIR/obs-cli.sh"
}

@test "obs-cli.sh uses set -euo pipefail" {
    grep -q 'set -euo pipefail' "$SCRIPTS_DIR/obs-cli.sh"
}

@test "obs-cli.sh defaults OBS_VAULT to knowledge" {
    grep -q 'OBS_VAULT:-knowledge' "$SCRIPTS_DIR/obs-cli.sh"
}

@test "obs-cli.sh checks obsidian command exists" {
    grep -q 'command_exists obsidian' "$SCRIPTS_DIR/obs-cli.sh"
}

@test "obs-cli.sh exits 2 when GUI not running (via fail_obsidian_gui)" {
    # After the AUDIT-005 extraction, the canonical "exit 2 + error message"
    # lives in utils.sh::fail_obsidian_gui. obs-cli.sh delegates via `|| fail_obsidian_gui`.
    grep -q 'fail_obsidian_gui' "$SCRIPTS_DIR/obs-cli.sh"
    grep -q '^fail_obsidian_gui()' "$SCRIPTS_DIR/utils.sh"
    grep -q 'exit 2' "$SCRIPTS_DIR/utils.sh"
}

@test "obs-cli.sh passes --no-sandbox flag" {
    grep -q '\-\-no-sandbox' "$SCRIPTS_DIR/obs-cli.sh"
}

@test "obs-cli.sh has usage function" {
    grep -q 'usage()' "$SCRIPTS_DIR/obs-cli.sh"
}

@test "obs-cli.sh shows help with --help flag" {
    grep -q '\-\-help' "$SCRIPTS_DIR/obs-cli.sh"
}

@test "obs-cli.sh passes shellcheck" {
    if command -v shellcheck >/dev/null 2>&1; then
        shellcheck "$SCRIPTS_DIR/obs-cli.sh"
    elif [[ -x "$HOME/.local/bin/shellcheck" ]]; then
        "$HOME/.local/bin/shellcheck" "$SCRIPTS_DIR/obs-cli.sh"
    else
        skip "shellcheck not available"
    fi
}
