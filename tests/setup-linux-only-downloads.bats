#!/usr/bin/env bats
# setup-linux.sh fetches age, eza, jq, gh and shellcheck from hard-coded
# linux-amd64 URLs. On the macOS bring-up (#2013) it placed four ELF binaries in
# ~/.local/bin, logged SUCCESS for each, and the unconditional `ls=eza` alias then
# made `ls` exit 126 in every new shell. Interim, ahead of #2013 W2 deleting the
# blocks: they run on linux-amd64 only, other hosts remove such leftovers, and
# the eza aliases exist only when eza does.

# bats file_tags=os-sensitive

load 'lib/refute'

# A mid-test `[[ ]]` carries `|| false`: under bash < 4.1 (macOS /bin/bash 3.2,
# which runs this tier) a failing `[[ ]]` does not trip errexit, so without it
# the assertion is silently skipped (#2164).

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    TMP="$(mktemp -d)"
    export HOME="$TMP/home"
    mkdir -p "$HOME/.local/bin" "$TMP/bin"
}

teardown() {
    rm -rf "$TMP"
}

# $1 = uname -s, $2 = uname -m
fake_uname() {
    printf '#!/bin/sh\ncase "$1" in -s) echo %s ;; -m) echo %s ;; esac\n' "$1" "$2" > "$TMP/bin/uname"
    chmod +x "$TMP/bin/uname"
}

utils() {
    PATH="$TMP/bin:$PATH" run bash -c '. "$1/scripts/utils.sh" >/dev/null 2>&1; shift; "$@"' _ "$REPO" "$@"
}

@test "host_is_linux_amd64: true only for Linux on x86_64" {
    fake_uname Linux x86_64
    utils host_is_linux_amd64
    [ "$status" -eq 0 ]
    fake_uname Darwin arm64
    utils host_is_linux_amd64
    [ "$status" -eq 1 ]
    fake_uname Darwin x86_64
    utils host_is_linux_amd64
    [ "$status" -eq 1 ]
    fake_uname Linux aarch64
    utils host_is_linux_amd64
    [ "$status" -eq 1 ]
}

@test "remove_unrunnable_tool: removes a binary the OS cannot execute (exit 126)" {
    # A file of no known executable format is the portable stand-in for an ELF
    # on macOS: the kernel refuses it (ENOEXEC), bash sees binary content and
    # reports 126 on both OSes. A missing #! interpreter is not: Linux bash
    # reports that as 127.
    printf '\000\000\000\000not a program\n' > "$HOME/.local/bin/eza"
    chmod +x "$HOME/.local/bin/eza"
    utils remove_unrunnable_tool eza
    [ "$status" -eq 0 ]
    [ ! -e "$HOME/.local/bin/eza" ]
    [[ "$output" == *"Removed $HOME/.local/bin/eza"* ]] || false
}

@test "remove_unrunnable_tool: leaves a working tool, a failing tool, a non-executable file, a symlink and an absent name alone" {
    # A native file that lost its execute bit also makes bash exit 126 (EACCES);
    # that says nothing about its platform, so it stays.
    printf '\000\000\000\000not a program\n' > "$HOME/.local/bin/sops"
    chmod 644 "$HOME/.local/bin/sops"
    printf '#!/bin/sh\nexit 0\n' > "$HOME/.local/bin/jq"
    printf '#!/bin/sh\nexit 1\n' > "$HOME/.local/bin/gh"
    printf '\000\000\000\000not a program\n' > "$TMP/elsewhere"
    chmod +x "$HOME/.local/bin/jq" "$HOME/.local/bin/gh" "$TMP/elsewhere"
    ln -s "$TMP/elsewhere" "$HOME/.local/bin/pi"
    utils remove_unrunnable_tool jq gh pi age sops
    [ "$status" -eq 0 ]
    [ -f "$HOME/.local/bin/sops" ]
    [ -f "$HOME/.local/bin/jq" ]
    [ -f "$HOME/.local/bin/gh" ]
    [ -L "$HOME/.local/bin/pi" ]
    [ -f "$TMP/elsewhere" ]
    [ -z "$output" ]
}

@test "setup-linux.sh: every linux-amd64 release literal sits inside the host gate" {
    # g=1 inside `if host_is_linux_amd64; then` up to its column-0 else/fi.
    # Code lines only: comments and the skip warning name the literal as prose.
    run awk '
        /^if host_is_linux_amd64; then$/ { g = 1; gates++; next }
        g == 1 && /^(else|fi)$/ { g = 0 }
        /^[[:space:]]*#/ || /log_(warning|info)/ { next }
        /linux-amd64|x86_64-unknown-linux|linux\.x86_64|linux_amd64/ && g != 1 { print NR": "$0 }
        END { if (gates < 1) print "no gate found" }
    ' "$REPO/setup-linux.sh"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "aliases.zsh: ls stays the system ls when eza is absent, and is eza when present" {
    local zsh_bin
    zsh_bin="$(command -v zsh)" || skip "zsh not installed"
    # aliases.zsh runs nothing at load but the `command -v` builtin, so PATH can
    # hold exactly the eza under test. A system dir would measure the host: apt
    # installs eza into /usr/bin (#2194).
    mkdir -p "$TMP/empty"
    run env PATH="$TMP/empty" "$zsh_bin" -f -c '. "$1"; alias ls; whence -w ll' _ "$REPO/.zsh/aliases.zsh"
    [[ "$output" != *eza* ]] || false
    [[ "$output" == *"ll: alias"* ]] || false
    printf '#!/bin/sh\nexit 0\n' > "$TMP/bin/eza"
    chmod +x "$TMP/bin/eza"
    run env PATH="$TMP/bin" "$zsh_bin" -f -c '. "$1"; alias ls' _ "$REPO/.zsh/aliases.zsh"
    [[ "$output" == *"eza --group-directories-first"* ]] || false
}
