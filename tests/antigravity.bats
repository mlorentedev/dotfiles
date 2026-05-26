#!/usr/bin/env bats
# Tests for Antigravity CLI configuration (SDD-007: fresh-install model, no symlinks).
# Regression suite for BUG-100 (issue #100): asserts no symlinks under ~/.gemini/config/
# and that the master MCP config lives at agy's canonical read path.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export GEMINI_HOME="$HOME/.gemini"
    export AGY_APP_DATA="$GEMINI_HOME/antigravity-cli"
    export MASTER_CONFIG="$GEMINI_HOME/config/mcp_config.json"
}

@test "agy binary exists and is executable" {
    command -v agy
}

@test "ANTIGRAVITY_ENDPOINT is set to production in .zshrc" {
    grep -q 'export ANTIGRAVITY_ENDPOINT="https://cloudcode-pa.googleapis.com"' "$DOTFILES_DIR/.zshrc"
}

@test "AGY_APP_DATA uses absolute path" {
    [[ "$AGY_APP_DATA" == /* ]]
}

@test "agy settings.json deployed to AGY_APP_DATA" {
    [ -f "$AGY_APP_DATA/settings.json" ]
    [ ! -L "$AGY_APP_DATA/settings.json" ]
}

@test "master MCP config exists at agy canonical path (~/.gemini/config/)" {
    [ -e "$MASTER_CONFIG" ]
}

@test "master MCP config is a regular file, NOT a symlink (BUG-100 regression)" {
    [ -f "$MASTER_CONFIG" ]
    [ ! -L "$MASTER_CONFIG" ]
}

@test "master MCP config is valid JSON" {
    jq '.' "$MASTER_CONFIG"
}

@test "master MCP config is not empty" {
    [ -s "$MASTER_CONFIG" ]
}

@test "no symlinks anywhere under ~/.gemini/config/ (BUG-100 guard)" {
    # find -type l would surface any leftover symlink; assert zero hits
    result=$(find "$GEMINI_HOME/config" -maxdepth 3 -type l 2>/dev/null | wc -l)
    [ "$result" -eq 0 ]
}

@test ".geminiignore deployed and contains sensitive/" {
    [ -f "$GEMINI_HOME/.geminiignore" ]
    grep -q '^sensitive/' "$GEMINI_HOME/.geminiignore"
}
