#!/usr/bin/env bats
# #2375: the PowerShell tests gate on a pwsh that runs, not on a file named
# pwsh (tests/lib/pwsh.bash). These pin the gate's three answers and keep the
# old `command -v pwsh` probe from coming back.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    BIN="$BATS_TEST_TMPDIR/bin"
    mkdir -p "$BIN"
}

# Runs require_pwsh with PATH holding only $BIN and the shell basics, and a
# skip that reports and ends the subshell, as bats' skip ends a test.
gate() {
    run env -i PATH="$BIN:/usr/bin:/bin" DOTFILES_CI_EXPECT_PWSH="${1:-}" bash -c '
        skip() { printf "SKIP: %s\n" "$*"; exit 0; }
        . "$0"
        require_pwsh && echo PASS
    ' "$REPO/tests/lib/pwsh.bash"
}

@test "require_pwsh skips when pwsh is on PATH but does not run (a dangling mise shim)" {
    printf '#!/bin/sh\necho "mise ERROR No version is set for shim: pwsh" >&2\nexit 1\n' > "$BIN/pwsh"
    chmod +x "$BIN/pwsh"
    gate
    [ "$status" -eq 0 ] || false
    [[ "$output" == "SKIP: pwsh is on PATH but does not run: mise ERROR No version is set for shim: pwsh" ]] || { echo "$output"; false; }
}

@test "require_pwsh passes a pwsh that runs" {
    printf '#!/bin/sh\nexit 0\n' > "$BIN/pwsh"
    chmod +x "$BIN/pwsh"
    gate
    [ "$status" -eq 0 ] && [ "$output" = PASS ] || { echo "$output"; false; }
}

@test "require_pwsh fails instead of skipping where the job promises pwsh" {
    printf '#!/bin/sh\nexit 1\n' > "$BIN/pwsh"
    chmod +x "$BIN/pwsh"
    gate 1
    [ "$status" -ne 0 ] || { echo "$output"; false; }
    [[ "$output" == *"DOTFILES_CI_EXPECT_PWSH is set and pwsh is on PATH but does not run"* ]] || { echo "$output"; false; }
    [[ "$output" != *SKIP* ]] || false
}

@test "no bats file gates on command -v pwsh" {
    # This file names the pattern it forbids, so it is the one exception.
    run grep -ln 'command -v pwsh' "$REPO"/tests/*.bats
    offenders=$(printf '%s\n' "$output" | grep -v '/bats-pwsh-gate\.bats$' || true)
    [ -z "$offenders" ] || { echo "use require_pwsh (tests/lib/pwsh.bash): $offenders"; false; }
}

@test "the CI steps that run the PowerShell bats promise pwsh" {
    for step in 'Run bats test suite' 'Run PowerShell bats subset'; do
        body="$(awk -v s="$step" 'index($0, "- name: " s){f=1; print; next} f && /^      - /{exit} f' "$REPO/.github/workflows/ci.yml")"
        [ -n "$body" ] || { echo "no step: $step"; false; }
        printf '%s\n' "$body" | grep -q "DOTFILES_CI_EXPECT_PWSH: '1'" || { echo "$step does not set DOTFILES_CI_EXPECT_PWSH"; false; }
    done
}
