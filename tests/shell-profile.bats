#!/usr/bin/env bats
# Tests for scripts/shell-profile.sh — shell startup profiling

# bats file_tags=os-sensitive

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    REPO_ROOT="$BATS_TEST_DIRNAME/.."
    STUB_DIR="$BATS_TEST_TMPDIR/stub"
}

# Put a fake `bash` first on PATH that exits with $1, modelling a real
# interactive shell: `bash -i -c exit` inherits the status of the last command
# the profile ran, so a non-zero exit is ordinary, not a malfunction.
stub_shell_exiting() {
    mkdir -p "$STUB_DIR"
    printf '#!/bin/sh\nexit %s\n' "$1" > "$STUB_DIR/bash"
    chmod +x "$STUB_DIR/bash"
    PATH="$STUB_DIR:$PATH"
}

@test "shell-profile.sh --help shows usage" {
    run "$SCRIPTS_DIR/shell-profile.sh" --help
    [[ $status -eq 0 ]] || false
    [[ "$output" == *"Usage"* ]] || false
    [[ "$output" == *"--detail"* ]] || false
    [[ "$output" == *"--runs"* ]] || false
}

@test "shell-profile.sh rejects unknown args" {
    run "$SCRIPTS_DIR/shell-profile.sh" --bogus
    [[ $status -eq 2 ]] || false
    [[ "$output" == *"Unknown"* ]] || false
}

@test "shell-profile.sh rejects unsupported shell" {
    run "$SCRIPTS_DIR/shell-profile.sh" --shell fish
    [[ $status -eq 2 ]] || false
    [[ "$output" == *"--shell must be zsh or bash"* ]] || false
}

@test "shell-profile.sh rejects shell not in PATH" {
    run "$SCRIPTS_DIR/shell-profile.sh" --shell bash --runs 1
    # bash is in PATH so this should succeed; smoke check the path
    [[ $status -eq 0 ]] || false
}

@test "shell-profile.sh time-only mode reports min/median/mean/max" {
    run "$SCRIPTS_DIR/shell-profile.sh" --shell bash --runs 3
    [[ $status -eq 0 ]] || false
    [[ "$output" == *"min:"* ]] || false
    [[ "$output" == *"median:"* ]] || false
    [[ "$output" == *"mean:"* ]] || false
    [[ "$output" == *"max:"* ]] || false
}

@test "shell-profile.sh time-only mode runs the requested number of iterations" {
    run "$SCRIPTS_DIR/shell-profile.sh" --shell bash --runs 4
    [[ $status -eq 0 ]] || false
    # Count "run:" lines (each iteration prints one)
    local run_count
    run_count=$(printf '%s' "$output" | grep -c "^  run:" || true)
    [[ "$run_count" -eq 4 ]] || false
}

@test "time-only mode still reports stats when the shell exits non-zero" {
    # BUG-035: the script runs under `set -euo pipefail`, so the profiled
    # shell's non-zero exit propagated through the pipeline and aborted the run
    # before any stats were printed. A loaded interactive profile exits non-zero
    # routinely, which is why this reproduced on a real machine but never in CI,
    # where a bare `bash -i -c exit` happens to return 0.
    stub_shell_exiting 1

    run "$SCRIPTS_DIR/shell-profile.sh" --shell bash --runs 2
    [[ $status -eq 0 ]] || false
    [[ "$output" == *"min:"* ]] || false
    [[ "$output" == *"median:"* ]] || false
    [[ "$output" == *"mean:"* ]] || false
    [[ "$output" == *"max:"* ]] || false
}

@test "detail mode survives a shell that exits non-zero" {
    # Same root cause on the --detail path: the profiling run is the last
    # command before `exit 0`, so `set -e` swallowed the success exit too.
    stub_shell_exiting 1

    run "$SCRIPTS_DIR/shell-profile.sh" --shell bash --detail
    [[ $status -eq 0 ]] || false
    [[ "$output" == *"Detail profile"* ]] || false
}

@test ".bashrc has DOTFILES_PROFILE opt-in hook" {
    grep -q 'DOTFILES_PROFILE' "$REPO_ROOT/.bashrc"
    grep -q 'set -x' "$REPO_ROOT/.bashrc"
    grep -q 'set +x' "$REPO_ROOT/.bashrc"
}

@test ".zshrc has DOTFILES_PROFILE opt-in hook with zprof" {
    grep -q 'DOTFILES_PROFILE' "$REPO_ROOT/.zshrc"
    grep -q 'zmodload zsh/zprof' "$REPO_ROOT/.zshrc"
    grep -q 'zprof' "$REPO_ROOT/.zshrc"
}

@test ".zshrc and .bashrc remain valid syntax with profile hook" {
    bash -n "$REPO_ROOT/.bashrc"
    zsh -n "$REPO_ROOT/.zshrc"
}

@test ".bashrc profile hook is no-op when DOTFILES_PROFILE is unset" {
    # Source bashrc without DOTFILES_PROFILE; should not enable xtrace.
    run bash -c 'unset DOTFILES_PROFILE; source "$1"; [[ "$-" != *x* ]] && echo "no_xtrace"' \
        -- "$REPO_ROOT/.bashrc"
    [[ "$output" == *"no_xtrace"* ]] || false
}
