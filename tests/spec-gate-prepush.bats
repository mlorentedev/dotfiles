#!/usr/bin/env bats
# Tests for scripts/spec-gate-prepush.sh (BUG-061).
#
# The defect this exists to prevent: check-spec-gate.sh's archive-on-merge
# credit only fires when SDD_PR_BODY names a closing keyword, and that variable
# is empty on every local pre-push run by design. A PR that correctly archived
# its spec in the same change therefore could not be pushed -- the LOC gate saw
# no active-spec touch and rejected a change the author had done exactly right.
#
# Unlike scripts/spec-gate-pr.sh (its CI sibling, which fails CLOSED on a bad
# live read to avoid deciding on a stale event-payload replay -- BUG-066), this
# adapter falls THROUGH to running the gate with no PR context on any failure
# to resolve one. "No PR context" is the ordinary local baseline, not an error
# condition, so a missing `gh`/`jq`, an unauthenticated `gh`, or no PR open yet
# for the branch must all degrade to today's exact behaviour rather than block.

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    FIX="/tmp/bats_specgateprepush_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$FIX/bin"

    # The adapter execs check-spec-gate.sh from its OWN directory, so the fake
    # gate must sit beside a COPY of the adapter -- putting it on PATH would not
    # be reached, and that is the property we want to keep true.
    cp "$SCRIPTS_DIR/spec-gate-prepush.sh" "$FIX/bin/spec-gate-prepush.sh"
    ADAPTER="$FIX/bin/spec-gate-prepush.sh"

    # Present even under a restricted PATH: the adapter's `#!/usr/bin/env bash`
    # shebang has `env` look `bash` up on PATH, and `dirname`/`cat` are the two
    # external tools the adapter itself shells out to (SCRIPT_DIR resolution,
    # the --help text). The missing-gh/missing-jq tests below deliberately
    # shrink PATH to just this directory, so all three must live here too, or
    # they would fail on "command not found" rather than on the thing under test.
    for _tool in bash dirname cat; do
        ln -sf "$(command -v "$_tool")" "$FIX/bin/$_tool"
    done

    GATE_LOG="$FIX/gate.log"
    export GATE_LOG

    # Gate stub: records the forwarded argv and the three env vars it was handed.
    cat > "$FIX/bin/check-spec-gate.sh" <<'STUB'
#!/usr/bin/env bash
{
    printf 'ARGS=[%s]\n' "$*"
    printf 'SDD_LABELS=[%s]\n' "${SDD_LABELS-<unset>}"
    printf 'SDD_PR_AUTHOR=[%s]\n' "${SDD_PR_AUTHOR-<unset>}"
    printf 'SDD_PR_BODY=[%s]\n' "${SDD_PR_BODY-<unset>}"
} >> "$GATE_LOG"
exit "${STUB_GATE_RC:-0}"
STUB
    chmod +x "$FIX/bin/check-spec-gate.sh"

    # gh stub: no --pr number is ever passed by this adapter, so record the
    # exact argv to prove that. STUB_PR_JSON is the answer, STUB_GH_RC forces
    # the no-PR-for-this-branch failure path.
    cat > "$FIX/bin/gh" <<'STUB'
#!/usr/bin/env bash
printf 'GH_ARGS=[%s]\n' "$*" >> "$GH_LOG"
if [[ "${STUB_GH_RC:-0}" -ne 0 ]]; then
    printf 'no pull requests found for branch\n' >&2
    exit "$STUB_GH_RC"
fi
printf '%s' "$STUB_PR_JSON"
STUB
    chmod +x "$FIX/bin/gh"

    GH_LOG="$FIX/gh.log"
    export GH_LOG
    PATH="$FIX/bin:$PATH"
    export PATH
    export STUB_PR_JSON='{"labels":[],"body":null,"author":{"login":"mlorentedev"}}'
}

teardown() {
    cd / || true
    rm -rf "$FIX"
}

@test "spec-gate-prepush: derives labels, body and author from the live gh read" {
    export STUB_PR_JSON='{"labels":[{"name":"skip-archive"},{"name":"bug"}],"body":"## Archive skip rationale\n\nThe spec stays active on purpose.","author":{"login":"mlorentedev"}}'
    run "$ADAPTER" --base-ref origin/main --head-ref HEAD --explain
    [ "$status" -eq 0 ]
    grep -qF 'SDD_LABELS=[skip-archive,bug]' "$GATE_LOG"
    grep -qF 'SDD_PR_AUTHOR=[mlorentedev]' "$GATE_LOG"
    grep -qF 'Archive skip rationale' "$GATE_LOG"
}

@test "spec-gate-prepush: resolves the CURRENT branch's PR, no --pr number involved" {
    run "$ADAPTER" --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 0 ]
    grep -qF -- '--json labels,body,author' "$GH_LOG"
    run grep -c -- '-p ' "$GH_LOG"
    [ "$output" -eq 0 ]
}

@test "spec-gate-prepush: forwards every argument to the gate verbatim" {
    run "$ADAPTER" --base-ref origin/main --head-ref HEAD --explain \
        --adjacency-issues /tmp/adjacency.tsv
    [ "$status" -eq 0 ]
    grep -qF 'ARGS=[--base-ref origin/main --head-ref HEAD --explain --adjacency-issues /tmp/adjacency.tsv]' "$GATE_LOG"
}

@test "spec-gate-prepush: no PR for this branch falls through, gate still runs with no PR context" {
    export STUB_GH_RC=1
    run "$ADAPTER" --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 0 ]
    grep -qF 'SDD_LABELS=[<unset>]' "$GATE_LOG"
    grep -qF 'SDD_PR_BODY=[<unset>]' "$GATE_LOG"
    grep -qF 'SDD_PR_AUTHOR=[<unset>]' "$GATE_LOG"
}

@test "spec-gate-prepush: missing gh falls through without invoking it" {
    # A restricted PATH, not just deleting the stub: the real system `gh`
    # (further down the inherited PATH) must not be reachable either, or this
    # test would pass by accident against a tool that answers for real.
    rm -f "$FIX/bin/gh"
    PATH="$FIX/bin" run "$ADAPTER" --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 0 ]
    [ ! -f "$GH_LOG" ]
    grep -qF 'SDD_LABELS=[<unset>]' "$GATE_LOG"
}

@test "spec-gate-prepush: missing jq falls through without ever calling gh" {
    # $FIX/bin is self-contained -- no jq in it, and the adapter needs nothing
    # else from PATH before it decides to fall through (the gate stub is a
    # script invoked by absolute path via `exec`, not looked up on PATH).
    cat > "$FIX/bin/gh" <<'STUB'
#!/usr/bin/env bash
printf 'GH_ARGS=[%s]\n' "$*" >> "$GH_LOG"
exit 1
STUB
    chmod +x "$FIX/bin/gh"
    PATH="$FIX/bin" run "$ADAPTER" --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 0 ]
    [ ! -f "$GH_LOG" ]
    grep -qF 'SDD_LABELS=[<unset>]' "$GATE_LOG"
}

@test "spec-gate-prepush: a null body and no labels become empty, not the string null" {
    export STUB_PR_JSON='{"labels":[],"body":null,"author":{"login":"dependabot[bot]"}}'
    run "$ADAPTER" --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 0 ]
    grep -qF 'SDD_LABELS=[]' "$GATE_LOG"
    grep -qF 'SDD_PR_BODY=[]' "$GATE_LOG"
    grep -qF 'SDD_PR_AUTHOR=[dependabot[bot]]' "$GATE_LOG"
}

@test "spec-gate-prepush: propagates the gate verdict instead of masking it" {
    export STUB_GATE_RC=1
    run "$ADAPTER" --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 1 ]
}

@test "spec-gate-prepush: --help exits 0 and documents the fall-through" {
    run "$ADAPTER" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Falls through"* ]]
}

@test "the pre-commit config wires sdd-spec-gate through this adapter, not the gate directly" {
    # Guards the wiring itself: check-spec-gate.sh alone cannot resolve a live
    # PR, so if this entry ever points back at it directly, BUG-061 is back --
    # silently, since nothing else would fail (bitacora-reconcile's lesson:
    # config logic unreachable by tests goes red-and-silent).
    run grep -A8 'id: sdd-spec-gate' "$BATS_TEST_DIRNAME/../.pre-commit-config.yaml"
    [ "$status" -eq 0 ]
    [[ "$output" == *"entry: ./scripts/spec-gate-prepush.sh"* ]]
    [[ "$output" != *"entry: ./scripts/check-spec-gate.sh"* ]]
}
