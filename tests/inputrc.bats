#!/usr/bin/env bats
# POLISH-004 (#133): .inputrc + .editorconfig cross-OS polish.
# Asserts repo files exist, carry the intended defaults, and that .inputrc is an
# ai/deploy.json entry (structural grep — no full setup run; #2013 P7).

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "POLISH-004: .inputrc exists at repo root" {
    [ -f "$DOTFILES_DIR/.inputrc" ]
}

@test "POLISH-004: .inputrc enables case-insensitive completion + arrow history search" {
    grep -q '^set completion-ignore-case on'   "$DOTFILES_DIR/.inputrc"
    grep -q '^set show-all-if-ambiguous on'     "$DOTFILES_DIR/.inputrc"
    grep -q '^set show-all-if-unmodified on'    "$DOTFILES_DIR/.inputrc"
    grep -q '^set colored-stats on'             "$DOTFILES_DIR/.inputrc"
    grep -q 'history-search-backward'           "$DOTFILES_DIR/.inputrc"
    grep -q 'history-search-forward'            "$DOTFILES_DIR/.inputrc"
}

@test "POLISH-004: .inputrc deploys to \$HOME/.inputrc through ai/deploy.json" {
    grep -qF '"dst": "{HOME}/.inputrc"' "$DOTFILES_DIR/ai/deploy.json"
}

@test "POLISH-004: .editorconfig exists with root=true and CRLF for PowerShell" {
    [ -f "$DOTFILES_DIR/.editorconfig" ]
    grep -q '^root = true' "$DOTFILES_DIR/.editorconfig"
    grep -q 'end_of_line = crlf' "$DOTFILES_DIR/.editorconfig"
}
