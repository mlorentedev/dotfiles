#!/usr/bin/env bats
# HARNESS-153 (#1712): `compile-harness.sh --deploy` renders one worktree's
# records into targets every session on the machine shares, so the last deploy
# wins. A checkout whose base predates main's last harness/ change renders back
# what main retired. The deploy warns, naming the missing commit, and still
# deploys: the check is fail-open, so a checkout it cannot judge is not blocked.

setup() {
    SCRIPT="$BATS_TEST_DIRNAME/../scripts/compile-harness.sh"
    TMP="$(mktemp -d)"
    REPO="$TMP/repo"
    mkdir -p "$REPO/harness"
    cd "$REPO" || exit 1
    git init -q -b main
    git config user.email t@t; git config user.name t; git config commit.gpgsign false
    # No skills or agents block: --deploy reaches the check, then has nothing to render.
    printf '{ "version": 1, "targets": [] }\n' > harness/manifest.json
    git add -A; git commit -q -m seed
    BASE="$(git rev-parse HEAD)"
    export HARNESS_REPO_ROOT="$REPO"
}

teardown() { cd / || true; rm -rf "$TMP"; }

# main gains a harness/ change after BASE; origin/main points at it.
retire_on_main() {
    printf 'retired\n' > harness/retired.txt
    git add -A; git commit -q -m "retire a skill"
    RETIRED="$(git rev-parse --short HEAD)"
    git update-ref refs/remotes/origin/main HEAD
}

@test "a checkout whose base predates main's last harness/ change warns, naming it" {
    retire_on_main
    git checkout -q -b stale "$BASE"
    run "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [[ "$output" == *"WARN"* ]] || false
    [[ "$output" == *"$RETIRED"* ]] || false
    [[ "$output" == *"retire a skill"* ]] || false
}

@test "a deploy from main does not warn" {
    retire_on_main
    run "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [[ "$output" != *"WARN"* ]] || false
}

@test "a branch that contains main's last harness/ change does not warn" {
    retire_on_main
    git checkout -q -b fresh
    printf 'x\n' > unrelated.txt
    git add -A; git commit -q -m "unrelated work"
    run "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [[ "$output" != *"WARN"* ]] || false
}

@test "a branch behind main on files outside harness/ does not warn" {
    git update-ref refs/remotes/origin/main HEAD
    git checkout -q -b behind
    git checkout -q main
    printf 'x\n' > unrelated.txt
    git add -A; git commit -q -m "unrelated on main"
    git update-ref refs/remotes/origin/main HEAD
    git checkout -q behind
    run "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [[ "$output" != *"WARN"* ]] || false
}

@test "a checkout without origin/main skips the check and deploys" {
    run "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [[ "$output" != *"WARN"* ]] || false
    [[ "$output" == *"nothing to deploy"* ]] || false
}

@test "a deploy root that is not a git checkout skips the check without a git error" {
    rm -rf "$REPO/.git"
    run "$SCRIPT" --deploy
    [ "$status" -eq 0 ]
    [[ "$output" != *"WARN"* ]] || false
    [[ "$output" != *"fatal"* ]] || false
}
