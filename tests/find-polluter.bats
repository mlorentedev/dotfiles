#!/usr/bin/env bats
# Tests for ai/skills/systematic-debugging/find-polluter.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPT="$DOTFILES_DIR/ai/skills/systematic-debugging/find-polluter.sh"
}

@test "find-polluter.sh valid bash syntax" {
    bash -n "$SCRIPT"
}

@test "find-polluter.sh valid zsh syntax" {
    zsh -n "$SCRIPT"
}

@test "find-polluter.sh shows usage with wrong args" {
    run bash "$SCRIPT" 2>&1
    [[ "$output" == *"Usage"* ]]
    [[ $status -eq 1 ]]
}

@test "find-polluter.sh shows usage under zsh" {
    run zsh "$SCRIPT" 2>&1
    [[ "$output" == *"Usage"* ]]
    [[ $status -eq 1 ]]
}
