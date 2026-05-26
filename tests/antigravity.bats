#!/usr/bin/env bats
# Tests for Antigravity CLI configuration (SDD-007: fresh-install model, no symlinks).
# Regression suite for BUG-100 (issue #100): asserts no symlinks under ~/.gemini/config/
# and that the master MCP config lives at agy's canonical read path.
#
# These tests assume agy is installed AND setup-linux.sh has been run.
# CI containers without agy binary or post-setup state will skip these tests.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export GEMINI_HOME="$HOME/.gemini"
    export AGY_APP_DATA="$GEMINI_HOME/antigravity-cli"
    export MASTER_CONFIG="$GEMINI_HOME/config/mcp_config.json"

    # Skip the entire suite when agy is not installed (CI / fresh machine).
    # These are integration-style tests, not unit tests of repo state.
    if ! command -v agy >/dev/null 2>&1; then
        skip "agy not installed (CI / fresh machine) — install from https://antigravity.google"
    fi
}

@test "agy binary exists and is executable" {
    command -v agy
}

@test "ai/agy/AGY.md H1 is '# AGY.md' (regression: GEMINI.md->AGY.md rename completeness)" {
    # SDD-007 renamed the file; the H1 and body content lagged. This test
    # locks in that the body actually reflects the file's new identity.
    [[ "$(head -n1 "$DOTFILES_DIR/ai/agy/AGY.md")" == "# AGY.md" ]]
    # No stray "Gemini-specific" / "GEMINI.md" body references in the
    # canonical SSOT (archived specs and the cleanup logic in setup-*.{sh,ps1}
    # are excluded by file scope).
    ! grep -qF 'Gemini-specific' "$DOTFILES_DIR/ai/agy/AGY.md"
    ! grep -qF '# GEMINI.md' "$DOTFILES_DIR/ai/agy/AGY.md"
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
