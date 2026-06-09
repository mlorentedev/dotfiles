#!/usr/bin/env bats
# OPS-003: after a successful compile-harness --refresh, setup-linux.sh must
# announce any resulting working-tree drift LOUDLY (warning + file list + commit
# hint), instead of leaving silent uncommitted changes that read as a parallel
# session's WIP (the failure mode that misled a session in #295).

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "setup-linux.sh checks the working tree for drift after --refresh" {
    grep -qF 'git -C "$CURRENT_DIR" status --porcelain -- harness/ AGENTS.md ai/claude/CLAUDE.md' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh announces refreshed harness records loudly (warning + commit hint)" {
    grep -qF 'Harness records changed by --refresh from the vault' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF "chore(harness): refresh records from vault" "$DOTFILES_DIR/setup-linux.sh"
}

@test "the drift announce is gated to the git repo (no-op in the ~/.dotfiles mirror)" {
    grep -qF 'git -C "$CURRENT_DIR" rev-parse --git-dir' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh stays valid bash after the announce block" {
    bash -n "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh stays valid zsh after the announce block" {
    zsh -n "$DOTFILES_DIR/setup-linux.sh"
}

# Windows runs --deploy only (ENGINE-001: --refresh/--check are Linux-only), so it
# never mutates the committed records and needs no announce. Encode the invariant
# so a future Windows --refresh port cannot silently reintroduce the drift class.
# A comment reference to --refresh is fine; a real invocation is not.
@test "setup-windows.ps1 has no real compile-harness --refresh invocation (Linux-only)" {
    run bash -c "grep -E 'compile-harness.*--refresh' '$DOTFILES_DIR/setup-windows.ps1' | grep -vE '^[[:space:]]*#' || true"
    [ -z "$output" ]
}
