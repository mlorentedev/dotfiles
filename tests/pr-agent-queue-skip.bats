#!/usr/bin/env bats
# The PR-Agent review step skips a PR that was merged, closed or turned back
# into a draft while its run waited in the one-at-a-time lane (#1923). These run
# the REAL `reviewable` step script, extracted from the workflow, against a stub
# gh: the skip is decided by the step's first call, and a definite answer is the
# only thing that skips.
#
# Each test sets its stub's answers with `export`, which bats scopes to that
# test's own subshell by design.
# shellcheck disable=SC2030,SC2031

bats_require_minimum_version 1.5.0

setup() {
    WF="$BATS_TEST_DIRNAME/../.github/workflows/pr-agent.yml"
    STEP="$BATS_TEST_TMPDIR/reviewable.sh"
    python3 - "$WF" > "$STEP" <<'PY'
import sys, yaml
steps = yaml.safe_load(open(sys.argv[1]))["jobs"]["review"]["steps"]
print(next(s for s in steps if s.get("id") == "reviewable")["run"])
PY
    [ -s "$STEP" ] || { echo "the reviewable step is gone from $WF" >&2; return 1; }

    STUB="$BATS_TEST_TMPDIR/bin"
    CALLS="$BATS_TEST_TMPDIR/calls"
    OUT="$BATS_TEST_TMPDIR/github_output"
    mkdir -p "$STUB"
    : > "$CALLS"
    : > "$OUT"
    # Answers the PR read from STUB_PULL ("fail" fails it); every other call is
    # logged and fails, so a run that goes past the gate stops at its next read.
    cat > "$STUB/gh" <<'SH'
#!/usr/bin/env bash
echo "$2" >> "$CALLS"
case "$2" in
    */pulls/7)
        [ "${STUB_PULL:-}" = fail ] && { echo "gh: Gateway Timeout (HTTP 504)" >&2; exit 1; }
        printf '%s\n' "$STUB_PULL"
        ;;
    *) exit 1 ;;
esac
SH
    chmod +x "$STUB/gh"
    export CALLS
}

_step() { # $1 = event name
    run env PATH="$STUB:$PATH" CALLS="$CALLS" GITHUB_OUTPUT="$OUT" \
        GITHUB_EVENT_NAME="$1" GITHUB_REPOSITORY=o/r PR_NUMBER=7 \
        DEFAULT_BRANCH=main WORKFLOW_SHA=abc STUB_PULL="${STUB_PULL-}" \
        bash -e "$STEP"
}

@test "a PR merged, closed or drafted while queued is skipped, after one read" {
    for state in merged closed draft; do
        : > "$CALLS"; : > "$OUT"
        STUB_PULL=$state _step pull_request
        [ "$status" -eq 0 ] || { echo "$state: $output"; false; }
        [[ "$output" == *"became $state while this review waited in the queue"* ]]
        grep -qx 'reviewable=false' "$OUT"
        [ "$(wc -l < "$CALLS")" -eq 1 ] || { echo "$state: went past the gate:"; cat "$CALLS"; false; }
    done
}

@test "an open PR goes on to the empty-diff check" {
    STUB_PULL=open _step pull_request
    run ! grep -q 'reviewable=false' "$OUT"
    grep -q 'contents/.github/workflows/pr-agent.yml' "$CALLS"
}

@test "a failed state read does not skip: the review runs, as before" {
    STUB_PULL=fail _step pull_request
    run ! grep -q 'reviewable=false' "$OUT"
    grep -q 'contents/.github/workflows/pr-agent.yml' "$CALLS"
}

@test "a /review comment on a merged PR is not skipped: a human asked" {
    STUB_PULL=merged _step issue_comment
    run ! grep -q 'reviewable=false' "$OUT"
    run ! grep -q 'pulls/7$' "$CALLS"
}
