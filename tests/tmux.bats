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

# copy_command: the script tmux.conf hands to copy-command, read from the file
# so the tests run the text tmux would run.
copy_command() {
    sed -n "s/^set -s copy-command '\(.*\)'\$/\1/p" "$CONFIG_FILE"
}

# copy_with TOOLS [ENV...]: run copy-command the way tmux does (/bin/sh -c) on a
# PATH holding only stubs for TOOLS and cat, feeding it "text"; print what the
# chosen stub received, or nothing when no tool was there, and return
# copy-command's own status.
copy_with() {
    local tools=$1 stub="$BATS_TEST_TMPDIR/stub" out="$BATS_TEST_TMPDIR/out" t
    shift
    rm -rf "$stub" "$out"
    mkdir -p "$stub"
    ln -s "$(command -v cat)" "$stub/cat"
    for t in $tools; do
        # shellcheck disable=SC2016  # $0 and $* expand in the stub, not here
        printf '#!/bin/sh\n{ printf "%%s %%s <" "${0##*/}" "$*"; cat; } > "%s"\n' "$out" > "$stub/$t"
        chmod +x "$stub/$t"
    done
    local rc=0
    printf 'text' | env -i PATH="$stub" "$@" /bin/sh -c "$(copy_command)" || rc=$?
    [ ! -f "$out" ] || cat "$out"
    return "$rc"
}

@test "every copy binding pipes to copy-command and names no tool of its own" {
    [ "$(grep -cE '^bind -T copy-mode-vi (y|MouseDragEnd1Pane|DoubleClick1Pane) .*copy-pipe-and-cancel$' "$CONFIG_FILE")" -eq 3 ]
    grep -qE '^set -s set-clipboard on$' "$CONFIG_FILE"
}

@test "copy-command reaches each OS's clipboard tool, and succeeds with none" {
    [ -n "$(copy_command)" ]
    [ "$(copy_with "pbcopy xclip")" = "pbcopy  <text" ]
    [ "$(copy_with "wl-copy xclip" WAYLAND_DISPLAY=wayland-0)" = "wl-copy  <text" ]
    # An X11 session on a machine that also has wl-copy installed.
    [ "$(copy_with "wl-copy xclip" DISPLAY=:0)" = "xclip -selection clipboard -in <text" ]
    [ "$(copy_with "clip.exe")" = "clip.exe  <text" ]
    # WSL, or SSH, with xclip installed and DISPLAY unset: the DISPLAY guard
    # skips xclip, so the chain reaches clip.exe. A stale DISPLAY still execs
    # xclip, which has already read the text, so no later branch could copy it.
    [ "$(copy_with "xclip clip.exe")" = "clip.exe  <text" ]
    run copy_with ""
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "tmux reads copy-command as the file writes it" {
    if ! command -v tmux >/dev/null 2>&1; then
        skip "tmux not installed"
    fi
    local socket="tmux_bats_cc_$$"
    tmux -f "$CONFIG_FILE" -L "$socket" new-session -d -s test 'sleep 5'
    run tmux -L "$socket" show -sv copy-command
    tmux -L "$socket" kill-server 2>/dev/null || true
    [ "$status" -eq 0 ]
    local want got
    want=$(copy_command)
    # tmux 3.4 (Ubuntu's, measured in CI) vis-escapes what it prints
    # (server_client_print, VIS_CSTYLE) and shows \$ for a stored $; 3.8 prints
    # it raw. Undo that one escape; the script holds no backslash of its own, so
    # the undo is exact.
    [[ "$want" != *\\* ]] || false
    got=$(printf '%s' "$output" | sed 's/\\\$/$/g')
    [ "$got" = "$want" ] || {
        printf 'tmux %s shows: %s\nthe file has:  %s\n' "$(tmux -V)" "$output" "$want"
        false
    }
}

# default_terminal_with: the default-terminal a real tmux settles on when the
# host's infocmp answers with the given exit status.
default_terminal_with() {
    local bin="$BATS_TEST_TMPDIR/bin" socket="tmux_bats_term_$$_$1"
    mkdir -p "$bin"
    printf '#!/bin/sh\nexit %s\n' "$1" > "$bin/infocmp"
    chmod +x "$bin/infocmp"
    PATH="$bin:$PATH" tmux -f "$CONFIG_FILE" -L "$socket" new-session -d -s test 'sleep 5'
    tmux -L "$socket" show -gv default-terminal
    tmux -L "$socket" kill-server 2>/dev/null || true
}

@test "tmux picks tmux-256color only where its terminfo entry exists" {
    if ! command -v tmux >/dev/null 2>&1; then
        skip "tmux not installed"
    fi
    [ "$(default_terminal_with 0)" = "tmux-256color" ]
    [ "$(default_terminal_with 1)" = "screen-256color" ]
}
