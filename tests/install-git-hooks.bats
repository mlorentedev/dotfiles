#!/usr/bin/env bats
# Tests for scripts/install-git-hooks.sh — the GUARD-001 dispatcher deploy + the
# machine-wide core.hooksPath wiring (#418).
#
# Hermetic: GIT_CONFIG_GLOBAL is redirected at a temp file, so every
# `git config --global` reads/writes the fixture, never the real ~/.gitconfig.
# A fake git-hooks/ source stands in for the tracked dispatcher tree.

mk_src() {
    local s="$1"
    mkdir -p "$s/lib"
    printf '#!/usr/bin/env bash\nexit 0\n' > "$s/pre-commit"
    : > "$s/commit-msg"
    : > "$s/prepare-commit-msg"
    : > "$s/pre-push"
    : > "$s/post-checkout"
    : > "$s/lib/memory-sink-guard.sh"
    : > "$s/lib/chain-local-hook.sh"
    : > "$s/lib/board-pickup.sh"
}

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    TMP="$(mktemp -d "/tmp/bats_ghooks_XXXXXX")"

    # Isolate git's global config at a fixture — no real ~/.gitconfig mutation.
    export GIT_CONFIG_GLOBAL="$TMP/gitconfig"
    : > "$GIT_CONFIG_GLOBAL"
    # Also redirect the DOTFILES_DIR default to a temp path so a fallback can
    # never touch the real ~/.dotfiles deploy mirror.
    export DOTFILES_DIR="$TMP/fallback"

    SRC="$TMP/src/git-hooks"
    DOTF="$TMP/dotfiles"
    DEST="$DOTF/git-hooks"
    mk_src "$SRC"

    # shellcheck source=/dev/null
    . "$SCRIPTS_DIR/install-git-hooks.sh"
}

teardown() {
    [ -z "${TMP:-}" ] || rm -rf "$TMP"
}

@test "install deploys the dispatcher and makes the entrypoint executable" {
    run install_git_hooks "$SRC" "$DOTF"
    [ "$status" -eq 0 ]
    [ -f "$DEST/pre-commit" ]
    [ -x "$DEST/pre-commit" ]
    [ -x "$DEST/post-checkout" ]
    [ -f "$DEST/lib/memory-sink-guard.sh" ]
    [ -f "$DEST/lib/board-pickup.sh" ]
}

@test "install wires core.hooksPath at the deployed dispatcher" {
    install_git_hooks "$SRC" "$DOTF"
    run git config --global --get core.hooksPath
    [ "$status" -eq 0 ]
    [ "$output" = "$DEST" ]
}

@test "re-run is idempotent (hooksPath unchanged, exit 0)" {
    install_git_hooks "$SRC" "$DOTF"
    run install_git_hooks "$SRC" "$DOTF"
    [ "$status" -eq 0 ]
    run git config --global --get core.hooksPath
    [ "$output" = "$DEST" ]
}

@test "an unrelated pre-existing core.hooksPath is preserved, not clobbered" {
    git config --global core.hooksPath "/home/u/.my-hooks"
    run install_git_hooks "$SRC" "$DOTF"
    [ "$status" -eq 0 ]
    [[ "$output" == *"preserving it"* ]]
    run git config --global --get core.hooksPath
    [ "$output" = "/home/u/.my-hooks" ]
}

@test "clean-mirror removes a stale hook on re-deploy" {
    install_git_hooks "$SRC" "$DOTF"
    touch "$DEST/stale-hook"
    install_git_hooks "$SRC" "$DOTF"
    [ ! -f "$DEST/stale-hook" ]
}

@test "refuses an unsafe root dotfiles dir (no destructive rm)" {
    run install_git_hooks "$SRC" "/"
    [ "$status" -ne 0 ]
    [ ! -d "/git-hooks" ]
}

@test "fails when the source dispatcher dir is missing" {
    run install_git_hooks "$TMP/nope" "$DOTF"
    [ "$status" -ne 0 ]
}

@test "fails when the source has no pre-commit dispatcher" {
    mkdir -p "$TMP/empty/git-hooks"
    run install_git_hooks "$TMP/empty/git-hooks" "$DOTF"
    [ "$status" -ne 0 ]
}
