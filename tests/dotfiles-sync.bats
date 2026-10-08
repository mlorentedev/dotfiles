#!/usr/bin/env bats
# Tests for scripts/dotfiles-sync.sh. Every run gets a temporary HOME and
# temporary directories: the script's defaults point at the real ~/.dotfiles
# and ~/Projects/dotfiles, and a test that reached them synced real files (#1795).

setup() {
    export SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    export HOME="$BATS_TEST_TMPDIR/home"
    mkdir -p "$HOME"
}

@test "dotfiles-sync.sh has valid bash syntax" {
    bash -n "$SCRIPTS_DIR/dotfiles-sync.sh"
}

@test "dotfiles-sync.sh has valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/dotfiles-sync.sh"
}

@test "dotfiles-sync.sh fails on a missing installation directory" {
    run bash -c 'DOTFILES_DIR="$HOME/absent" DOTFILES_REPO_DIR="$HOME" "$1/dotfiles-sync.sh"' -- "$SCRIPTS_DIR"
    [ "$status" -ne 0 ]
    [[ "$output" == *"not found"* ]] || false
}

@test "dotfiles-sync.sh refuses the same directory for both sides" {
    mkdir -p "$HOME/same"
    run bash -c 'DOTFILES_DIR="$HOME/same" DOTFILES_REPO_DIR="$HOME/same" "$1/dotfiles-sync.sh" 2>&1' -- "$SCRIPTS_DIR"
    [ "$status" -ne 0 ]
    [[ "$output" == *"same directory"* ]] || false
}

@test "dotfiles-sync.sh rejects the retired --secrets-only flag under bash and zsh" {
    for sh in bash zsh; do
        run "$sh" -c 'DOTFILES_DIR="$HOME/l" DOTFILES_REPO_DIR="$HOME/r" "$1/dotfiles-sync.sh" --secrets-only 2>&1' -- "$SCRIPTS_DIR"
        [ "$status" -eq 2 ]
        [[ "$output" == *"#1795"* ]] || false
    done
}

# #1795: a blob left only in the installation must never reach the repo, and the
# installation's sensitive/ must never be rewritten from the repo.
@test "dotfiles-sync.sh never copies sensitive/ in either direction" {
    command -v rsync >/dev/null || { echo "rsync is required by dotfiles-sync.sh itself" >&2; return 1; }
    local repo="$HOME/repo" inst="$HOME/inst"
    mkdir -p "$repo/sensitive" "$inst/sensitive"
    git -C "$repo" init -q
    printf 'x\n' > "$repo/file"
    git -C "$repo" add file
    git -C "$repo" -c user.name=t -c user.email=t@t commit -qm init
    printf 'retired\n' > "$inst/sensitive/retired.secret.age"
    printf 'repo-side\n' > "$repo/sensitive/tracked.secret.age"

    run bash -c 'DOTFILES_DIR="$1" DOTFILES_REPO_DIR="$2" "$3/dotfiles-sync.sh" 2>&1' -- "$inst" "$repo" "$SCRIPTS_DIR"
    [ "$status" -eq 0 ]
    [ ! -e "$repo/sensitive/retired.secret.age" ]
    [ ! -e "$inst/sensitive/tracked.secret.age" ]
    [ -f "$inst/file" ]
}

# #1795: scripts/test.sh runs as a pre-commit hook against the real $HOME, so it
# may check this script's syntax (`bash -n`) but must never execute it: neither
# as a command, env assignments allowed before it, nor through an interpreter.
@test "scripts/test.sh never executes dotfiles-sync.sh" {
    run grep -nE '^[[:space:]]*([A-Za-z_]+=[^[:space:]]*[[:space:]]+)*"\$SCRIPTS_DIR/dotfiles-sync\.sh"|(bash|zsh|sh)[[:space:]]+"\$SCRIPTS_DIR/dotfiles-sync\.sh"' "$SCRIPTS_DIR/test.sh"
    [ "$status" -eq 1 ]
}
