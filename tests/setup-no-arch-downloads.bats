#!/usr/bin/env bats
# setup-linux.sh used to fetch age, eza, jq, gh and shellcheck from hard-coded
# linux-amd64 URLs. On the macOS bring-up (#2013 F-030) it placed four ELF
# binaries in ~/.local/bin, logged SUCCESS for each, and the unconditional
# `ls=eza` alias then made `ls` exit 126 in every new shell. The blocks are gone
# (#2013 W2): age, jq and shellcheck are mise pins, eza and gh are `system`
# entries in packages.json, installed through the OS manager. The guard below is
# #2013 X2: an installer may not name one OS/arch's release asset again.

# bats file_tags=os-sensitive

# A mid-test `[[ ]]` carries `|| false`: under bash < 4.1 (macOS /bin/bash 3.2,
# which runs this tier) a failing `[[ ]]` does not trip errexit, so without it
# the assertion is silently skipped (#2164).

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    TMP="$(mktemp -d)"
    mkdir -p "$TMP/bin"
}

teardown() {
    rm -rf "$TMP"
}

# The installers a machine runs: the entrypoints and everything they source.
# CI's workflows and tests/Dockerfile.integration are not in the list: they run
# on GitHub's ubuntu amd64 runners by construction, so an amd64 asset there is
# the right one. A tool every OS needs goes in packages.json or versions.conf,
# which name an asset per OS and arch.
installers() {
    printf '%s\n' "$REPO/setup-linux.sh" "$REPO/setup-windows.ps1" "$REPO/install.sh" "$REPO/install.ps1" "$REPO"/scripts/*.sh
}

# Code lines only: a comment names the literal as prose. A literal is an OS
# joined to an arch, either way round, in any of the separators release assets
# use: linux-amd64, linux_x86_64, x86_64-unknown-linux-gnu, darwin-arm64,
# aarch64-apple-darwin. An arch alone (`x86_64 | amd64)` in install.sh's
# detection) is not an asset name, so it passes.
arch_literals() {
    awk '
        /^[[:space:]]*#/ { next }
        {
            l = tolower($0)
            if (l ~ /(linux|darwin|macos|windows)[-_.](amd64|x86_64|x64|arm64|aarch64)/ ||
                l ~ /(amd64|x86_64|arm64|aarch64)[-_.](unknown[-_.])?(linux|apple|darwin|pc[-_.]windows)/)
                print FILENAME":"FNR": "$0
        }
    ' "$@"
}

@test "no installer names one OS-arch's release asset (#2013 X2)" {
    local files=() f
    while IFS= read -r f; do
        # Every named installer must exist: a rename would otherwise drop it
        # from the scan while the guard stayed green.
        [ -f "$f" ] || { echo "installer missing: $f"; return 1; }
        files+=("$f")
    done < <(installers)
    run arch_literals "${files[@]}"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "the X2 guard sees an OS-arch asset on a code line, in any spelling, and ignores prose and arch detection" {
    printf '#!/bin/sh\n# fetch foo_linux_amd64.tar.gz, as prose\n' > "$TMP/prose.sh"
    run arch_literals "$TMP/prose.sh"
    [ -z "$output" ]
    local asset
    for asset in foo-linux-amd64.tar.gz foo_Linux_x86_64.tar.gz eza_x86_64-unknown-linux-gnu.tar.gz \
        foo-darwin-arm64.zip foo-aarch64-apple-darwin.tar.gz foo_windows_amd64.zip; do
        printf '#!/bin/sh\ncurl -LO https://x/%s\n' "$asset" > "$TMP/code.sh"
        run arch_literals "$TMP/code.sh"
        [[ "$output" == *"code.sh:2:"* ]] || { echo "missed: $asset"; false; }
    done
    printf 'case "$(uname -m)" in x86_64 | amd64) arch=amd64 ;; esac\n' > "$TMP/detect.sh"
    run arch_literals "$TMP/detect.sh"
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
