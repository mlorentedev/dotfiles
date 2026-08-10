#!/usr/bin/env bats
# Real-dependency sibling of spec-gate-pr.bats (see tests/stub-real-pairing.bats).
#
# The stub suite proves what we ASKED `gh` to do. It cannot prove that `gh`
# accepts it -- and the likeliest way scripts/spec-gate-pr.sh breaks in
# production is a --json field name that is wrong or renamed upstream, which a
# stub answering any invocation will never surface. That is the BUG-055 shape
# exactly: 15 green cases over a branch the real tool rejected.
#
# `gh pr view --json <fields>` validates field names locally, BEFORE it resolves
# a repository, so the contract can be pinned with the real binary and no token,
# no network and no live PR.

# A real-dependency suite that quietly skips is a green proving nothing
# (BUG-055, #807). So a missing precondition is FATAL under CI -- where the
# workflow supplies the token -- and only relaxed on a dev machine.
_need() {
    [ -z "${CI:-}" ] && skip "$1"
    printf 'real-dependency precondition unmet in CI: %s\n' "$1" >&2
    return 1
}

setup() {
    command -v gh >/dev/null || _need "gh is not installed"

    # Scoped to THIS suite rather than exporting GH_TOKEN for the whole bats
    # run: every other suite keeps the auth state it has today, so none of them
    # can start making real API calls as a side effect of this file.
    [ -n "${DOTF_TEST_GH_TOKEN:-}" ] && export GH_TOKEN="$DOTF_TEST_GH_TOKEN"

    # An UNAUTHENTICATED gh exits at the auth check before it ever validates
    # --json field names. Without this precondition the field tests below assert
    # nothing while still reporting green -- which is precisely how they passed
    # locally (authenticated) and failed in CI (not) on their first run.
    gh auth status >/dev/null 2>&1 || _need "gh is not authenticated"

    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    OUTSIDE="/tmp/bats_specgatepr_real_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$OUTSIDE"
    # Nothing here may resolve to a repository: these tests must fail on repo
    # resolution, never on a real API call.
    unset GH_REPO GITHUB_REPOSITORY
}

teardown() {
    cd / || true
    rm -rf "$OUTSIDE"
}

@test "real gh accepts every --json field the adapter asks for" {
    cd "$OUTSIDE" || return 1
    run gh pr view --json labels,body,author
    # It must fail (there is no repo here) but NOT because of the field names.
    [ "$status" -ne 0 ]
    [[ "$output" != *"Unknown JSON field"* ]]
}

@test "the field check is a detector, not a tautology" {
    cd "$OUTSIDE" || return 1
    run gh pr view --json labels,body,author,notAFieldName
    [ "$status" -ne 0 ]
    [[ "$output" == *"Unknown JSON field"* ]]
    [[ "$output" == *"notAFieldName"* ]]
}

@test "a real gh failure makes the adapter fail closed" {
    cd "$OUTSIDE" || return 1
    # Real gh, real failure (no repo to resolve): the adapter must refuse rather
    # than fall back to anything. check-spec-gate.sh sits beside the real script,
    # so reaching it at all would be visible as a different error.
    run "$SCRIPTS_DIR/spec-gate-pr.sh" --pr 1 --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 2 ]
    [[ "$output" == *"could not read live metadata"* ]]
    [[ "$output" == *"Refusing to fall back to the event payload"* ]]
}
