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
    # LOAD_RC_ENV: extra NAME=value words for the scrubbed environment (no spaces).
    # shellcheck disable=SC2086
    env -i HOME="$SANDBOX/home" ZDOTDIR="$SANDBOX/home" DOTFILES_DIR="$SANDBOX/dotfiles" $LOAD_RC_ENV \
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

# DX-005: the dump rule. compinit -C trusts the dump and never scans fpath, so a
# completion added after it is invisible until a full run rebuilds it. COMPDIR
# holds a `foo` completion that is told apart by _comps[foo]: 1 when compinit
# scanned it, 0 when it trusted a dump that predates it.
comp_setup() {
    COMPDIR="$SANDBOX/home/.zsh/completions"
    DUMP="$SANDBOX/home/.zcompdump"
    mkdir -p "$COMPDIR"
    run load_rc 'true'              # no dump yet: a full run writes one
    [ "$status" -eq 0 ] || false
    [ -s "$DUMP" ] || false
}

add_foo_completion() {
    printf '#compdef foo\n_foo() { :; }\n' > "$COMPDIR/_foo"
}

@test "a fresh dump is trusted: a completion older than it is not scanned (dump)" {
    comp_setup
    add_foo_completion
    touch -t 202001010000 "$COMPDIR/_foo"
    run load_rc 'print -r -- "foo=${+_comps[foo]}"'
    [ "$status" -eq 0 ] || false
    [ "$output" = "foo=0" ] || false
}

@test "a dump older than a day is rebuilt (dump)" {
    comp_setup
    add_foo_completion
    touch -t 202001010000 "$COMPDIR/_foo" "$DUMP"
    run load_rc 'print -r -- "foo=${+_comps[foo]}"'
    [ "$status" -eq 0 ] || false
    [ "$output" = "foo=1" ] || false
}

@test "an insecure completion directory neither prompts nor blocks the rebuild (dump)" {
    # Stock Debian and Ubuntu (and GitHub's runners) have a group-writable
    # site-functions on fpath; a plain compinit warns on stderr and asks.
    comp_setup
    chmod 777 "$COMPDIR"
    add_foo_completion
    touch -t 202001010000 "$DUMP"
    run load_rc 'print -r -- "foo=${+_comps[foo]}"'
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    [ "$output" = "foo=1" ] || false
}

@test "a completion newer than a fresh dump rebuilds it (dump)" {
    comp_setup
    sleep 1
    add_foo_completion
    run load_rc 'print -r -- "foo=${+_comps[foo]}"'
    [ "$status" -eq 0 ] || false
    [ "$output" = "foo=1" ] || false
}

@test "~/.zsh/completions leads fpath and fpath holds no duplicate (fpath)" {
    mkdir -p "$SANDBOX/home/.zsh/completions"
    run load_rc 'print -rl -- $fpath'
    [ "$status" -eq 0 ] || false
    [ "${lines[0]}" = "$SANDBOX/home/.zsh/completions" ] || false
    dups=$(printf '%s\n' "${lines[@]}" | sort | uniq -d)
    [ -z "$dups" ] || { echo "duplicate fpath entries: $dups"; false; }
}

@test "the Homebrew block runs before compinit, so its site-functions are in the dump (fpath)" {
    brew_line=$(grep -n 'shellenv' "$REPO/.zshrc" | head -1 | cut -d: -f1)
    compinit_line=$(grep -n '^[[:space:]]*if \[\[ -n "\$_zcfull" \]\]' "$REPO/.zshrc" | head -1 | cut -d: -f1)
    [ -n "$brew_line" ] && [ -n "$compinit_line" ] || false
    [ "$brew_line" -lt "$compinit_line" ] || false
}

@test "completion menus are selectable (menu)" {
    run load_rc 'zstyle -L ":completion:*" menu'
    [ "$status" -eq 0 ] || false
    [[ "$output" == *"menu select"* ]] || false
}

# DX-007: the plugin mise installs is sourced from mise's data dir, after
# compinit, and its absence costs nothing and prints nothing.
plugin_stub() {
    stub_dir="$1/installs/http-zsh-autosuggestions/latest"
    mkdir -p "$stub_dir"
    printf 'ZAS_LOADED=1\nZAS_AFTER_COMPINIT=${+_comps}\n' > "$stub_dir/zsh-autosuggestions.zsh"
}

@test "zsh-autosuggestions is sourced from mise's default data dir, after compinit (plugin)" {
    plugin_stub "$SANDBOX/home/.local/share/mise"
    run load_rc 'print -r -- "loaded=${ZAS_LOADED-no} after=${ZAS_AFTER_COMPINIT-no}"'
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    [ "$output" = "loaded=1 after=1" ] || { echo "$output"; false; }
}

@test "MISE_DATA_DIR and XDG_DATA_HOME move where the plugin is looked for (plugin)" {
    plugin_stub "$SANDBOX/elsewhere/mise"
    LOAD_RC_ENV="MISE_DATA_DIR=$SANDBOX/elsewhere/mise" run load_rc 'print -r -- "loaded=${ZAS_LOADED-no}"'
    [ "$output" = "loaded=1" ] || { echo "MISE_DATA_DIR: $output"; false; }
    plugin_stub "$SANDBOX/xdg/mise"
    LOAD_RC_ENV="XDG_DATA_HOME=$SANDBOX/xdg" run load_rc 'print -r -- "loaded=${ZAS_LOADED-no}"'
    [ "$output" = "loaded=1" ] || { echo "XDG_DATA_HOME: $output"; false; }
}

@test "an rc without the plugin loads clean and sources nothing (plugin)" {
    run load_rc 'print -r -- "loaded=${ZAS_LOADED-no}"'
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    [ "$output" = "loaded=no" ] || false
}

@test "a plugin another loader already defined is not sourced twice (plugin)" {
    plugin_stub "$SANDBOX/home/.local/share/mise"
    # The probe runs after the rc, so define the function in a wrapper rc.
    printf '_zsh_autosuggest_start() { :; }\n. "%s/.zshrc"\n' "$REPO" > "$SANDBOX/wrap.zsh"
    env -i HOME="$SANDBOX/home" ZDOTDIR="$SANDBOX/home" DOTFILES_DIR="$SANDBOX/dotfiles" \
        PATH="$SANDBOX/bin:$SANDBOX/sysbin" TERM=dumb HOMEBREW_PREFIX="$SANDBOX/brew-loaded" \
        "$ZSH_BIN" -f -c '. "$1"; print -r -- "loaded=${ZAS_LOADED-no}"' _ "$SANDBOX/wrap.zsh" > "$SANDBOX/out" 2>"$SANDBOX/err"
    [ ! -s "$SANDBOX/err" ] || { cat "$SANDBOX/err"; false; }
    [ "$(cat "$SANDBOX/out")" = "loaded=no" ] || { cat "$SANDBOX/out"; false; }
}
