#!/usr/bin/env bats
# Real-pwsh coverage for tests/bats-pwsh-gate.bats, which stubs pwsh to pin the
# gate's three answers. Here the gate meets whatever pwsh this machine has: a
# working one on the CI runners, a dangling mise shim or none on a dev box.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

# The machine's own answer, asked without the gate: does pwsh start?
pwsh_runs() {
    command -v pwsh >/dev/null 2>&1 &&
        pwsh -NoProfile -NonInteractive -Command 'exit 0' >/dev/null 2>&1 </dev/null
}

gate() {
    run env DOTFILES_CI_EXPECT_PWSH="${1:-}" bash -c '
        skip() { printf "SKIP: %s\n" "$*"; exit 0; }
        . "$0"
        require_pwsh && echo PASS
    ' "$REPO/tests/lib/pwsh.bash"
}

@test "require_pwsh agrees with the real pwsh on this machine" {
    gate
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    if pwsh_runs; then
        [ "$output" = PASS ] || { echo "pwsh runs here, gate said: $output"; false; }
    else
        [[ "$output" == "SKIP: pwsh "* ]] || { echo "pwsh does not run here, gate said: $output"; false; }
    fi
}

@test "a promised pwsh passes where it runs and fails where it does not" {
    gate 1
    if pwsh_runs; then
        [ "$status" -eq 0 ] && [ "$output" = PASS ] || { echo "$output"; false; }
    else
        [ "$status" -ne 0 ] || { echo "$output"; false; }
        [[ "$output" == *"DOTFILES_CI_EXPECT_PWSH is set and pwsh "* ]] || { echo "$output"; false; }
    fi
}
