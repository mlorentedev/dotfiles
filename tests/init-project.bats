#!/usr/bin/env bats
# Tests for scripts/init-project.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export TEST_DIR="/tmp/bats_init_$$"
    mkdir -p "$TEST_DIR"
}

teardown() {
    rm -rf "$TEST_DIR"
}

@test "init-project.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/init-project.sh"
}

@test "init-project.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/init-project.sh"
}

@test "init-project.sh creates project structure" {
    run bash -c 'cd "$1" && bash "$2/init-project.sh" . none 2>&1' -- "$TEST_DIR" "$SCRIPTS_DIR"
    [[ -d "$TEST_DIR/src" ]]
    [[ -d "$TEST_DIR/tests" ]]
    [[ -d "$TEST_DIR/tasks" ]]
    [[ -f "$TEST_DIR/tasks/todo.md" ]]
    [[ -f "$TEST_DIR/tasks/lessons.md" ]]
}

@test "init-project.sh creates structure under zsh" {
    local zdir="/tmp/bats_init_zsh_$$"
    mkdir -p "$zdir"
    run zsh -c 'cd "$1" && zsh "$2/init-project.sh" . none 2>&1' -- "$zdir" "$SCRIPTS_DIR"
    [[ -d "$zdir/src" ]]
    [[ -d "$zdir/tasks" ]]
    rm -rf "$zdir"
}
