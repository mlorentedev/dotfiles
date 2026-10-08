#!/usr/bin/env bats
# Real-dependency sibling of setup-mock-env.bats (BUG-055 pairing): the stub
# suite proves each failure path, this one proves the script works with the
# age-keygen CI actually installs, against this repository's real tree.
#
# A missing age-keygen is a skip on a developer machine and a failure in CI,
# where both jobs install the pinned age before the step runs.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    if ! age-keygen --version >/dev/null 2>&1; then
        [ -z "${CI:-}" ] || { echo "age-keygen must work in CI" >&2; return 1; }
        skip "age-keygen not runnable here"
    fi
    TMP="$(mktemp -d)"
    export HOME="$TMP/home"
    mkdir -p "$HOME"
}

teardown() {
    [ -z "${TMP:-}" ] || rm -rf "$TMP"
}

@test "setup-mock-env (real age): writes a usable identity and mirrors the repo" {
    cd "$REPO"
    run ./scripts/setup-mock-env.sh
    [ "$status" -eq 0 ]
    key="$HOME/.config/age/key.txt"
    grep -q '^AGE-SECRET-KEY-1' "$key"
    # The identity yields a recipient, so it is a key age can use, not just text.
    run age-keygen -y "$key"
    [ "$status" -eq 0 ]
    [[ "$output" == age1* ]] || false
    [ -f "$HOME/.dotfiles/versions.conf" ]
    [ -f "$HOME/.dotfiles/scripts/utils.sh" ]
    [ -f "$HOME/.dotfiles/sensitive/README.md" ]
}
