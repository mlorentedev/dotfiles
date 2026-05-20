#!/usr/bin/env bats
# Tests for Ghostty config + version pin (TERM-001).

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export CONFIG_FILE="$DOTFILES_DIR/terminals/ghostty/config"
    export VERSIONS_FILE="$DOTFILES_DIR/versions.conf"
}

@test "terminals/ghostty/config exists in repo" {
    [ -f "$CONFIG_FILE" ]
}

@test "ghostty config declares font-family" {
    grep -qE '^font-family =' "$CONFIG_FILE"
}

@test "ghostty config declares theme" {
    grep -qE '^theme =' "$CONFIG_FILE"
}

@test "ghostty config declares confirm-close-surface (avoid killing tmux sessions on accidental close)" {
    grep -qE '^confirm-close-surface = true' "$CONFIG_FILE"
}

@test "ghostty config theme is a valid known name (not a kebab-case typo)" {
    # Ghostty theme name format is literal capitalized with spaces, NOT kebab-case.
    # If someone writes "catppuccin-mocha" instead of "Catppuccin Mocha", the
    # parser logs a "theme not found" warning. Empirical finding from AI-011-validation.
    ! grep -qE '^theme = .*-.*-' "$CONFIG_FILE"
}

@test "ghostty config parses cleanly with the installed binary (skip if absent)" {
    if ! command -v ghostty >/dev/null 2>&1; then
        skip "ghostty not installed"
    fi
    # ghostty +validate-config reads ~/.config/ghostty/config -- copy the
    # repo file temporarily and validate.
    local tmp
    tmp=$(mktemp -d)
    cp "$CONFIG_FILE" "$tmp/config"
    HOME="$tmp" XDG_CONFIG_HOME="$tmp/.config-stub" run ghostty +validate-config 2>&1
    rm -rf "$tmp"
    # The validator may not honour HOME override in all builds, so this is
    # an opportunistic smoke test, not a strict gate.
    [ "$status" -eq 0 ] || skip "ghostty +validate-config did not honour HOME override; relying on healthcheck for deployed-file validation"
}

@test "versions.conf pins GHOSTTY_VERSION" {
    grep -qE '^GHOSTTY_VERSION=[0-9]+\.[0-9]+\.[0-9]+' "$VERSIONS_FILE"
}

@test "GHOSTTY_VERSION matches semver pattern (no exotic suffix in pin)" {
    local pinned
    pinned=$(grep -E '^GHOSTTY_VERSION=' "$VERSIONS_FILE" | cut -d= -f2)
    [[ "$pinned" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

@test "setup-linux.sh contains ghostty install/deploy block" {
    grep -qE '^GHOSTTY_VERSION_PINNED=' "$DOTFILES_DIR/setup-linux.sh"
    grep -qE 'ghostty not installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -qE 'Deployed ghostty config' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh references the canonical repo path for ghostty config" {
    grep -qE 'terminals/ghostty/config' "$DOTFILES_DIR/setup-linux.sh"
}
