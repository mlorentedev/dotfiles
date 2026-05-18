#!/usr/bin/env bats
# Tests for tmux configuration

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export CONFIG_FILE="$DOTFILES_DIR/tmux.conf"
}

@test "tmux.conf exists at repo root" {
    [ -f "$CONFIG_FILE" ]
}

@test "tmux.conf parses cleanly with real tmux" {
    if ! command -v tmux >/dev/null 2>&1; then
        skip "tmux not installed"
    fi
    local socket="tmux_bats_$$"
    run tmux -f "$CONFIG_FILE" -L "$socket" new-session -d -s test 'true'
    local rc=$status
    tmux -L "$socket" kill-server 2>/dev/null || true
    [ "$rc" -eq 0 ]
}

@test "tmux.conf enables mouse" {
    grep -qE '^set -g mouse on' "$CONFIG_FILE"
}

@test "tmux.conf uses vi mode-keys" {
    grep -qE '^setw -g mode-keys vi' "$CONFIG_FILE"
}

@test "tmux.conf sets 1-based indexing for windows and panes" {
    grep -qE '^set -g base-index 1' "$CONFIG_FILE"
    grep -qE '^setw -g pane-base-index 1' "$CONFIG_FILE"
}

@test "tmux.conf binds 'y' to copy selection to system clipboard via xclip" {
    grep -qE "^bind -T copy-mode-vi y .*copy-pipe-and-cancel .*xclip" "$CONFIG_FILE"
}

@test "tmux.conf pipes mouse-drag-end selection to system clipboard" {
    grep -qE "^bind -T copy-mode-vi MouseDragEnd1Pane .*copy-pipe-and-cancel .*xclip" "$CONFIG_FILE"
}

@test "tmux.conf enables truecolor passthrough for ghostty" {
    grep -qE 'xterm-ghostty:Tc' "$CONFIG_FILE"
}
