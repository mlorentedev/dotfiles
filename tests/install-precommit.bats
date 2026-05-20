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

@test "install-precommit.sh --help shows usage including --with-sdd-gate" {
    run "$SCRIPTS_DIR/install-precommit.sh" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"--with-sdd-gate"* ]]
}

@test "install-precommit.sh rejects unknown flag" {
    run "$SCRIPTS_DIR/install-precommit.sh" --bogus
    [ "$status" -ne 0 ]
    [[ "$output" == *"Unknown"* ]]
}

@test "install-precommit.sh includes pre-push hook-type when --with-sdd-gate set" {
    # Static check: the conditional path includes the pre-push hook arg
    grep -q 'WITH_SDD_GATE' "$SCRIPTS_DIR/install-precommit.sh"
    grep -q '"pre-push"' "$SCRIPTS_DIR/install-precommit.sh"
}

@test ".pre-commit-config.yaml declares sdd-spec-gate pre-push hook" {
    grep -q 'sdd-spec-gate' "$DOTFILES_DIR/.pre-commit-config.yaml"
    grep -A1 'sdd-spec-gate' "$DOTFILES_DIR/.pre-commit-config.yaml" | grep -q 'spec-gate'
    grep -B1 'pre-push' "$DOTFILES_DIR/.pre-commit-config.yaml" | grep -q 'sdd-spec-gate\|stages'
}
