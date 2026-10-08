#!/usr/bin/env bats
# OPS-003 / #2162: setup-linux.sh refreshes the harness records through
# `dotf harness refresh`, which checks the vault is level with its upstream
# before reading it and announces any change the refresh leaves in the checkout.
# Both behaviours are tested in Go (cli/internal/cmd/harness_refresh_test.go);
# this file guards the setup side: that setup calls the command, and that the
# unchecked refresh it replaced does not come back.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "setup-linux.sh refreshes through dotf harness refresh" {
    grep -qF '"$_dotf" harness refresh --repo "$CURRENT_DIR"' "$DOTFILES_DIR/setup-linux.sh"
}

# A direct `compile-harness.sh --refresh` reads whatever the local vault clone
# holds. From a clone that was behind, it rewrote merged records to their older
# state and setup asked for the revert to be committed (#2162). Comments may
# name the script; a real invocation may not.
@test "setup-linux.sh has no direct compile-harness --refresh invocation (#2162)" {
    run bash -c "grep -nE 'compile-harness[^ ]* --refresh' '$DOTFILES_DIR/setup-linux.sh' | grep -vE '^[0-9]+:[[:space:]]*#' || true"
    [ -z "$output" ]
}

@test "the no-direct-refresh guard detects the shape it forbids" {
    local probe
    probe="$(mktemp)"
    printf '%s\n' '# compile-harness.sh --refresh in a comment is fine' \
        '    if ( cd "$CURRENT_DIR" && "$CURRENT_DIR/scripts/compile-harness.sh" --refresh ) >/dev/null 2>&1; then' > "$probe"
    run bash -c "grep -nE 'compile-harness[^ ]* --refresh' '$probe' | grep -vE '^[0-9]+:[[:space:]]*#' || true"
    rm -f "$probe"
    [[ "$output" == "2:"* ]] || false
}

@test "setup-linux.sh stays valid bash" {
    bash -n "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh stays valid zsh" {
    zsh -n "$DOTFILES_DIR/setup-linux.sh"
}

# Windows runs --deploy only (ENGINE-001: --refresh/--check are Linux-only), so it
# never mutates the committed records and needs no announce. Encode the invariant
# so a future Windows --refresh port cannot silently reintroduce the drift class:
# one that wants a refresh calls `dotf harness refresh`, which is OS-agnostic.
# A comment reference to --refresh is fine; a real invocation is not.
@test "setup-windows.ps1 has no real compile-harness --refresh invocation (Linux-only)" {
    run bash -c "grep -E 'compile-harness.*--refresh' '$DOTFILES_DIR/setup-windows.ps1' | grep -vE '^[[:space:]]*#' || true"
    [ -z "$output" ]
}
