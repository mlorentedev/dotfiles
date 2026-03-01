#!/usr/bin/env bats
# Tests for scripts/init-project.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export TEST_DIR="/tmp/bats_init_$$"
    export MOCK_HOME="/tmp/bats_home_$$"
    mkdir -p "$TEST_DIR"
    mkdir -p "$MOCK_HOME"
}

teardown() {
    rm -rf "$TEST_DIR"
    rm -rf "$MOCK_HOME"
}

@test "init-project.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/init-project.sh"
}

@test "init-project.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/init-project.sh"
}

@test "init-project.sh creates project structure" {
    run bash -c 'export HOME="$3"; cd "$1" && bash "$2/init-project.sh" . none 2>&1' -- "$TEST_DIR" "$SCRIPTS_DIR" "$MOCK_HOME"
    [[ -d "$TEST_DIR/src" ]]
    [[ -d "$TEST_DIR/tests" ]]
    [[ ! -d "$TEST_DIR/tasks" ]]
    
    local kb_dir="$MOCK_HOME/Projects/knowledge/10_projects/$(basename "$TEST_DIR")"
    [[ -d "$kb_dir" ]]
    [[ -f "$kb_dir/11-tasks.md" ]]
    [[ -f "$kb_dir/90-lessons.md" ]]
}

@test "init-project.sh creates structure under zsh" {
    local zdir="/tmp/bats_init_zsh_$$"
    mkdir -p "$zdir"
    run zsh -c 'export HOME="$3"; cd "$1" && zsh "$2/init-project.sh" . none 2>&1' -- "$zdir" "$SCRIPTS_DIR" "$MOCK_HOME"
    [[ -d "$zdir/src" ]]
    [[ ! -d "$zdir/tasks" ]]
    
    local kb_dir="$MOCK_HOME/Projects/knowledge/10_projects/$(basename "$zdir")"
    [[ -d "$kb_dir" ]]
    rm -rf "$zdir"
}
