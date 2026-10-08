#!/usr/bin/env bats
# scripts/setup-mock-env.sh builds the ~/.dotfiles mock the CI suite runs
# against, for the Linux `test` and the `test-macos` jobs (#2152).
#
# The inline step it replaced discarded age-keygen's stderr and ended the
# fixture copy in `2>/dev/null || true`. The suite skips on a missing key or
# fixture, so a setup failure became an absence and the leg read as a pass.
# Each failure below must now exit non-zero with an ::error:: annotation.
#
# PATH holds only the tools the script needs plus a stub age-keygen, so the
# result does not depend on whether the host has age installed.

load 'lib/refute'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SCRIPT="$REPO/scripts/setup-mock-env.sh"
    BASH_BIN="$(command -v bash)"
    TMP="$(mktemp -d)"
    export HOME="$TMP/home"
    mkdir -p "$HOME" "$TMP/bin" "$TMP/root/scripts" "$TMP/root/sensitive/dr"
    for tool in mkdir cp find; do
        ln -s "$(command -v "$tool")" "$TMP/bin/$tool"
    done
    printf 'AGE_VERSION=1.3.1\n' > "$TMP/root/versions.conf"
    printf 'echo hi\n' > "$TMP/root/scripts/utils.sh"
    printf 'cipher\n' > "$TMP/root/sensitive/id.secret.age"
    printf 'nested\n' > "$TMP/root/sensitive/dr/escrow.age"
}

teardown() {
    rm -rf "$TMP"
}

# $1: the stub's body after the shebang. `$2` inside it is the -o path.
stub_keygen() {
    printf '#!%s\n%s\n' "$BASH_BIN" "$1" > "$TMP/bin/age-keygen"
    chmod +x "$TMP/bin/age-keygen"
}

run_script() {
    cd "$TMP/root"
    PATH="$TMP/bin" run "$BASH_BIN" "$SCRIPT"
}

@test "setup-mock-env: builds the key, versions.conf and both trees" {
    stub_keygen 'printf "AGE-SECRET-KEY-1STUB\n" > "$2"'
    run_script
    [ "$status" -eq 0 ]
    [ -s "$HOME/.config/age/key.txt" ]
    [ -f "$HOME/.dotfiles/versions.conf" ]
    [ -f "$HOME/.dotfiles/scripts/utils.sh" ]
    [ -f "$HOME/.dotfiles/sensitive/id.secret.age" ]
    [ -f "$HOME/.dotfiles/sensitive/dr/escrow.age" ]
}

@test "setup-mock-env: a missing age-keygen fails the step, never a silent skip" {
    run_script
    [ "$status" -eq 1 ]
    [[ "$output" == *"::error::setup-mock-env: age-keygen is not on PATH"* ]]
    [ ! -e "$HOME/.config/age/key.txt" ]
}

@test "setup-mock-env: age-keygen exiting 0 without writing a key still fails" {
    stub_keygen 'exit 0'
    run_script
    [ "$status" -eq 1 ]
    [[ "$output" == *"left no key at $HOME/.config/age/key.txt"* ]]
}

@test "setup-mock-env: a failing age-keygen fails the step with its stderr visible" {
    stub_keygen 'echo "keygen exploded" >&2; exit 3'
    run_script
    [ "$status" -eq 1 ]
    [[ "$output" == *"keygen exploded"* ]]
    [[ "$output" == *"::error::setup-mock-env: age-keygen could not write"* ]]
}

@test "setup-mock-env: a failing copy fails the step with an annotation" {
    stub_keygen 'printf "AGE-SECRET-KEY-1STUB\n" > "$2"'
    # A copy that cannot land: the scripts/ destination is a file, not a dir.
    mkdir -p "$HOME/.dotfiles"
    printf 'not a dir\n' > "$HOME/.dotfiles/scripts"
    run_script
    [ "$status" -eq 1 ]
    [[ "$output" == *"::error::setup-mock-env:"* ]]
}

@test "setup-mock-env: an empty sensitive/ fails the step instead of copying nothing" {
    stub_keygen 'printf "AGE-SECRET-KEY-1STUB\n" > "$2"'
    rm -rf "$TMP/root/sensitive"
    mkdir "$TMP/root/sensitive"
    run_script
    [ "$status" -eq 1 ]
    [[ "$output" == *"/.dotfiles/sensitive is empty after the copy"* ]]
}

@test "setup-mock-env: outside the repository root it refuses before touching HOME" {
    stub_keygen 'printf "AGE-SECRET-KEY-1STUB\n" > "$2"'
    rm "$TMP/root/versions.conf"
    run_script
    [ "$status" -eq 1 ]
    [[ "$output" == *"run from the repository root"* ]]
    [ ! -e "$HOME/.dotfiles" ]
}

@test "setup-mock-env: a rerun keeps the existing key" {
    stub_keygen 'printf "AGE-SECRET-KEY-1FIRST\n" > "$2"'
    run_script
    [ "$status" -eq 0 ]
    stub_keygen 'printf "AGE-SECRET-KEY-1SECOND\n" > "$2"'
    run_script
    [ "$status" -eq 0 ]
    grep -q 'FIRST' "$HOME/.config/age/key.txt"
}

@test "setup-mock-env: discards no error of its own" {
    # Code lines only: the header quotes the step it replaced. grep exits 1 on
    # no match; 2 (an error) must not pass as clean, and neither may a missing
    # script, which the inner grep would turn into an empty, "clean" input.
    [ -s "$SCRIPT" ]
    run grep -nE '2>/dev/null|\|\|[[:space:]]*true' <(grep -vE '^[[:space:]]*#' "$SCRIPT")
    [ "$status" -eq 1 ]
}

@test "ci: both mock-environment steps run the shared script and nothing else" {
    # Two copies of an inline step is how the Linux job's silencers reached
    # test-macos in the first place (#2147).
    run grep -c -E '^[[:space:]]+- name: Setup mock environment$' "$REPO/.github/workflows/ci.yml"
    [ "$output" = "2" ]
    run grep -A4 -E '^[[:space:]]+- name: Setup mock environment$' "$REPO/.github/workflows/ci.yml"
    [ "$(printf '%s\n' "$output" | grep -c -E '^[[:space:]]+run: \./scripts/setup-mock-env\.sh$')" = "2" ]
    # The inline form, not every age-keygen: test-windows makes its own key.
    refute_grep_fixed 'age-keygen -o ~/.config/age/key.txt' "$REPO/.github/workflows/ci.yml"
}

@test "ci: no step discards a command's error with 2>/dev/null || true (#2152)" {
    # The form that turned a broken mock into skipped tests. Measured absent from
    # ci.yml once the mock step moved to the script; this keeps it absent. A step
    # that genuinely tolerates a failure says so with an explicit test instead.
    refute_grep '2>/dev/null[[:space:]]*\|\|[[:space:]]*true' "$REPO/.github/workflows/ci.yml"
}
