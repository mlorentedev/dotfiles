#!/usr/bin/env bats
# Tests for scripts/dotfiles-sync.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
}

@test "dotfiles-sync.sh has valid bash syntax" {
    bash -n "$SCRIPTS_DIR/dotfiles-sync.sh"
}

@test "dotfiles-sync.sh has valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/dotfiles-sync.sh"
}

@test "dotfiles-sync.sh handles missing DOTFILES_REPO_DIR" {
    run bash -c 'DOTFILES_REPO_DIR="" DOTFILES_DIR="/tmp/nonexist_$$" "$1/dotfiles-sync.sh" --secrets-only' -- "$SCRIPTS_DIR"
    # Should error about directory not found
    [[ $status -ne 0 ]] || [[ "$output" == *"not found"* ]]
}

@test "dotfiles-sync.sh handles same directory" {
    run bash -c 'DOTFILES_REPO_DIR="/tmp/same" DOTFILES_DIR="/tmp/same" "$1/dotfiles-sync.sh" --secrets-only 2>&1' -- "$SCRIPTS_DIR"
    [[ "$output" == *"same directory"* ]] || [[ $status -ne 0 ]]
}

@test "dotfiles-sync.sh --secrets-only flag recognized under bash" {
    # Create temp dirs to simulate
    local tdir="/tmp/bats_sync_$$"
    mkdir -p "$tdir/local/sensitive" "$tdir/repo/sensitive"
    run bash -c 'DOTFILES_DIR="'"$tdir/local"'" DOTFILES_REPO_DIR="'"$tdir/repo"'" "$1/dotfiles-sync.sh" --secrets-only' -- "$SCRIPTS_DIR"
    [[ "$output" == *"Sync"* ]] || [[ "$output" == *"sync"* ]]
    rm -rf "$tdir"
}

@test "dotfiles-sync.sh --secrets-only works under zsh" {
    local tdir="/tmp/bats_sync_zsh_$$"
    mkdir -p "$tdir/local/sensitive" "$tdir/repo/sensitive"
    run zsh -c 'DOTFILES_DIR="'"$tdir/local"'" DOTFILES_REPO_DIR="'"$tdir/repo"'" "$1/dotfiles-sync.sh" --secrets-only' -- "$SCRIPTS_DIR"
    [[ "$output" == *"Sync"* ]] || [[ "$output" == *"sync"* ]]
    rm -rf "$tdir"
}
