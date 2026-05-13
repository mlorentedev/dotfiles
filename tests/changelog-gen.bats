#!/usr/bin/env bats
# Tests for scripts/changelog-gen.sh

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    REPO_FIXTURE="/tmp/bats_changelog_$$"
    mkdir -p "$REPO_FIXTURE"
    cd "$REPO_FIXTURE" || exit
    git init -q
    git config user.email test@test
    git config user.name test
    git commit -q --allow-empty -m "feat(core): initial commit"
    git commit -q --allow-empty -m "fix(api): correct null check"
    git commit -q --allow-empty -m "docs(readme): clarify install"
    git commit -q --allow-empty -m "random subject without type"
}

teardown() {
    rm -rf "$REPO_FIXTURE"
}

@test "changelog-gen.sh --help prints usage" {
    run "$SCRIPTS_DIR/changelog-gen.sh" --help
    [[ $status -eq 0 ]]
    [[ "$output" == *"Usage"* ]]
    [[ "$output" == *"Conventional"* ]]
}

@test "changelog-gen.sh rejects unknown args" {
    run "$SCRIPTS_DIR/changelog-gen.sh" --bogus
    [[ $status -eq 2 ]]
    [[ "$output" == *"Unknown"* ]]
}

@test "changelog-gen.sh exits 2 outside a git repo" {
    rm -rf "$REPO_FIXTURE/.git"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md"
    [[ $status -eq 2 ]]
    [[ "$output" == *"not a git repo"* ]]
}

@test "changelog-gen.sh writes a CHANGELOG with type sections" {
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md"
    [[ $status -eq 0 ]]
    [[ -f "$REPO_FIXTURE/CHANGELOG.md" ]]
    grep -q "^## Features" "$REPO_FIXTURE/CHANGELOG.md"
    grep -q "^## Bug Fixes" "$REPO_FIXTURE/CHANGELOG.md"
    grep -q "^## Documentation" "$REPO_FIXTURE/CHANGELOG.md"
}

@test "changelog-gen.sh buckets non-conventional commits under Other" {
    env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md"
    grep -q "^## Other" "$REPO_FIXTURE/CHANGELOG.md"
    grep -q "random subject without type" "$REPO_FIXTURE/CHANGELOG.md"
}

@test "changelog-gen.sh entry format includes date and short hash" {
    env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md"
    # Format: "- YYYY-MM-DD: subject (hash)"
    grep -qE '^- [0-9]{4}-[0-9]{2}-[0-9]{2}: .+ \([0-9a-f]+\)$' "$REPO_FIXTURE/CHANGELOG.md"
}

@test "changelog-gen.sh is idempotent (running twice produces same file)" {
    env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md"
    cp "$REPO_FIXTURE/CHANGELOG.md" "$REPO_FIXTURE/first.md"
    env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md"
    cmp -s "$REPO_FIXTURE/first.md" "$REPO_FIXTURE/CHANGELOG.md"
}

@test "changelog-gen.sh --check exits 0 when CHANGELOG is current" {
    env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md" --check
    [[ $status -eq 0 ]]
    [[ "$output" == *"up to date"* ]]
}

@test "changelog-gen.sh --check exits 1 when CHANGELOG is stale" {
    env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md"
    git commit --allow-empty -q -m "feat(new): added since"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md" --check
    [[ $status -eq 1 ]]
    [[ "$output" == *"out of date"* ]]
}

@test "changelog-gen.sh --check exits 1 when CHANGELOG is missing" {
    rm -f "$REPO_FIXTURE/CHANGELOG.md"
    run env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md" --check
    [[ $status -eq 1 ]]
}

@test "changelog-gen.sh --since limits to commits after revision" {
    local baseline
    baseline="$(git rev-parse HEAD)"
    git commit --allow-empty -q -m "feat(post): after baseline"
    env DOTFILES_REPO_DIR="$REPO_FIXTURE" "$SCRIPTS_DIR/changelog-gen.sh" --output "$REPO_FIXTURE/CHANGELOG.md" --since "$baseline"
    grep -q "after baseline" "$REPO_FIXTURE/CHANGELOG.md"
    ! grep -q "initial commit" "$REPO_FIXTURE/CHANGELOG.md"
}
