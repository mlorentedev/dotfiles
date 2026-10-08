#!/usr/bin/env bats
# Real-dependency sibling of setup-linux-only-downloads.bats (BUG-055 pairing).
# The stub suite fakes `uname` to drive every host branch; this one checks the
# helpers against the host's real `uname` and a real executable, so a gate that
# only agrees with its own stub cannot pass.

# bats file_tags=os-sensitive

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    TMP="$(mktemp -d)"
    export HOME="$TMP/home"
    mkdir -p "$HOME/.local/bin"
}

teardown() {
    rm -rf "$TMP"
}

utils() {
    run bash -c '. "$1/scripts/utils.sh" >/dev/null 2>&1; shift; "$@"' _ "$REPO" "$@"
}

@test "host_is_linux_amd64 (real uname): agrees with this host" {
    expected=1
    if [ "$(uname -s)" = "Linux" ]; then
        case "$(uname -m)" in x86_64 | amd64) expected=0 ;; esac
    fi
    utils host_is_linux_amd64
    [ "$status" -eq "$expected" ]
}

@test "remove_unrunnable_tool (real binary): keeps a native executable that does not exit 126" {
    # A copy of the host's ls: GNU ls exits 0 on --version. On macOS a copied
    # system binary is killed on launch (137, and bash prints "Killed: 9"), and
    # BSD ls would exit 1 anyway. None of these is 126, so the file stays.
    cp "$(command -v ls)" "$HOME/.local/bin/ls"
    utils remove_unrunnable_tool ls
    [ "$status" -eq 0 ]
    [ -x "$HOME/.local/bin/ls" ]
    [[ "$output" != *"Removed"* ]] || false
}
