#!/usr/bin/env bats
# Tests for Antigravity CLI configuration and stability

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export GEMINI_HOME="$HOME/.gemini"
    export AGY_APP_DATA="$GEMINI_HOME/antigravity-cli"
}

@test "Antigravity binary exists and is executable" {
    command -v agy
}

@test "Antigravity production endpoint is set" {
    # Check .zshrc for the export
    grep -q "export ANTIGRAVITY_ENDPOINT=\"https://cloudcode-pa.googleapis.com\"" "$HOME/.zshrc"
}

@test "AGY_APP_DATA uses absolute path" {
    # It should not be relative
    [[ "$AGY_APP_DATA" == /* ]]
}

@test "Antigravity settings.json exists in app data" {
    [ -f "$AGY_APP_DATA/settings.json" ]
}

@test "Antigravity mcp_config.json is valid JSON" {
    [ -f "$AGY_APP_DATA/mcp_config.json" ]
    jq '.' "$AGY_APP_DATA/mcp_config.json"
}

@test "Antigravity mcp_config.json is not empty" {
    [ -s "$AGY_APP_DATA/mcp_config.json" ]
}

@test "Old Gemini settings.json schema compatibility" {
    # Verify gemini doesn't crash on settings.json if it exists
    if command -v gemini >/dev/null 2>&1; then
        # This might still fail if we haven't patched it, but the goal is to fix it
        gemini --version
    else
        skip "gemini-cli not installed"
    fi
}
