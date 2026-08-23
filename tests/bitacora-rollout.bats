#!/usr/bin/env bats
# Tests for the backfill path of scripts/bitacora-rollout.sh (#884, #887).
#
# The defect these exist to prevent: the backfill wrote to the board with
# `gh project item-add --owner`, a form that makes the gh CLI resolve the
# owner's TYPE first. BITACORA_PAT cannot perform that lookup, so every item
# failed with `unknown owner type` — while the event-driven add-to-project.yml
# stayed green with the SAME token, because its GraphQL mutation never resolves
# an owner. The reconciler shipped with an invocation form nobody had exercised
# against the credential it runs under, and failed all three of its first runs.
#
# So these tests do not read the script's text; they RUN the backfill against a
# gh stub and assert which call it makes. A test that only checked "an item was
# added" would have passed against the broken version too, since the stub would
# happily answer either form.

load 'lib/refute'

setup() {
    SCRIPT="$BATS_TEST_DIRNAME/../scripts/bitacora-rollout.sh"
    FIX="/tmp/bats_rollout_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$FIX/bin"

    GH_LOG="$FIX/gh-calls.log"
    export GH_LOG

    # gh stub: logs every invocation, then answers the three GraphQL shapes and
    # the two listings the backfill uses. STUB_* steer the failure branches.
    cat > "$FIX/bin/gh" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$GH_LOG"
all="$*"
case "$all" in
    *addProjectV2ItemById*)
        exit "${STUB_MUTATION_RC:-0}" ;;
    *"projectV2(number: 1) { id }"*)
        [ -n "${STUB_NO_PROJECT:-}" ] && exit 0
        printf 'PVT_test123\n' ;;
    *"repositories(first: 100)"*)
        printf '%s\n' "${STUB_LINKED:-}" ;;
esac
# Real gh terminates its output with a newline and prints nothing at all when
# the list is empty. A stub that drops the trailing newline makes `while read`
# discard the final item — a failure the live script does not have.
case "$1 $2" in
    "issue list") [ -n "${STUB_ISSUES:-}" ] && printf '%s\n' "$STUB_ISSUES" ;;
    "pr list")    [ -n "${STUB_PRS:-}" ] && printf '%s\n' "$STUB_PRS" ;;
esac
exit 0
STUB
    chmod +x "$FIX/bin/gh"
    PATH="$FIX/bin:$PATH"
    export PATH
}

teardown() {
    rm -rf "$FIX"
}

# Two open issues, as the id+url pairs the listing now yields.
default_issues() {
    export STUB_ISSUES='I_node1 https://github.com/mlorentedev/demo/issues/1
I_node2 https://github.com/mlorentedev/demo/issues/2'
}

count_calls() {
    grep -c "$1" "$GH_LOG" 2>/dev/null || true
}

# ── the regression ────────────────────────────────────────────────────────────

@test "the regression: backfill never calls 'gh project' at all" {
    default_issues
    run "$SCRIPT" --backfill-only demo
    [ "$status" -eq 0 ]
    # `gh project item-add --owner` is the form BITACORA_PAT cannot execute.
    # Nothing in the backfill may reach for it again.
    refute_grep '^project ' "$GH_LOG"
}

@test "the regression: each open item is added via addProjectV2ItemById" {
    default_issues
    run "$SCRIPT" --backfill-only demo
    [ "$status" -eq 0 ]
    [ "$(count_calls 'addProjectV2ItemById')" -eq 2 ]
    grep -q 'contentId=I_node1' "$GH_LOG"
    grep -q 'contentId=I_node2' "$GH_LOG"
}

@test "the regression: the listing asks for the node id, not just the url" {
    default_issues
    run "$SCRIPT" --backfill-only demo
    [ "$status" -eq 0 ]
    # The mutation needs a node ID. Requesting it from the listing that has to
    # happen anyway is what keeps this at zero extra round-trips per item.
    grep -q 'issue list .*--json id,url' "$GH_LOG"
    grep -q 'pr list .*--json id,url' "$GH_LOG"
}

@test "the project node id is resolved once per run, not once per repo" {
    default_issues
    run "$SCRIPT" --backfill-only demo other third
    [ "$status" -eq 0 ]
    [ "$(count_calls 'projectV2(number: 1) { id }')" -eq 1 ]
}

@test "the mutation carries the resolved project id" {
    default_issues
    run "$SCRIPT" --backfill-only demo
    [ "$status" -eq 0 ]
    grep -q 'project=PVT_test123' "$GH_LOG"
}

# ── failure reporting (#887) ──────────────────────────────────────────────────

@test "every add failing reports the repo as failed, not as a green zero" {
    default_issues
    STUB_MUTATION_RC=1 run "$SCRIPT" --backfill-only demo
    [ "$status" -ne 0 ]
    # The bug: '0 open item(s) ensured' with a leading tick, identical to the
    # healthy no-op. The summary must name the failures.
    [[ "$output" != *"✅ demo: backfill"* ]]
    [[ "$output" == *"0 ensured, 2 failed"* ]]
}

@test "a partial failure is still reported as a failure" {
    export STUB_ISSUES='I_node1 https://github.com/mlorentedev/demo/issues/1'
    export STUB_PRS='I_node2 https://github.com/mlorentedev/demo/pull/2'
    # Fail only the second mutation.
    cat > "$FIX/bin/gh" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$GH_LOG"
all="$*"
case "$all" in
    *addProjectV2ItemById*)
        n=$(grep -c addProjectV2ItemById "$GH_LOG")
        [ "$n" -ge 2 ] && exit 1
        exit 0 ;;
    *"projectV2(number: 1) { id }"*) printf 'PVT_test123\n' ;;
esac
# Real gh terminates its output with a newline and prints nothing at all when
# the list is empty. A stub that drops the trailing newline makes `while read`
# discard the final item — a failure the live script does not have.
case "$1 $2" in
    "issue list") [ -n "${STUB_ISSUES:-}" ] && printf '%s\n' "$STUB_ISSUES" ;;
    "pr list")    [ -n "${STUB_PRS:-}" ] && printf '%s\n' "$STUB_PRS" ;;
esac
exit 0
STUB
    chmod +x "$FIX/bin/gh"
    run "$SCRIPT" --backfill-only demo
    [ "$status" -ne 0 ]
    [[ "$output" == *"1 ensured, 1 failed"* ]]
}

@test "a repo with nothing to add stays green - the healthy no-op" {
    export STUB_ISSUES=''
    export STUB_PRS=''
    run "$SCRIPT" --backfill-only demo
    [ "$status" -eq 0 ]
    [[ "$output" == *"✅ demo: backfill — 0 open item(s) ensured on board"* ]]
}

# ── loud abort ────────────────────────────────────────────────────────────────

@test "an unresolvable project id aborts before touching any repo" {
    default_issues
    STUB_NO_PROJECT=1 run "$SCRIPT" --backfill-only demo
    [ "$status" -ne 0 ]
    [[ "$output" == *"cannot resolve project #1"* ]]
    # The point of aborting: 27 repos of identical per-item errors would bury
    # the one line that explains them.
    [ "$(count_calls 'addProjectV2ItemById')" -eq 0 ]
    [[ "$output" != *"item-add failed"* ]]
}
