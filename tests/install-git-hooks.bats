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

# BUG-040 AC1: hooksPath pointing at a DIFFERENT tree that is nonetheless a GUARD
# dispatcher means the guard IS running (the normal state when the hooks are
# developed from a repo checkout). Reporting it INACTIVE was the false negative.
@test "an equivalent dispatcher at another path is reported active, not INACTIVE" {
    OTHER="$TMP/checkout/git-hooks"
    mk_src "$OTHER"
    git config --global core.hooksPath "$OTHER"

    run install_git_hooks "$SRC" "$DOTF"
    [ "$status" -eq 0 ]
    [[ "$output" != *"INACTIVE"* ]]
    [[ "$output" == *"$OTHER"* ]]

    # No-clobber still stands: the value is reported, never repointed.
    run git config --global --get core.hooksPath
    [ "$output" = "$OTHER" ]
}

# BUG-040 AC2: a pre-commit alone is NOT a GUARD dispatcher — pre-commit execs
# lib/memory-sink-guard.sh, so without it the guard genuinely cannot run. The
# effectiveness test must not be relaxed into "any hooks dir passes".
@test "a pre-commit without the memory-sink guard still warns" {
    OTHER="$TMP/other/git-hooks"
    mkdir -p "$OTHER"
    printf '#!/usr/bin/env bash\nexit 0\n' > "$OTHER/pre-commit"
    git config --global core.hooksPath "$OTHER"

    run install_git_hooks "$SRC" "$DOTF"
    [ "$status" -eq 0 ]
    [[ "$output" == *"preserving it"* ]]
    [[ "$output" == *"INACTIVE"* ]]
}

# BUG-040 AC3: a trailing separator names the same directory; it must resolve to
# the already-wired tier, not fall through to the foreign-path WARN.
@test "a trailing-slash variant of the target counts as already wired" {
    install_git_hooks "$SRC" "$DOTF"
    git config --global core.hooksPath "$DEST/"

    run install_git_hooks "$SRC" "$DOTF"
    [ "$status" -eq 0 ]
    [[ "$output" == *"already wired to the GUARD dispatcher"* ]]
    [[ "$output" != *"INACTIVE"* ]]
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

# #695: in the in-place layout src and dest resolve to the same dir. The
# clean-mirror (rm -rf dest THEN cp src/. dest) would EMPTY the dispatcher and
# still report success. deploy_git_hooks must detect the collision and no-op.
@test "deploy_git_hooks is a safe no-op when src and dest are the same dir (#695)" {
    run deploy_git_hooks "$SRC" "$SRC"
    [ "$status" -eq 0 ]
    [ -f "$SRC/pre-commit" ]                    # dispatcher NOT emptied
    [ -f "$SRC/lib/memory-sink-guard.sh" ]      # lib subtree intact
}

# BUG-068: cp is byte-verbatim, so a CRLF-tainted checkout would propagate a CRLF
# shebang into the deploy mirror and every hook would die "No such file or
# directory". deploy_git_hooks must normalize the deployed dispatchers to LF.
@test "deploy normalizes CRLF hook shebangs to LF (BUG-068)" {
    printf '#!/usr/bin/env bash\r\nexit 0\r\n' > "$SRC/pre-commit"
    run deploy_git_hooks "$SRC" "$DEST"
    [ "$status" -eq 0 ]
    run grep -c "$(printf '\r')" "$DEST/pre-commit"
    [ "$output" -eq 0 ]
    [ "$(head -1 "$DEST/pre-commit")" = "#!/usr/bin/env bash" ]
}
