#!/usr/bin/env bats
# Tests for setup-linux.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "setup-linux.sh valid bash syntax" {
    bash -n "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh valid zsh syntax" {
    zsh -n "$DOTFILES_DIR/setup-linux.sh"
}

# --- Developer tools section ---

@test "setup-linux.sh creates ~/.local/bin directory" {
    grep -q 'ensure_directory.*\.local/bin' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh adds ~/.local/bin to PATH" {
    grep -q 'export PATH.*\.local/bin' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs uv if missing" {
    grep -q 'command -v uv' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'astral.sh/uv/install.sh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs poetry via uv" {
    grep -q 'command -v poetry' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'uv tool install poetry' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs age if missing" {
    grep -q 'command -v age' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'filippo.io/age' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs eza if missing" {
    grep -q 'command -v eza' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'eza.*linux.*tar.gz' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs jq if missing" {
    grep -q 'command -v jq' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'jq-linux-amd64' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs gh if missing" {
    grep -q 'command -v gh' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'cli/cli/releases' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs zoxide if missing" {
    grep -q 'command -v zoxide' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'zoxide.*install.sh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs direnv if missing" {
    grep -q 'command -v direnv' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'direnv.net/install.sh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs shellcheck if missing" {
    grep -q 'command -v shellcheck' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'ShellCheck/releases' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs bats if missing" {
    grep -q 'command -v bats' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'bats-core/bats-core' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh skips tools already installed" {
    grep -q 'uv already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'poetry already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'age already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'eza already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'jq already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'gh already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'zoxide already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'direnv already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'shellcheck already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'bats already installed' "$DOTFILES_DIR/setup-linux.sh"
}

# --- tmux integration ---

@test "setup-linux.sh copies tmux.conf into deploy dir" {
    grep -qE 'safe_copy "\$CURRENT_DIR/tmux\.conf" "\$DOTFILES_DIR/"' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh symlinks tmux.conf to ~/.tmux.conf" {
    grep -qE 'ln -sf "\$DOTFILES_DIR/tmux\.conf" "\$HOME/\.tmux\.conf"' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh checks for tmux presence" {
    grep -qE 'command -v tmux' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh tells user how to install tmux when missing" {
    grep -qE 'sudo apt install -y tmux' "$DOTFILES_DIR/setup-linux.sh"
}

# --- SessionStart hook registration (issue #20 prevention) ---

@test "setup-linux.sh registers SessionStart hook in Claude settings" {
    grep -q 'SessionStart' "$DOTFILES_DIR/setup-linux.sh"
}

# Hook command must point at the canonical deploy directory under ~/.dotfiles/scripts,
# matching where the install step actually lands claude-session-start.sh.
@test "setup-linux.sh SessionStart hook command points at dotfiles scripts dir" {
    grep -qE 'EXPECTED_HOOK_COMMAND="\$HOME/\.dotfiles/scripts/claude-session-start\.sh"' "$DOTFILES_DIR/setup-linux.sh"
}

# Hook registration must reconcile (compare and rewrite) rather than skip when
# any SessionStart entry already exists. Mirrors the Windows guard.
@test "setup-linux.sh SessionStart hook self-heals on path drift" {
    grep -q 'EXISTING_HOOK_COMMAND' "$DOTFILES_DIR/setup-linux.sh"
    grep -qE '\[ "\$EXISTING_HOOK_COMMAND" = "\$EXPECTED_HOOK_COMMAND" \]' "$DOTFILES_DIR/setup-linux.sh"
}

# --- MCP server registration (SSOT + idempotence) ---

@test "mcp-servers.json exists and is valid JSON with at least one server" {
    [ -f "$DOTFILES_DIR/mcp-servers.json" ]
    if command -v jq >/dev/null 2>&1; then
        run jq -e '.servers | length > 0' "$DOTFILES_DIR/mcp-servers.json"
        [ "$status" -eq 0 ]
    fi
}

# MCP server list must live in mcp-servers.json, not hardcoded in setup-linux.sh.
@test "setup-linux.sh MCP registration reads from mcp-servers.json" {
    grep -q 'mcp-servers\.json' "$DOTFILES_DIR/setup-linux.sh"
    ! grep -qE 'claude mcp add --transport (stdio|http) (drawio|socket|context7|sequential-thinking|hive)' "$DOTFILES_DIR/setup-linux.sh"
}

# MCP registration must check existence before adding, not blindly retry.
@test "setup-linux.sh MCP registration checks existence with claude mcp get" {
    grep -q 'claude mcp get' "$DOTFILES_DIR/setup-linux.sh"
}

# Both setup scripts must read the same SSOT.
@test "parity: both setup scripts source mcp-servers.json" {
    grep -q 'mcp-servers\.json' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'mcp-servers\.json' "$DOTFILES_DIR/setup-windows.ps1"
}

# --- claude-mem-heal cross-OS parity ---

@test "claude-mem-heal.ps1 exists alongside the bash version" {
    [ -f "$DOTFILES_DIR/scripts/claude-mem-heal.sh" ]
    [ -f "$DOTFILES_DIR/scripts/claude-mem-heal.ps1" ]
}

# Both heal scripts must use --ignore-scripts to avoid triggering native
# postinstalls (tree-sitter -> node-gyp -> MSBuild on Windows, build-essential
# on Linux). Only zod's pure-JS files are needed.
@test "claude-mem-heal scripts both use --ignore-scripts on npm install" {
    grep -q -- '--ignore-scripts' "$DOTFILES_DIR/scripts/claude-mem-heal.sh"
    grep -q -- '--ignore-scripts' "$DOTFILES_DIR/scripts/claude-mem-heal.ps1"
}

# Both SessionStart hooks must invoke the heal at session start.
@test "parity: both SessionStart hooks invoke claude-mem-heal" {
    grep -q 'claude-mem-heal\.sh' "$DOTFILES_DIR/scripts/claude-session-start.sh"
    grep -q 'claude-mem-heal\.ps1' "$DOTFILES_DIR/scripts/claude-session-start.ps1"
}
