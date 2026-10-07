#!/usr/bin/env bats
# Behaviour of the pr-agent publication guard ("Fail if no review was published"
# in .github/workflows/pr-agent.yml) when the GitHub API fails (#2069).
#
# A read timeout on the registry used to become an empty marker list, and the
# guard then failed a reviewed PR with "no review marker declared": the wrong
# diagnosis, measured downstream on leaving-denver run 37155990064. These run the
# step's REAL script, extracted from the workflow, against a stub gh whose every
# answer is set per test. pr-agent-config.bats pins the step's text; this file
# pins what it does.
#
# Each test sets its stub's answers with `export`, which bats scopes to that
# test's own subshell by design.
# shellcheck disable=SC2030,SC2031

setup() {
    WF="$BATS_TEST_DIRNAME/../.github/workflows/pr-agent.yml"
    GUARD="$BATS_TEST_TMPDIR/guard.sh"
    python3 - "$WF" > "$GUARD" <<'PY'
import sys, yaml
steps = yaml.safe_load(open(sys.argv[1]))["jobs"]["review"]["steps"]
print(next(s for s in steps if s.get("name") == "Fail if no review was published")["run"])
PY
    [ -s "$GUARD" ] || { echo "the guard step is gone from $WF" >&2; return 1; }

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
path="$2"
case "$path" in
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

# Runs the guard as the workflow does: bash, errexit and pipefail, the step's env.
_guard() {
    run env PATH="$STUB:$PATH" CALLS="$CALLS" GUARD_RETRY_SECONDS=0 \
        GITHUB_REPOSITORY=o/r PR_NUMBER=7 BASE_REF=main STARTED=2026-10-07T09:00:00Z \
        HEAD_SHA="${HEAD_SHA-abc123}" PR_AGENT_OUTCOME=success REVIEW_MODEL=m \
        bash -eo pipefail "$GUARD"
}

calls_to() { grep -c "^$1\$" "$CALLS" || true; }

@test "a registry read that keeps failing is reported as an API failure, not a missing marker" {
    export STUB_BASE=fail:504 STUB_HEAD=fail:504
    _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"the GitHub API failed while determining whether a review marker is declared"* ]]
    [[ "$output" == *"review-attestation.json?ref=main"* ]]
    [[ "$output" == *"HTTP 504"* ]]
    [[ "$output" != *"no review marker declared"* ]]
    [ "$(calls_to BASE)" -eq 3 ]
}

@test "a registry read that fails once is retried, and the answer is not prefixed by the failed body" {
    export STUB_BASE=flaky:502 STUB_BASE_OK="$REGISTRY" STUB_COMMENTS="$REVIEW"
    _guard
    [ "$status" -eq 0 ]
    [[ "$output" == *'review published (markers: ["PR Reviewer Guide"])'* ]]
    [ "$(calls_to BASE)" -eq 2 ]
}

@test "a registry with no github-actions entry still fails with the existing message" {
    export STUB_BASE="$NO_ENTRY" STUB_HEAD="$NO_ENTRY"
    _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"no review marker declared for github-actions"* ]]
    [[ "$output" != *"GitHub API failed"* ]]
}

@test "a 404 is an answer, not an outage: no retry, and the head-ref fallback still applies" {
    export STUB_BASE=fail:404 STUB_HEAD="$REGISTRY" STUB_COMMENTS="$REVIEW"
    _guard
    [ "$status" -eq 0 ]
    [[ "$output" == *"reviewer registry entry found only on the PR head"* ]]
    [ "$(calls_to BASE)" -eq 1 ]
}

@test "a failed head-SHA lookup on an issue_comment run is reported as an API failure" {
    export STUB_BASE="$NO_ENTRY" STUB_PULL=fail:503
    HEAD_SHA="" _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"the GitHub API failed while determining whether a review marker is declared"* ]]
    [[ "$output" == *"pulls/7"* ]]
    [[ "$output" != *"no review marker declared"* ]]
    [ "$(calls_to PULL)" -eq 3 ]
}

@test "a comment listing that keeps failing is reported as an API failure, not a missing review" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS=fail:500
    _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"the GitHub API failed while listing the PR's comments"* ]]
    [[ "$output" != *"published no review"* ]]
}

@test "a comment listing that answers 404 is reported as such: no retry, no empty outage report" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS=fail:404
    _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"listing the PR's comments answered 404"* ]]
    [ "$(calls_to COMMENTS)" -eq 1 ]
    [[ "$output" != *"GitHub API failed"* ]]
}

@test "with every call answering and no review comment, the published-no-review diagnosis stands" {
    export STUB_BASE="$REGISTRY" STUB_COMMENTS='[]'
    _guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"PR-Agent reported success but published no review"* ]]
}
