#!/usr/bin/env bats
# Tests for the systematic-debugging skill's find-polluter.sh helper.
# The skill now lives in the vault; its committed record under harness/skills/
# is the in-repo SSOT (SDD-008), so the auxiliary script is tested there.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPT="$DOTFILES_DIR/harness/skills/systematic-debugging/find-polluter.sh"
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
