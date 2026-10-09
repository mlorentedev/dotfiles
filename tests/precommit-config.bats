#!/usr/bin/env bats
# The repo's own .pre-commit-config.yaml. No installer: the global dispatcher
# (git-hooks/) hands every stage to `pre-commit hook-impl`, and `dotf tools
# install pre-commit` puts the binary on the machine.

# bats file_tags=os-sensitive

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
}

@test ".pre-commit-config.yaml declares sdd-spec-gate pre-push hook" {
    grep -q 'sdd-spec-gate' "$DOTFILES_DIR/.pre-commit-config.yaml"
    grep -A1 'sdd-spec-gate' "$DOTFILES_DIR/.pre-commit-config.yaml" | grep -q 'spec-gate'
    grep -B1 'pre-push' "$DOTFILES_DIR/.pre-commit-config.yaml" | grep -q 'sdd-spec-gate\|stages'
}

@test ".pre-commit-config.yaml declares check-bats-names at pre-commit stage [#925]" {
    # scripts/test.sh (the dotfiles-test hook) never ran bats or this script —
    # a non-ASCII @test name shipped past local pre-commit and was only
    # caught by a full adversarial-review round. This hook is the guard.
    grep -q 'check-bats-names' "$DOTFILES_DIR/.pre-commit-config.yaml"
    grep -A2 'id: check-bats-names' "$DOTFILES_DIR/.pre-commit-config.yaml" | grep -q 'check-bats-names.sh'
    grep -A5 'id: check-bats-names' "$DOTFILES_DIR/.pre-commit-config.yaml" | grep -q 'pre-commit'
}

@test "check-bats-names.sh exits 0 on this repo's own tests/ tree [#925]" {
    # The hook is only a guard if it actually passes on a clean tree — pins
    # that invariant so a future non-ASCII @test name is caught here, not
    # three rounds of review later.
    run "$SCRIPTS_DIR/check-bats-names.sh" "$DOTFILES_DIR/tests/"
    [ "$status" -eq 0 ]
}
