#!/usr/bin/env bats
# Real-Git coverage for tests/check-doc-paths.bats, which stubs git only for
# Windows-worktree and malformed hook-environment failure paths.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    GUARD="$DOTFILES_DIR/scripts/check-doc-paths.sh"
}

@test "check-doc-paths: real Git auto-discovers tracked instruction files [#1021]" {
    run "$GUARD"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "check-doc-paths: OK AGENTS.md" ]] || false
    [[ "$output" =~ "check-doc-paths: OK ai/claude/CLAUDE.md" ]] || false
}
