#!/usr/bin/env bats
# Tests for scripts/healthcheck.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
}

@test "healthcheck.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh is executable" {
    [[ -x "$SCRIPTS_DIR/healthcheck.sh" ]]
}

@test "healthcheck.sh sources utils.sh" {
    grep -q 'utils.sh' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh sources versions.conf" {
    grep -q 'versions.conf' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh has all 12 sections" {
    [[ $(grep -c 'section "' "$SCRIPTS_DIR/healthcheck.sh") -eq 12 ]]
}

@test "healthcheck.sh uses set -euo pipefail" {
    grep -q 'set -euo pipefail' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh defines pass/fail/skip functions" {
    grep -q '^pass()' "$SCRIPTS_DIR/healthcheck.sh"
    grep -q '^fail()' "$SCRIPTS_DIR/healthcheck.sh"
    grep -q '^skip()' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh exits 1 on failures" {
    grep -q 'exit 1' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh exits 0 on success" {
    grep -q 'exit 0' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh passes shellcheck" {
    if command -v shellcheck >/dev/null 2>&1; then
        shellcheck "$SCRIPTS_DIR/healthcheck.sh"
    elif [[ -x "$HOME/.local/bin/shellcheck" ]]; then
        "$HOME/.local/bin/shellcheck" "$SCRIPTS_DIR/healthcheck.sh"
    else
        skip "shellcheck not available"
    fi
}

# --- tmux section 9/12 ---

@test "healthcheck.sh has section 9/12 for tmux" {
    grep -qE 'section "9/12" "tmux"' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh verifies tmux binary" {
    grep -qE 'command -v tmux' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh verifies ~/.tmux.conf symlink target" {
    grep -qE 'readlink "\$HOME/\.tmux\.conf"' "$SCRIPTS_DIR/healthcheck.sh"
}

# --- ghostty section 11/12 (TERM-001) ---

@test "healthcheck.sh has section 11/12 for ghostty" {
    grep -qE 'section "11/12" "Ghostty"' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh verifies ghostty binary presence" {
    grep -qE 'command -v ghostty' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh verifies ghostty config deployed" {
    grep -qE 'HOME/\.config/ghostty/config' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh verifies ghostty version match against versions.conf" {
    grep -qE 'GHOSTTY_VERSION=' "$SCRIPTS_DIR/healthcheck.sh"
}

# --- drift section 12/12 ---

@test "healthcheck.sh has section 12/12 for drift" {
    grep -qE 'section "12/12" "Repo' "$SCRIPTS_DIR/healthcheck.sh"
}

@test "healthcheck.sh invokes diff-check.sh" {
    grep -q 'diff-check.sh' "$SCRIPTS_DIR/healthcheck.sh"
}
