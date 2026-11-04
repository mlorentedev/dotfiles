#!/usr/bin/env bats
# Guard for #1855: check-actions-pinned.sh must judge a repository with a
# single workflow file the same as one with several.
#
# grep prints the "file:" prefix only when it is given more than one file. The
# sed that extracts the reference expected "file:line:", so with one workflow it
# left the raw line, which never ends in a SHA, and every pinned action was
# reported. ADR-042 makes one short caller workflow the normal shape for a
# consumer repository, so the false positive landed where the fleet is heading.

setup() {
    SCRATCH="$BATS_TEST_TMPDIR/check-actions-pinned"
    mkdir -p "$SCRATCH/scripts" "$SCRATCH/.github/workflows"
    cp "$BATS_TEST_DIRNAME/../scripts/check-actions-pinned.sh" "$SCRATCH/scripts/"
    SHA="0123456789abcdef0123456789abcdef01234567"
}

teardown() {
    rm -rf "$SCRATCH"
}

workflow() {
    printf 'on: pull_request\njobs:\n  review:\n    uses: %s\n' "$2" > "$SCRATCH/.github/workflows/$1"
}

@test "check-actions-pinned: one workflow file with a SHA-pinned action passes" {
    workflow caller.yml "mlorentedev/ci-workflows/.github/workflows/pr-agent.yml@$SHA # v1"
    run bash "$SCRATCH/scripts/check-actions-pinned.sh"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    [[ "$output" == *"OK (1 workflow files)"* ]] || { echo "$output"; false; }
}

@test "check-actions-pinned: one workflow file with a tag-pinned action fails and names the file" {
    workflow caller.yml "mlorentedev/ci-workflows/.github/workflows/pr-agent.yml@v1"
    run bash "$SCRATCH/scripts/check-actions-pinned.sh"
    [ "$status" -eq 1 ] || { echo "$output"; false; }
    [[ "$output" == *"caller.yml:4:mlorentedev/ci-workflows/.github/workflows/pr-agent.yml@v1"* ]] || { echo "$output"; false; }
}

@test "check-actions-pinned: two workflow files report only the unpinned one" {
    workflow pinned.yml "actions/checkout@$SHA"
    workflow tagged.yml "actions/checkout@v4"
    run bash "$SCRATCH/scripts/check-actions-pinned.sh"
    [ "$status" -eq 1 ] || { echo "$output"; false; }
    [[ "$output" == *"tagged.yml:4:actions/checkout@v4"* ]] || { echo "$output"; false; }
    [[ "$output" != *"pinned.yml"* ]] || { echo "$output"; false; }
}

# Guard for #3103: the script took its root from its own location and ignored
# any argument, so `check-actions-pinned.sh <other-repo>` printed OK about THIS
# repository. These run the script from the real tree against a temp root, so a
# regression that ignores the argument answers about the (pinned) real tree and
# fails the unpinned case.

@test "check-actions-pinned: a root argument is judged, not the script's own repository (unpinned fails)" {
    workflow caller.yml "actions/checkout@v4"
    run bash "$BATS_TEST_DIRNAME/../scripts/check-actions-pinned.sh" "$SCRATCH"
    [ "$status" -eq 1 ] || { echo "$output"; false; }
    [[ "$output" == *"caller.yml:4:actions/checkout@v4"* ]] || { echo "$output"; false; }
}

@test "check-actions-pinned: a root argument holding only SHA-pinned actions passes" {
    workflow caller.yml "actions/checkout@$SHA"
    run bash "$BATS_TEST_DIRNAME/../scripts/check-actions-pinned.sh" "$SCRATCH"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    [[ "$output" == *"OK (1 workflow files)"* ]] || { echo "$output"; false; }
}

@test "check-actions-pinned: a missing root directory is a usage error (exit 2)" {
    run bash "$BATS_TEST_DIRNAME/../scripts/check-actions-pinned.sh" "$SCRATCH/does-not-exist"
    [ "$status" -eq 2 ] || { echo "$output"; false; }
    [[ "$output" == *"usage:"* ]] || { echo "$output"; false; }
}

@test "check-actions-pinned: a file instead of a directory is a usage error (exit 2)" {
    run bash "$BATS_TEST_DIRNAME/../scripts/check-actions-pinned.sh" "$SCRATCH/scripts/check-actions-pinned.sh"
    [ "$status" -eq 2 ] || { echo "$output"; false; }
    [[ "$output" == *"usage:"* ]] || { echo "$output"; false; }
}

@test "check-actions-pinned: more than one argument is a usage error (exit 2)" {
    run bash "$BATS_TEST_DIRNAME/../scripts/check-actions-pinned.sh" "$SCRATCH" "$SCRATCH"
    [ "$status" -eq 2 ] || { echo "$output"; false; }
    [[ "$output" == *"usage:"* ]] || { echo "$output"; false; }
}

@test "check-actions-pinned: with no argument it still checks its own repository" {
    run bash "$BATS_TEST_DIRNAME/../scripts/check-actions-pinned.sh"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    [[ "$output" == *"check-actions-pinned: OK"* ]] || { echo "$output"; false; }
}
