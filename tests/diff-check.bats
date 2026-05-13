#!/usr/bin/env bats
# Tests for scripts/diff-check.sh — drift detection between repo and deploy-dir

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    REPO_FIXTURE="/tmp/bats_diff_repo_$$"
    DEPLOY_FIXTURE="/tmp/bats_diff_deploy_$$"
    mkdir -p "$REPO_FIXTURE/scripts" "$REPO_FIXTURE/.zsh" "$REPO_FIXTURE/tests"
    mkdir -p "$DEPLOY_FIXTURE/scripts" "$DEPLOY_FIXTURE/.zsh" "$DEPLOY_FIXTURE/tests"

    # Initialize a git repo so `git ls-files` works
    cd "$REPO_FIXTURE" || exit
    git init -q
    git config user.email test@test
    git config user.name test
}

teardown() {
    rm -rf "$REPO_FIXTURE" "$DEPLOY_FIXTURE"
}

# Stage and commit one file in the repo fixture
_track() {
    cd "$REPO_FIXTURE" || return 1
    git add "$1" 2>/dev/null
    git commit -q -m "add $1" 2>/dev/null
}

@test "diff-check.sh --help shows usage and exits 0" {
    run "$SCRIPTS_DIR/diff-check.sh" --help
    [[ $status -eq 0 ]]
    [[ "$output" == *"Usage"* ]]
    [[ "$output" == *"DOTFILES_REPO_DIR"* ]]
}

@test "diff-check.sh rejects unknown args" {
    run "$SCRIPTS_DIR/diff-check.sh" --bogus
    [[ $status -eq 2 ]]
    [[ "$output" == *"Unknown argument"* ]]
}

@test "diff-check.sh exits 2 when repo is not a git repo" {
    rm -rf "$REPO_FIXTURE/.git"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" DOTFILES_DIR="$DEPLOY_FIXTURE" \
        "$SCRIPTS_DIR/diff-check.sh"
    [[ $status -eq 2 ]]
    [[ "$output" == *"not a git repo"* ]]
}

@test "diff-check.sh exits 2 when deploy-dir is missing" {
    rm -rf "$DEPLOY_FIXTURE"
    echo "x" > "$REPO_FIXTURE/tmux.conf"
    _track "tmux.conf"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" DOTFILES_DIR="$DEPLOY_FIXTURE" \
        "$SCRIPTS_DIR/diff-check.sh"
    [[ $status -eq 2 ]]
    [[ "$output" == *"deploy-dir not found"* ]]
}

@test "diff-check.sh exits 0 when managed files match" {
    echo "same content" > "$REPO_FIXTURE/tmux.conf"
    echo "same content" > "$DEPLOY_FIXTURE/tmux.conf"
    _track "tmux.conf"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" DOTFILES_DIR="$DEPLOY_FIXTURE" \
        "$SCRIPTS_DIR/diff-check.sh"
    [[ $status -eq 0 ]]
    [[ "$output" == *"No drift"* ]]
}

@test "diff-check.sh exits 1 and reports drifted file" {
    echo "repo version" > "$REPO_FIXTURE/tmux.conf"
    echo "deployed version" > "$DEPLOY_FIXTURE/tmux.conf"
    _track "tmux.conf"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" DOTFILES_DIR="$DEPLOY_FIXTURE" \
        "$SCRIPTS_DIR/diff-check.sh"
    [[ $status -eq 1 ]]
    [[ "$output" == *"DRIFT"* ]]
    [[ "$output" == *"tmux.conf"* ]]
    [[ "$output" == *"setup-linux.sh"* ]]
}

@test "diff-check.sh ignores tracked files outside the managed allowlist" {
    # tests/ is tracked in the repo but NOT deployed by setup-linux.sh.
    # A divergent copy in the deploy-dir should NOT be flagged as drift.
    echo "repo test" > "$REPO_FIXTURE/tests/example.bats"
    echo "stale test" > "$DEPLOY_FIXTURE/tests/example.bats"
    _track "tests/example.bats"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" DOTFILES_DIR="$DEPLOY_FIXTURE" \
        "$SCRIPTS_DIR/diff-check.sh"
    [[ $status -eq 0 ]]
    [[ "$output" == *"No drift"* ]]
}

@test "diff-check.sh ignores managed files not yet deployed" {
    # tmux.conf in repo but not in deploy-dir is "not yet deployed", not drift.
    echo "new" > "$REPO_FIXTURE/tmux.conf"
    _track "tmux.conf"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" DOTFILES_DIR="$DEPLOY_FIXTURE" \
        "$SCRIPTS_DIR/diff-check.sh"
    [[ $status -eq 0 ]]
    [[ "$output" == *"No drift"* ]]
}

@test "diff-check.sh --verbose includes diff output for drifted files" {
    printf 'line1\nline2 repo\n' > "$REPO_FIXTURE/.zshrc"
    printf 'line1\nline2 deploy\n' > "$DEPLOY_FIXTURE/.zshrc"
    _track ".zshrc"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" DOTFILES_DIR="$DEPLOY_FIXTURE" \
        "$SCRIPTS_DIR/diff-check.sh" --verbose
    [[ $status -eq 1 ]]
    [[ "$output" == *"line2 repo"* ]]
    [[ "$output" == *"line2 deploy"* ]]
}

@test "diff-check.sh detects drift in nested managed dir (.zsh/)" {
    echo "alias x=1" > "$REPO_FIXTURE/.zsh/aliases.zsh"
    echo "alias x=2" > "$DEPLOY_FIXTURE/.zsh/aliases.zsh"
    _track ".zsh/aliases.zsh"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" DOTFILES_DIR="$DEPLOY_FIXTURE" \
        "$SCRIPTS_DIR/diff-check.sh"
    [[ $status -eq 1 ]]
    [[ "$output" == *".zsh/aliases.zsh"* ]]
}

@test "diff-check.sh runs under zsh" {
    echo "x" > "$REPO_FIXTURE/tmux.conf"
    echo "x" > "$DEPLOY_FIXTURE/tmux.conf"
    _track "tmux.conf"
    run zsh -c 'DOTFILES_REPO_DIR="$1" DOTFILES_DIR="$2" "$3/diff-check.sh"' \
        -- "$REPO_FIXTURE" "$DEPLOY_FIXTURE" "$SCRIPTS_DIR"
    [[ $status -eq 0 ]]
}
