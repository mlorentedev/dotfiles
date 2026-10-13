#!/usr/bin/env bats
# GUARD: agy's MCP server configs are written by dotf, never by a setup twin
# (CLI-105, #2384).
#
# setup-linux.sh and setup-windows.ps1 each built ~/.gemini/config/mcp_config.json
# and the hive-vault plugin pair with their own jq or ConvertTo-Json, so a Mac,
# which runs neither, had no agy MCP config at all. cli/internal/agy writes them
# now, from the bare `dotf deploy` and converge's agent-extensions step. A block
# that grew back in either twin would be a second writer of the same files, and
# the two would drift exactly as the twins did. Any mention of agy's source list
# or of the file name in a twin's code is that block.

load 'lib/refute'

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "neither setup twin writes agy's MCP configs" {
    for twin in setup-linux.sh setup-windows.ps1; do
        [ -s "$DOTFILES_DIR/$twin" ] || { echo "$twin missing or empty"; false; }
        refute_grep 'mcp_config|ai[/\\]agy[/\\]mcp_servers' "$DOTFILES_DIR/$twin"
    done
}

@test "the bare deploy and converge both call the one writer" {
    grep -qF 'agy.ApplyMCP(' "$DOTFILES_DIR/cli/internal/cmd/deploy_agy.go"
    grep -qF 'agy.ApplyMCP(' "$DOTFILES_DIR/cli/internal/converge/extensions.go"
}
