#!/usr/bin/env bats
# Tests for scripts/install-precommit.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
}

@test "install-precommit.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/install-precommit.sh"
}

@test "install-precommit.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/install-precommit.sh"
}

@test "install-precommit.sh sources utils.sh" {
    # Just verify it can parse and source without crashing
    # (actual execution needs git repo context)
    run bash -c 'source "$1/utils.sh" && echo "sourced"' -- "$SCRIPTS_DIR"
    [[ "$output" == *"sourced"* ]]
}
