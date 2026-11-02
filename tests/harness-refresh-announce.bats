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

# scripts/compile-harness.sh is deleted (CLI-026). A call to it that came back
# would fail, and setup runs such calls with output discarded, so the failure
# would be silent: the records and the agent copies would just stop updating.
# Comments may name the script; a real invocation may not, in either setup.
_script_calls() {
    grep -nE 'compile-harness' "$1" | grep -vE '^[0-9]+:[[:space:]]*#' || true
}

@test "neither setup script invokes the deleted compile-harness.sh" {
    run _script_calls "$DOTFILES_DIR/setup-linux.sh"
    [ -z "$output" ]
    run _script_calls "$DOTFILES_DIR/setup-windows.ps1"
    [ -z "$output" ]
}

@test "the no-invocation guard detects the shape it forbids" {
    local probe="$BATS_TEST_TMPDIR/probe.sh"
    printf '%s\n' '# compile-harness.sh --refresh in a comment is fine' \
        '    if ( cd "$CURRENT_DIR" && "$CURRENT_DIR/scripts/compile-harness.sh" --refresh ) >/dev/null 2>&1; then' > "$probe"
    run _script_calls "$probe"
    [[ "$output" == "2:"* ]] || false
}

@test "setup-linux.sh stays valid bash" {
    bash -n "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh stays valid zsh" {
    zsh -n "$DOTFILES_DIR/setup-linux.sh"
}
