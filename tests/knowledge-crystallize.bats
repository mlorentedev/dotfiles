#!/usr/bin/env bats
# Tests for scripts/knowledge-crystallize.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
}

@test "knowledge-crystallize.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh is executable" {
    [[ -x "$SCRIPTS_DIR/knowledge-crystallize.sh" ]]
}

@test "knowledge-crystallize.sh uses set -euo pipefail" {
    grep -q 'set -euo pipefail' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh sources utils.sh" {
    grep -q 'utils.sh' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh has --help or usage output" {
    grep -q 'usage\(\)' "$SCRIPTS_DIR/knowledge-crystallize.sh"
    grep -q '\-\-help' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh passes shellcheck" {
    if command -v shellcheck >/dev/null 2>&1; then
        shellcheck "$SCRIPTS_DIR/knowledge-crystallize.sh"
    elif [[ -x "$HOME/.local/bin/shellcheck" ]]; then
        "$HOME/.local/bin/shellcheck" "$SCRIPTS_DIR/knowledge-crystallize.sh"
    else
        skip "shellcheck not available"
    fi
}

@test "knowledge-crystallize.sh has encode_path function" {
    grep -q 'encode_path\(\)' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh has find_memory_file function" {
    grep -q 'find_memory_file\(\)' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh has stamp_last_crystallized function" {
    grep -q 'stamp_last_crystallized\(\)' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh prints checklist" {
    grep -q 'print_checklist' "$SCRIPTS_DIR/knowledge-crystallize.sh"
    grep -q 'crystallize\|crystallization' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh --help exits 0" {
    run "$SCRIPTS_DIR/knowledge-crystallize.sh" --help
    [[ "$status" -eq 0 ]]
}

@test "knowledge-crystallize.sh --help shows usage" {
    run "$SCRIPTS_DIR/knowledge-crystallize.sh" --help
    [[ "$output" == *"USAGE"* ]]
}
