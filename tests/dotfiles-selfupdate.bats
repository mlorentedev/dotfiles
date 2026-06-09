#!/usr/bin/env bats
# OPS-001: behavior tests for scripts/dotfiles-selfupdate.sh
#
# The self-deploy entrypoint: guard a dirty worktree, fetch, fast-forward ONLY,
# and re-run the (injected) setup only when HEAD actually moved. Anything that is
# not a clean fast-forward is logged and skipped (exit 0). A real setup failure
# is the ONLY non-zero exit.
#
# Fixture: a bare remote + a local clone + a stub setup command injected via
# DOTFILES_SELFUPDATE_SETUP_CMD that records each run to $RAN.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    SCRIPT="$DOTFILES_DIR/scripts/dotfiles-selfupdate.sh"

    TMP="$(mktemp -d)"
    REMOTE="$TMP/remote.git"
    REPO="$TMP/repo"
    RAN="$TMP/setup-ran"

    git init -q --bare "$REMOTE"
    git init -q "$REPO"
    git -C "$REPO" config user.email t@t.t
    git -C "$REPO" config user.name tester
    git -C "$REPO" config commit.gpgsign false
    printf 'v1\n' > "$REPO/file"
    git -C "$REPO" add file
    git -C "$REPO" commit -qm init
    git -C "$REPO" remote add origin "$REMOTE"
    git -C "$REPO" push -qu origin HEAD

    STUB="$TMP/stub-setup.sh"
    printf '#!/usr/bin/env bash\necho ran >> "%s"\n' "$RAN" > "$STUB"
    chmod +x "$STUB"

    export DOTFILES_REPO_DIR="$REPO"
    export DOTFILES_SELFUPDATE_SETUP_CMD="$STUB"
}

teardown() {
    [ -n "${TMP:-}" ] && rm -rf "$TMP"
}

# Advance the remote by one commit (a second clone commits + pushes).
advance_remote() {
    local work="$TMP/work_$RANDOM"
    git clone -q "$REMOTE" "$work"
    git -C "$work" config user.email t@t.t
    git -C "$work" config user.name tester
    git -C "$work" config commit.gpgsign false
    printf 'remote change\n' >> "$work/file"
    git -C "$work" commit -qam "remote advance"
    git -C "$work" push -q origin HEAD
    rm -rf "$work"
}

ran_count() {
    [ -f "$RAN" ] && wc -l < "$RAN" | tr -d ' ' || echo 0
}

# --- syntax (cross-shell, per project standard) ---

@test "dotfiles-selfupdate.sh has valid bash syntax" {
    bash -n "$SCRIPT"
}

@test "dotfiles-selfupdate.sh has valid zsh syntax" {
    zsh -n "$SCRIPT"
}

# --- AC1: dirty worktree -> skip, exit 0, no setup ---

@test "skips on a dirty worktree without running setup" {
    printf 'local edit\n' >> "$REPO/file"   # uncommitted change
    run "$SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" == *[Dd]irty* ]]
    [ "$(ran_count)" -eq 0 ]
}

# --- AC2: diverged / non-fast-forward -> skip, exit 0, no setup ---

@test "skips when local has diverged (non fast-forward) without running setup" {
    advance_remote                              # remote moves ahead
    printf 'local only\n' >> "$REPO/file"       # and a divergent local commit
    git -C "$REPO" commit -qam "local advance"
    run "$SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" == *diverg* ]] || [[ "$output" == *"fast-forward"* ]]
    [ "$(ran_count)" -eq 0 ]
}

# --- AC3: already current -> exit 0, no setup ---

@test "is a no-op when already current (no setup run)" {
    run "$SCRIPT"
    [ "$status" -eq 0 ]
    [ "$(ran_count)" -eq 0 ]
}

# --- AC4: clean fast-forward -> ff + run setup exactly once ---

@test "fast-forwards and runs setup exactly once when the remote is ahead" {
    advance_remote
    run "$SCRIPT"
    [ "$status" -eq 0 ]
    [ "$(ran_count)" -eq 1 ]
    # HEAD now matches the upstream (fast-forward happened)
    [ "$(git -C "$REPO" rev-parse HEAD)" = "$(git -C "$REPO" rev-parse '@{u}')" ]
}

# --- AC5: setup failure -> non-zero exit ---

@test "exits non-zero when the setup command fails" {
    advance_remote
    printf '#!/usr/bin/env bash\nexit 7\n' > "$STUB"   # failing setup
    chmod +x "$STUB"
    run "$SCRIPT"
    [ "$status" -ne 0 ]
}

# --- network fetch failure -> skip, exit 0 (transient) ---

@test "skips (exit 0) when the remote is unreachable" {
    git -C "$REPO" remote set-url origin "$TMP/does-not-exist.git"
    run "$SCRIPT"
    [ "$status" -eq 0 ]
    [ "$(ran_count)" -eq 0 ]
}

# --- missing repo dir -> skip, exit 0 (no spam) ---

@test "skips (exit 0) when the repo dir is not a git repo" {
    export DOTFILES_REPO_DIR="$TMP/nonrepo"
    mkdir -p "$DOTFILES_REPO_DIR"
    run "$SCRIPT"
    [ "$status" -eq 0 ]
    [ "$(ran_count)" -eq 0 ]
}
