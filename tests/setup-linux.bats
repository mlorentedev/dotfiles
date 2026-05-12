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
