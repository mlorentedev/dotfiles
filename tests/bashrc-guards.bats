#!/usr/bin/env bats
# #2013 W6, the bash half of what tests/zshrc-guards.bats proves for zsh: the
# rc only ever prepends to PATH, and each *_HOME is exported only when its
# directory exists. Each test sources the real .bashrc in an interactive bash
# (the rc returns early otherwise) with a scratch HOME and a scrubbed
# environment, and asserts what the shell ends up with. A grep for the guard
# text would pass on a guard that no longer works.
#
# Every mid-test `[[ ]]` ends in `|| false`: under bash 3.2 (macOS's /bin/bash) a
# failing `[[ ]]` does not trip errexit, so only the last one would count.
# shellcheck disable=SC2016  # the probes are bash code, expanded by the bash under test

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SANDBOX="$BATS_TEST_TMPDIR/sandbox"
    mkdir -p "$SANDBOX/home" "$SANDBOX/dotfiles" "$SANDBOX/bin" "$SANDBOX/inherited"
    cp "$REPO/versions.conf" "$SANDBOX/dotfiles/"
}

# load_rc PROBE: source the repo's .bashrc in an interactive bash with a scratch
# HOME, then evaluate PROBE in the same shell. PATH holds a directory standing
# for what the login shell or the OS put there, then the system directories the
# rc's own tools live in. An interactive bash without a terminal reports that it
# has no job control on stderr; that notice is the harness's, not the rc's, and
# is dropped. Anything else on stderr fails the load.
load_rc() {
    env -i HOME="$SANDBOX/home" DOTFILES_DIR="$SANDBOX/dotfiles" \
        PATH="$SANDBOX/bin:$SANDBOX/inherited:/usr/bin:/bin" TERM=dumb \
        bash --norc --noprofile -i -c '. "$1"; eval "$2"' _ "$REPO/.bashrc" "$1" \
        2>"$SANDBOX/stderr" </dev/null || return
    grep -vE 'job control|terminal process group' "$SANDBOX/stderr" >"$SANDBOX/stderr.rc" || true
    if [ -s "$SANDBOX/stderr.rc" ]; then
        printf 'stderr: %s\n' "$(cat "$SANDBOX/stderr.rc")"
        return 1
    fi
}

@test "the inherited PATH survives the rc: it is prepended to, never replaced" {
    run load_rc 'printf "%s\n" "$PATH"'
    [ "$status" -eq 0 ]
    [[ "$output" == *":$SANDBOX/inherited:"* ]] || false
    # The user's own bin directory is still first among what the rc adds.
    [[ "$output" == "$SANDBOX/home/.local/bin:"* || "$output" == *":$SANDBOX/home/.local/bin:"* ]] || false
}

@test "no *_HOME names a directory that is not there, and none reaches PATH" {
    run load_rc 'printf "homes=%s\n%s\n" "${JAVA_HOME-}${MAVEN_HOME-}${MINIKUBE_HOME-}${GO_HOME-}" "$PATH"'
    [ "$status" -eq 0 ]
    [[ "$output" == *"homes="$'\n'* ]] || false
    [[ "$output" != *"$SANDBOX/home/Applications/"* ]] || false
}

@test "a tool home that exists is exported and put on PATH" {
    # shellcheck disable=SC1091  # versions.conf is the repo's, read for the pin
    jdk="$SANDBOX/home/Applications/jdk-$(. "$REPO/versions.conf"; printf '%s' "$JAVA_VERSION")"
    mkdir -p "$jdk/bin"
    run load_rc 'printf "java=%s\n%s\n" "${JAVA_HOME-}" "$PATH"'
    [ "$status" -eq 0 ]
    [[ "$output" == *"java=$jdk"$'\n'* ]] || false
    [[ "$output" == *"$jdk/bin:"* ]] || false
}
