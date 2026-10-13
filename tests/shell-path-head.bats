#!/usr/bin/env bats
# #2013 W2b: a shell that reads no interactive rc never runs `mise activate`,
# so mise's shims at the head of its startup PATH are the only way it runs the
# pins. .zsh/path.sh puts them there, and .zshenv (zsh -c), .profile (bash -lc)
# and the top of .bashrc (bash run by sshd for a command) source it. Each test
# starts the real shell from an empty environment with a scratch HOME holding
# the repo's files, and asserts the PATH it builds: a grep for the source line
# would pass on a line that no longer runs.
#
# Every mid-test `[[ ]]` ends in `|| false`: under bash 3.2 (macOS's /bin/bash) a
# failing `[[ ]]` does not trip errexit, so only the last one would count.
# shellcheck disable=SC2016  # the probes are shell code, expanded by the shell under test

# bash -lc reads /etc/profile, which differs by OS (path_helper on macOS).
# bats file_tags=os-sensitive

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    H="$BATS_TEST_TMPDIR/home"
    mkdir -p "$H/.zsh" "$H/.local/bin"
    cp "$REPO/.zsh/path.sh" "$H/.zsh/path.sh"
    SHIMS="$H/.local/share/mise/shims"
    LOCAL="$H/.local/bin"
}

# head_of PATH: the first two entries, colon-joined.
head_of() {
    printf '%s\n' "$1" | awk -F: '{ print $1 ":" $2 }'
}

# The sh, bash and zsh (and dash, where present) a target may run as /bin/sh.
shells() {
    for s in sh bash zsh dash; do
        command -v "$s" >/dev/null 2>&1 && printf '%s\n' "$s"
    done
}

@test "path.sh puts the shims, then ~/.local/bin, first in every POSIX shell, once" {
    for s in $(shells); do
        run env -i HOME="$H" PATH="$LOCAL:/usr/bin:$SHIMS:/bin" "$s" -c \
            '. "$HOME/.zsh/path.sh"; . "$HOME/.zsh/path.sh"; printf %s "$PATH"'
        [ "$status" -eq 0 ] || false
        [[ "$output" == "$SHIMS:$LOCAL:/usr/bin:/bin" ]] || { echo "$s: $output"; false; }
    done
}

@test "path.sh leaves no helper or variable behind" {
    for s in $(shells); do
        run env -i HOME="$H" PATH=/usr/bin:/bin "$s" -c \
            '. "$HOME/.zsh/path.sh"; command -v _dotf_path_front; printf "[%s]" "${_dotf_rest-unset}"'
        [[ "$output" == "[unset]" ]] || { echo "$s: $output"; false; }
    done
}

@test "path.sh follows MISE_DATA_DIR, as mise does" {
    run env -i HOME="$H" PATH=/usr/bin:/bin MISE_DATA_DIR="$BATS_TEST_TMPDIR/mise" sh -c \
        '. "$HOME/.zsh/path.sh"; printf %s "$PATH"'
    [ "$(head_of "$output")" = "$BATS_TEST_TMPDIR/mise/shims:$LOCAL" ]
}

@test "zsh -c reads .zshenv and starts with the shims first" {
    command -v zsh >/dev/null 2>&1 || skip "no zsh on this host"
    cp "$REPO/.zshenv" "$H/.zshenv"
    run env -i HOME="$H" PATH=/usr/bin:/bin zsh -c 'printf %s "$PATH"'
    [ "$status" -eq 0 ] || false
    [ "$(head_of "$output")" = "$SHIMS:$LOCAL" ]
}

@test "bash -lc reads .profile and starts with the shims first" {
    cp "$REPO/.profile" "$H/.profile"
    run env -i HOME="$H" PATH=/usr/bin:/bin bash -lc 'printf %s "$PATH"'
    [ "$status" -eq 0 ] || false
    # The last line: /etc/profile may print a notice first.
    [ "$(head_of "${lines[${#lines[@]}-1]}")" = "$SHIMS:$LOCAL" ]
}

@test "a non-interactive .bashrc sets the PATH head before it returns" {
    cp "$REPO/.bashrc" "$H/.bashrc"
    run env -i HOME="$H" PATH=/usr/bin:/bin bash -c '. "$HOME/.bashrc"; printf %s "$PATH"'
    [ "$status" -eq 0 ] || false
    [ "$(head_of "$output")" = "$SHIMS:$LOCAL" ]
}

@test "every file that sources path.sh is a deploy entry, and so is path.sh" {
    for src in .zshenv .profile .bashrc .zsh/path.sh; do
        run jq -e --arg s "$src" '.configs[] | select(.src == $s)' "$REPO/ai/deploy.json"
        [ "$status" -eq 0 ] || { echo "not deployed: $src"; false; }
    done
}
