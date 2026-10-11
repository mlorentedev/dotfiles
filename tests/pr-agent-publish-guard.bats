#!/usr/bin/env bats
# Behaviour of the pr-agent publication guard (scripts/pr-agent-publish-guard.sh,
# run by "Fail if no review was published" in .github/workflows/pr-agent.yml)
# when the GitHub API fails (#2069), and of its --probe mode, which gates the
# second attempt of the review pool (AI-045 AC9, #1923).
#
# A read timeout on the registry used to become an empty marker list, and the
# guard then failed a reviewed PR with "no review marker declared": the wrong
# diagnosis, measured downstream on leaving-denver run 37155990064. These run the
# REAL script against a stub gh whose every answer is set per test.
# pr-agent-config.bats pins how the workflow calls it; this file pins what it does.
#
# Each test sets its stub's answers with `export`, which bats scopes to that
# test's own subshell by design.
# shellcheck disable=SC2030,SC2031

setup() {
    GUARD="$BATS_TEST_DIRNAME/../scripts/pr-agent-publish-guard.sh"
    [ -x "$GUARD" ] || { echo "the guard script is gone or not executable: $GUARD" >&2; return 1; }

    STUB="$BATS_TEST_TMPDIR/bin"
    CALLS="$BATS_TEST_TMPDIR/calls"
    mkdir -p "$STUB"
    : > "$CALLS"
    cat > "$STUB/gh" <<'SH'
#!/usr/bin/env bash
# Answers `gh api PATH ...` from STUB_<KEY>, where KEY is the call's subject:
# BASE / HEAD (the registry at each ref), PULL, COMMENTS. A value is either
# "fail:<code>" (always fails), "flaky:<code>" (fails on the first call only)
# or the response itself. Every call is logged to $CALLS.
api_path="$2"
case "$api_path" in
    *contents/harness/review-attestation.json?ref=main) key=BASE ;;
    *contents/harness/review-attestation.json?ref=*) key=HEAD ;;
    */pulls/*) key=PULL ;;
    */comments) key=COMMENTS ;;
    *) echo "stub gh: unexpected call: $*" >&2; exit 99 ;;
esac
echo "$key" >> "$CALLS"
eval "answer=\${STUB_$key:-}"
fail() {
    echo '{"message":"stub failure body"}'
    case "$1" in
        404) echo "gh: Not Found (HTTP 404)" >&2 ;;
        *) echo "gh: Gateway Timeout (HTTP $1)" >&2 ;;
    esac
    exit 1
}
case "$answer" in
    fail:*) fail "${answer#fail:}" ;;
    flaky:*)
        if [ "$(grep -c "^$key\$" "$CALLS")" -eq 1 ]; then fail "${answer#flaky:}"; fi
        answer=$(eval "printf '%s' \"\${STUB_${key}_OK}\"")
        ;;
esac
printf '%s\n' "$answer"
SH
    chmod +x "$STUB/gh"

    REGISTRY=$(printf '%s' '{"reviewers":[{"login":"github-actions","review_markers":["PR Reviewer Guide"]}]}' | base64 | tr -d "\n")
    NO_ENTRY=$(printf '%s' '{"reviewers":[{"login":"coderabbitai[bot]","review_markers":["Walkthrough"]}]}' | base64 | tr -d "\n")
    # shellcheck disable=SC2089  # a JSON literal, never word-split
    REVIEW='[{"user":{"login":"github-actions[bot]"},"updated_at":"2026-10-07T10:00:00Z","body":"## PR Reviewer Guide"}]'
    # shellcheck disable=SC2090  # the same literal, exported as data
    export CALLS REGISTRY NO_ENTRY REVIEW
}

# Runs the guard as the workflow does, with the final guard's env. Arguments
# reach the script, so `_guard --probe --output F` runs the probe.
_guard() {
    run env PATH="$STUB:$PATH" CALLS="$CALLS" GUARD_RETRY_SECONDS=0 \
        GITHUB_REPOSITORY=o/r PR_NUMBER=7 BASE_REF=main STARTED=2026-10-07T09:00:00Z \
        HEAD_SHA="${HEAD_SHA-abc123}" ATTEMPTS="${ATTEMPTS-$(attempts skipped success skipped)}" \
        ROUTE_NOTE="${ROUTE_NOTE-}" ROUTE_OUTCOME="${ROUTE_OUTCOME-success}" \
        "$GUARD" "$@"
}

# The workflow's ATTEMPTS, in execution order: Anthropic first (model `hk`), NaN
# (model `m`), Anthropic second. Each argument is that attempt's outcome.
attempts() { printf '%s hk\n%s m\n%s hk\n' "$1" "$2" "$3"; }

# The probe's answer, from the file it appends to.
_probe() {
    PROBE_OUT="$BATS_TEST_TMPDIR/probe.out"
    : > "$PROBE_OUT"
    _guard --probe --output "$PROBE_OUT"
}

calls_to() { grep -c "^$1\$" "$CALLS" || true; }

@test "a registry read that keeps failing is reported as an API failure, not a missing marker" {
    export STUB_BASE=fail:504 STUB_HEAD=fail:504
    _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"the GitHub API failed while determining whether a review marker is declared"* ]] || false
    [[ "$output" == *"review-attestation.json?ref=main"* ]] || false
    [[ "$output" == *"HTTP 504"* ]] || false
    [[ "$output" != *"no review marker declared"* ]] || false
    [ "$(calls_to BASE)" -eq 3 ]
}

@test "a registry read that fails once is retried, and the answer is not prefixed by the failed body" {
    export STUB_BASE=flaky:502 STUB_BASE_OK="$REGISTRY" STUB_COMMENTS="$REVIEW"
    _guard
    [ "$status" -eq 0 ]
    [[ "$output" == *'review published (markers: ["PR Reviewer Guide"])'* ]] || false
    [ "$(calls_to BASE)" -eq 2 ]
}

@test "a registry with no github-actions entry still fails with the existing message" {
    export STUB_BASE="$NO_ENTRY" STUB_HEAD="$NO_ENTRY"
    _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"no review marker declared for github-actions"* ]] || false
    [[ "$output" != *"GitHub API failed"* ]] || false
}

@test "a 404 is an answer, not an outage: no retry, and the head-ref fallback still applies" {
    export STUB_BASE=fail:404 STUB_HEAD="$REGISTRY" STUB_COMMENTS="$REVIEW"
    _guard
    [ "$status" -eq 0 ]
    [[ "$output" == *"reviewer registry entry found only on the PR head"* ]] || false
    [ "$(calls_to BASE)" -eq 1 ]
}

@test "a failed head-SHA lookup on an issue_comment run is reported as an API failure" {
    export STUB_BASE="$NO_ENTRY" STUB_PULL=fail:503
    HEAD_SHA="" _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"the GitHub API failed while determining whether a review marker is declared"* ]] || false
    [[ "$output" == *"pulls/7"* ]] || false
    [[ "$output" != *"no review marker declared"* ]] || false
    [ "$(calls_to PULL)" -eq 3 ]
}

@test "a comment listing that keeps failing is reported as an API failure, not a missing review" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS=fail:500
    _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"the GitHub API failed while listing the PR's comments"* ]] || false
    [[ "$output" != *"published no review"* ]] || false
}

@test "a comment listing that answers 404 is reported as such: no retry, no empty outage report" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS=fail:404
    _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"listing the PR's comments answered 404"* ]] || false
    [ "$(calls_to COMMENTS)" -eq 1 ]
    [[ "$output" != *"GitHub API failed"* ]] || false
}

@test "with every call answering and no review comment, the published-no-review diagnosis stands" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS='[]'
    _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"PR-Agent reported success but published no review"* ]] || false
    [[ "$output" == *"NaN's per-model concurrency limit"* ]] || false
}

@test "an Anthropic attempt that published nothing is not blamed on NaN" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS='[]'
    ATTEMPTS=$(printf 'success anthropic/claude-haiku-5-5\nskipped m\n') _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"PR-Agent reported success but published no review"* ]] || false
    [[ "$output" == *"No cause is measured yet for the Anthropic attempt"* ]] || false
    [[ "$output" != *"NaN"* ]] || false
}

@test "the published review names the attempt that produced it" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS="$REVIEW"
    ATTEMPTS=$(attempts skipped failure success) _guard
    [ "$status" -eq 0 ]
    [[ "$output" == *"on hk"* ]] || false
}

# --- probe: the gate in front of the second attempt (AI-045 AC9) -------------

@test "probe: a review published by this run answers true" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS="$REVIEW"
    _probe
    [ "$status" -eq 0 ]
    [ "$(cat "$PROBE_OUT")" = "published=true" ]
}

@test "probe: no review answers false, the only answer that runs a second attempt" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS='[]'
    _probe
    [ "$status" -eq 0 ]
    [ "$(cat "$PROBE_OUT")" = "published=false" ]
}

@test "probe: a review older than this run's start is not this run's review" {
    local old='[{"user":{"login":"github-actions[bot]"},"updated_at":"2026-10-07T08:00:00Z","body":"## PR Reviewer Guide"}]'
    export STUB_BASE="$REGISTRY" STUB_COMMENTS="$old"
    _probe
    [ "$(cat "$PROBE_OUT")" = "published=false" ]
}

@test "probe: an API that keeps failing answers unknown, never false" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS=fail:502
    _probe
    [ "$status" -eq 0 ]
    [ "$(cat "$PROBE_OUT")" = "published=unknown" ]
    [[ "$output" == *"HTTP 502"* ]] || false
}

@test "probe: no declared marker answers unknown, never false" {
    export STUB_BASE="$NO_ENTRY" STUB_HEAD="$NO_ENTRY"
    _probe
    [ "$(cat "$PROBE_OUT")" = "published=unknown" ]
}

@test "probe: a comment listing that is not JSON answers unknown, never false" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS='<html>bad gateway</html>'
    _probe
    [ "$(cat "$PROBE_OUT")" = "published=unknown" ]
}

# --- final guard: the last attempt that ran is the one judged -----------------

@test "a failed second attempt is reported with its own model, not the first one's" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS='[]'
    ATTEMPTS=$(attempts skipped failure failure) _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"PR-Agent ended failure on hk"* ]] || false
}

@test "Anthropic drawn first, then NaN: the NaN attempt is the one judged" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS='[]'
    ATTEMPTS=$(attempts failure cancelled skipped) _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"PR-Agent ended cancelled on m"* ]] || false
}

@test "Anthropic drawn first and alone: its outcome is the one judged" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS="$REVIEW"
    ATTEMPTS=$(attempts success skipped skipped) _guard
    [ "$status" -eq 0 ]
    [[ "$output" == *"on hk"* ]] || false
}

@test "a failure with no second attempt repeats why a member was out of the draw" {
    ATTEMPTS=$(attempts skipped failure skipped) ROUTE_NOTE="PR_AGENT_ANTHROPIC_API_KEY is not set" _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"PR-Agent ended failure on m"* ]] || false
    [[ "$output" == *"PR_AGENT_ANTHROPIC_API_KEY is not set"* ]] || false
}

@test "no attempt at all (no pool member answered) fails and says so" {
    ATTEMPTS=$(attempts skipped skipped skipped) _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"no review attempt ran"* ]] || false
    [[ "$output" == *"no member of the review pool answered"* ]] || false
}

# A draw that exits non-zero (an unknown PR_AGENT_PROVIDER) writes no outputs,
# so every attempt reads skipped too; blaming the pool would send the reader to
# probe warnings that were never printed (#2188 review).
@test "no attempt because the draw failed names the draw, not the pool" {
    ATTEMPTS=$(attempts skipped skipped skipped) ROUTE_OUTCOME=failure _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"the draw itself failed"* ]] || false
    [[ "$output" != *"no member of the review pool answered"* ]] || false
}

@test "an ATTEMPTS list with nothing evaluated is no attempt, not a success" {
    ATTEMPTS="$(printf ' hk\n m\n hk\n')" _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"no review attempt ran"* ]] || false
}

@test "a second attempt that reviewed after a first failure passes" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS="$REVIEW"
    ATTEMPTS=$(attempts skipped failure success) _guard
    [ "$status" -eq 0 ]
}
