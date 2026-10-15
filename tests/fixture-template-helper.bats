#!/usr/bin/env bats
# CI-012 (#2059) N3: tests/lib/fixture-template.bash builds a git fixture once per
# file and hands each test its own copy. Five files ran `git init` plus a commit
# in every setup(); the copy is what keeps that saving honest, so these cases pin
# what a test may rely on: the copy holds the template's history, the copy is
# its own to write, and the template itself cannot be written.

load 'lib/fixture-template'

_seed_repo() {
    git init -q -b main
    git config user.email t@t
    git config user.name t
    git config commit.gpgsign false
    echo seed > seed.txt
    git add seed.txt
    git commit -q -m seed
}

setup_file() {
    fixture_template_build repo _seed_repo
}

teardown_file() { fixture_template_release; }

@test "a copy holds the template's committed history and a clean tree" {
    fixture_template_copy repo "$BATS_TEST_TMPDIR/c"
    run git -C "$BATS_TEST_TMPDIR/c" log --format=%s
    [ "$status" -eq 0 ]
    [ "$output" = seed ]
    run git -C "$BATS_TEST_TMPDIR/c" status --porcelain
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "a copy is the test's own to write and commit to" {
    fixture_template_copy repo "$BATS_TEST_TMPDIR/c"
    cd "$BATS_TEST_TMPDIR/c"
    printf 'more\n' >> seed.txt
    git commit -q -am more
    run git log --format=%s
    [ "${lines[0]}" = more ]
}

@test "a write into the template fails" {
    [ "$(id -u)" -ne 0 ] || skip "root ignores the mode bits that keep the template read-only"
    run touch "$BATS_FILE_TMPDIR/template-repo/.written-by-a-test"
    [ "$status" -ne 0 ]
    # An append, not a touch: the owner may reset a timestamp without write access.
    run sh -c 'printf x >> "$1"' _ "$BATS_FILE_TMPDIR/template-repo/seed.txt"
    [ "$status" -ne 0 ]
}

@test "a copy of one test does not leak into the next" {
    fixture_template_copy repo "$BATS_TEST_TMPDIR/c"
    [ ! -e "$BATS_TEST_TMPDIR/c/more" ]
    run git -C "$BATS_TEST_TMPDIR/c" rev-list --count HEAD
    [ "$output" = 1 ]
}

@test "a builder that fails makes the build fail and names the template" {
    run bash -c '
        BATS_FILE_TMPDIR="$1"
        . "$2/lib/fixture-template.bash"
        broken() { git init -q; false; echo reached-after-the-failure; }
        # Under `||` on purpose: that context switches errexit off in every
        # subshell below it, and the builder must still stop at its failure.
        rc=0; fixture_template_build broken broken || rc=$?
        fixture_template_release
        exit "$rc"
    ' _ "$BATS_TEST_TMPDIR" "$BATS_TEST_DIRNAME"
    [ "$status" -ne 0 ]
    [[ "$output" == *"template broken"* ]] || false
    # The failure is mid-builder on purpose: only the last command's status
    # would reach a caller that ran the builder with errexit ignored.
    [[ "$output" != *reached-after-the-failure* ]] || false
}

@test "a builder that reads a variable the caller did not export fails" {
    run bash -c '
        BATS_FILE_TMPDIR="$1"
        . "$2/lib/fixture-template.bash"
        NOT_EXPORTED=value
        reads() { echo "$NOT_EXPORTED" > seen; }
        rc=0; fixture_template_build reads reads || rc=$?
        fixture_template_release
        exit "$rc"
    ' _ "$BATS_TEST_TMPDIR" "$BATS_TEST_DIRNAME"
    [ "$status" -ne 0 ]
    [[ "$output" == *NOT_EXPORTED* ]] || false
}

@test "copying a template that was never built fails instead of handing out an empty tree" {
    run fixture_template_copy never-built "$BATS_TEST_TMPDIR/c"
    [ "$status" -ne 0 ]
    [[ "$output" == *"template never-built"* ]] || false
    [ ! -e "$BATS_TEST_TMPDIR/c" ]
}

@test "a copy refuses a destination that already exists" {
    mkdir -p "$BATS_TEST_TMPDIR/c"
    run fixture_template_copy repo "$BATS_TEST_TMPDIR/c"
    [ "$status" -ne 0 ]
    # cp -a into an existing directory nests the template inside it, which every
    # path in the test would then miss.
    [ ! -e "$BATS_TEST_TMPDIR/c/template-repo" ]
}
