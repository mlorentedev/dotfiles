#!/usr/bin/env bats
# Tests for .zsh/aliases.zsh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export ALIASES_FILE="$DOTFILES_DIR/.zsh/aliases.zsh"
}

@test "aliases.zsh exists" {
    [[ -f "$ALIASES_FILE" ]]
}

@test "aliases.zsh valid bash syntax" {
    bash -n "$ALIASES_FILE"
}

@test "aliases.zsh valid zsh syntax" {
    zsh -n "$ALIASES_FILE"
}

# --- DevOps aliases ---

@test "aliases.zsh defines kubectl alias" {
    grep -q 'alias k="kubectl"' "$ALIASES_FILE"
}

@test "aliases.zsh defines helm alias" {
    grep -q 'alias h="helm"' "$ALIASES_FILE"
}

@test "aliases.zsh defines terraform alias" {
    grep -q 'alias tf="terraform"' "$ALIASES_FILE"
}

# --- Aider tier aliases ---

@test "aliases.zsh defines ai alias for daily tier" {
    grep -q 'alias ai="aider"' "$ALIASES_FILE"
}

@test "aliases.zsh defines aic alias for coding tier" {
    grep -q 'alias aic=.*qwen.*coder' "$ALIASES_FILE"
}

@test "aliases.zsh defines aia alias for architecture tier" {
    grep -q 'alias aia=.*architect.*speciale' "$ALIASES_FILE"
}

@test "aliases.zsh aic uses openrouter provider" {
    grep 'alias aic=' "$ALIASES_FILE" | grep -q 'openrouter/'
}

@test "aliases.zsh aia uses openrouter provider" {
    grep 'alias aia=' "$ALIASES_FILE" | grep -q 'openrouter/'
}

# --- Knowledge aliases ---

@test "aliases.zsh defines kc alias" {
    grep -q 'alias kc=' "$ALIASES_FILE"
}

@test "aliases.zsh defines kca alias" {
    grep -q 'alias kca=' "$ALIASES_FILE"
}
