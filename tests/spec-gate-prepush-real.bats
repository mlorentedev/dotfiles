#!/usr/bin/env bats
# Real-dependency sibling of spec-gate-prepush.bats (see tests/stub-real-pairing.bats).
#
# The stub suite proves what we ASKED `gh` to do. It cannot prove that `gh`
# accepts it -- and the likeliest way scripts/spec-gate-prepush.sh breaks in
# production is a flag that is wrong or renamed upstream, which a stub
# answering any invocation will never surface. That is the BUG-055 shape
# exactly: 15 green cases over a branch the real tool rejected.
#
# `gh api` rejects an unknown flag locally, BEFORE it resolves a repository, so
# the contract can be pinned with the real binary and no live PR. (The lookup
# moved from `gh pr list --json` to REST in #2710; there are no field names left
# to validate offline.)

# A real-dependency suite that quietly skips is a green proving nothing
# (BUG-055, #807). So a missing precondition is FATAL under CI -- where the
# workflow supplies the token -- and only relaxed on a dev machine.
_need() {
    [ -z "${CI:-}" ] && skip "$1"
    printf 'real-dependency precondition unmet in CI: %s\n' "$1" >&2
    return 1
}

load 'lib/gh-auth'

setup() {
    command -v gh >/dev/null || _need "gh is not installed"

    # Scoped to THIS suite rather than exporting GH_TOKEN for the whole bats
    # run: every other suite keeps the auth state it has today, so none of them
    # can start making real API calls as a side effect of this file.
    [ -n "${DOTF_TEST_GH_TOKEN:-}" ] && export GH_TOKEN="$DOTF_TEST_GH_TOKEN"

    # An UNAUTHENTICATED gh exits at the auth check before it ever validates
    # --json field names. Without this precondition the field tests below assert
    # nothing while still reporting green -- which is precisely how spec-gate-
    # pr-real.bats's equivalent test passed locally (authenticated) and failed
    # in CI (not) on its first run.
    # Offline on purpose: `gh auth status` probes the API, and a blip there
    # read as a missing token and turned main red (#2442).
    gh_has_token || _need "gh has no token (GH_TOKEN, DOTF_TEST_GH_TOKEN or a stored login)"

    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    OUTSIDE="/tmp/bats_specgateprepush_real_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$OUTSIDE"
    # Nothing here may resolve to a repository: these tests must fail on repo
    # resolution, never on a real API call.
    unset GH_REPO GITHUB_REPOSITORY
}

teardown() {
    cd / || true
    rm -rf "$OUTSIDE"
}

@test "real gh accepts every flag the adapter's REST lookup uses" {
    cd "$OUTSIDE" || return 1
    run gh api 'repos/{owner}/{repo}/pulls?head={owner}:some-branch&state=open&per_page=1' \
        --jq '.[0] // empty | {labels: [.labels[] | {name}], body, author: {login: .user.login}}'
    # It must fail (there is no repo here) but NOT because gh rejected a flag.
    [ "$status" -ne 0 ]
    [[ "$output" != *"unknown flag"* ]] || false
    [[ "$output" != *"unknown command"* ]] || false
}

@test "the flag check is a detector, not a tautology" {
    cd "$OUTSIDE" || return 1
    run gh api 'repos/{owner}/{repo}/pulls' --notaflag
    [ "$status" -ne 0 ]
    [[ "$output" == *"unknown flag"* ]] || false
}

@test "a real gh failure falls through to the gate instead of failing closed" {
    cd "$OUTSIDE" || return 1
    # Real gh, real failure (no repo to resolve, so no PR to find). Unlike
    # spec-gate-pr.sh (which must fail closed in CI to avoid a stale event-
    # payload replay, BUG-066), this adapter has no such fallback to guard
    # against: it runs the gate with no PR context, same as it would with no
    # `gh` installed at all.
    run "$SCRIPTS_DIR/spec-gate-prepush.sh" --base-ref origin/main --head-ref HEAD
    # No git repo here either, so check-spec-gate.sh itself now fails (exit 2,
    # "Not in a git repo") -- proof the adapter reached it and ran the gate
    # rather than swallowing the gh failure into some exit code of its own.
    [ "$status" -eq 2 ]
    [[ "$output" == *"Not in a git repo"* ]] || false
}
