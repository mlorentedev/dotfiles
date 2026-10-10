#!/usr/bin/env bats
# #2055: the #2013 W6 hardening of .zshrc, proved by sourcing the real file.
# Every optional piece is guarded on what it needs: oh-my-zsh on its install
# (with a compinit fallback), each *_HOME on its directory, brew shellenv on
# brew, terraform completion on the terraform that PATH resolves. A grep for
# the guard text would pass on a guard that no longer works, so each test runs
# zsh on the rc in a scratch HOME with a scrubbed environment and asserts what
# the shell ends up with.
#
# Every mid-test `[[ ]]` ends in `|| false`: under bash 3.2 (macOS's /bin/bash) a
# failing `[[ ]]` does not trip errexit, so only the last one would count.
# shellcheck disable=SC2016  # the probes are zsh code, expanded by the zsh under test

setup() {
    ZSH_BIN=$(command -v zsh) || skip "zsh not available"
    REPO="$BATS_TEST_DIRNAME/.."
    SANDBOX="$BATS_TEST_TMPDIR/sandbox"
    mkdir -p "$SANDBOX/home" "$SANDBOX/dotfiles" "$SANDBOX/bin"
    # versions.conf names the tool-home directories; paths.sh is left absent so
    # the rc takes its bootstrap fallback, as on a machine that has not run setup.
    cp "$REPO/versions.conf" "$SANDBOX/dotfiles/"
    # The rc's PATH holds the test's own stubs and links to RC_TOOLS, never a
    # host directory: with /usr/bin on it, a host terraform (or direnv, mise,
    # zoxide, dotf) switches on a block the test assumes is off (#2149). A tool
    # the rc needs and RC_TOOLS lacks fails the load through its "command not
    # found" on stderr, unless the rc silences that call's stderr.
    mkdir -p "$SANDBOX/sysbin"
    for tool in $RC_TOOLS; do
        target=$(command -v "$tool") || continue
        ln -s "$target" "$SANDBOX/sysbin/$tool"
    done
}

# The external commands the rc runs whatever the host has: compinit's dump is
# renamed into place with mv.
RC_TOOLS="mv"

# load_rc PROBE: source the repo's .zshrc in a scratch HOME, then evaluate PROBE
# in the same shell. stdout is the probe's. Anything the rc writes to stderr
# fails the load: zsh reports the probe's status, so an rc that errors midway
# would otherwise still exit 0 in every test. HOMEBREW_PREFIX reads as already
# loaded, since the rc runs brew by absolute path and a PATH cannot hide it;
# LOAD_RC_BREW="" unsets it for the brew test.
load_rc() {
    env -i HOME="$SANDBOX/home" ZDOTDIR="$SANDBOX/home" DOTFILES_DIR="$SANDBOX/dotfiles" \
        PATH="$SANDBOX/bin:$SANDBOX/sysbin" TERM=dumb \
        HOMEBREW_PREFIX="${LOAD_RC_BREW-$SANDBOX/brew-loaded}" \
        "$ZSH_BIN" -f -c '. "$1"; eval "$2"' _ "$REPO/.zshrc" "$1" 2>"$SANDBOX/stderr" || return
    if [ -s "$SANDBOX/stderr" ]; then
        printf 'stderr: %s\n' "$(cat "$SANDBOX/stderr")"
        return 1
    fi
}

@test "loads without errors in a HOME with no oh-my-zsh, no tool homes and no terraform" {
    run load_rc 'print -r -- "homes=${JAVA_HOME-}${MAVEN_HOME-}${PYTHON_HOME-}${MINIKUBE_HOME-}${GO_HOME-} compdef=$(whence -w compdef) complete=$(whence -w complete)"; print -rl -- $path'
    [ "$status" -eq 0 ]
    # No *_HOME names a directory that is not there, and none reaches PATH.
    [[ "$output" == *"homes= "* ]] || false
    [[ "$output" != *"$SANDBOX/home/Applications/"* ]] || false
    # The compinit fallback ran in place of oh-my-zsh.
    [[ "$output" == *"compdef=compdef: function"* ]] || false
    # bashcompinit is loaded only for terraform.
    [[ "$output" == *"complete=complete: none"* ]] || false
}

@test "the sandbox resolves none of the optional tools the rc probes for" {
    # Every `command -v X` in the rc is a block a test assumes is off unless it
    # stubs X. Read them from the rc, so a new probe is covered without an edit
    # here, and fail on the first that resolves to anything on the host (#2149).
    # A probe whose name is a variable cannot be read statically, so it fails
    # here rather than drop out of the sweep.
    local rc_probes='command -v[[:space:]]+["'"'"']?[$A-Za-z0-9_.-]+'
    probes=$(grep -oE "$rc_probes" "$REPO/.zshrc" | sed -E 's/^command -v[[:space:]]+["'"'"']?//' | sort -u | tr '\n' ' ')
    [ -n "$probes" ] || false
    [[ "$probes" != *'$'* ]] || { printf 'a probe names its tool through a variable: %s\n' "$probes"; false; }
    run load_rc 'for t in '"$probes"'; do whence -p $t >/dev/null && print -r -- "leak: $t -> $(whence -p $t)"; done; true'
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "brew shellenv runs exactly when brew is installed and not yet loaded" {
    # Already loaded: a nested shell inherits the prefix and its PATH, and the rc
    # does not run brew again.
    run load_rc 'print -rl -- $path'
    [ "$status" -eq 0 ]
    [[ "$output" != *homebrew* ]] || false
    LOAD_RC_BREW="" run load_rc 'print -rl -- $path'
    [ "$status" -eq 0 ]
    # Which direction runs depends on the host, since the rc probes brew's two
    # absolute install paths: the Linux job proves the absent side, a Mac with
    # Homebrew the present side.
    for brew in /opt/homebrew/bin/brew /usr/local/bin/brew; do
        if [ -x "$brew" ]; then
            [[ "$output" == *$'\n'"${brew%/brew}"* ]] || false
            return
        fi
    done
    [[ "$output" != *homebrew* ]] || false
}

@test "a tool home that exists is exported and put on PATH" {
    # shellcheck disable=SC1091
    . "$REPO/versions.conf"
    mkdir -p "$SANDBOX/home/Applications/jdk-$JAVA_VERSION" "$SANDBOX/home/Applications/go-$GO_VERSION"
    run load_rc 'print -r -- "$JAVA_HOME|$GO_HOME|${MAVEN_HOME-unset}"; print -rl -- $path'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "$SANDBOX/home/Applications/jdk-$JAVA_VERSION|$SANDBOX/home/Applications/go-$GO_VERSION|unset" ]
    [[ "$output" == *$'\n'"$SANDBOX/home/Applications/jdk-$JAVA_VERSION/bin"* ]] || false
    [[ "$output" == *$'\n'"$SANDBOX/home/Applications/go-$GO_VERSION/bin"* ]] || false
}

@test "oh-my-zsh is sourced when it is installed" {
    mkdir -p "$SANDBOX/home/.oh-my-zsh"
    printf 'OMZ_LOADED=yes\n' > "$SANDBOX/home/.oh-my-zsh/oh-my-zsh.sh"
    run load_rc 'print -r -- "${OMZ_LOADED-no}"'
    [ "$status" -eq 0 ]
    [ "$output" = "yes" ]
}

@test "terraform completion calls the terraform that PATH resolves" {
    printf '#!/bin/sh\n' > "$SANDBOX/bin/terraform"
    chmod +x "$SANDBOX/bin/terraform"
    run load_rc 'print -r -- "${_comps[terraform]-unset}"'
    [ "$status" -eq 0 ]
    [[ "$output" == *"-C $SANDBOX/bin/terraform" ]] || false
}
