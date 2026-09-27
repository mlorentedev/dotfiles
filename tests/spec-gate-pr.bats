#!/usr/bin/env bats
# Tests for scripts/spec-gate-pr.sh and the shape of .github/workflows/spec-gate.yml
# (BUG-066).
#
# The defect these exist to prevent: the gate read SDD_LABELS/SDD_PR_BODY from
# github.event.pull_request.*, and a re-run replays the ORIGINAL event payload.
# Its own documented escape ("skip-archive" + "## Archive skip rationale") could
# therefore not be applied to a live PR without pushing an unrelated commit.
#
# The derivation is EXECUTED here rather than grepped out of the workflow text,
# for the reason recorded in tests/bitacora-reconcile.bats: logic inside a `run:`
# block is unreachable by tests and went red-and-silent twice (BUG-063). Only the
# two facts that are declarative config -- the concurrency conditional and the
# permissions block -- are asserted by shape, because they cannot be executed.

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    WORKFLOW="$BATS_TEST_DIRNAME/../.github/workflows/spec-gate.yml"
    FIX="/tmp/bats_specgatepr_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$FIX/bin"

    # The adapter execs check-spec-gate.sh from its OWN directory, so the fake
    # gate must sit beside a COPY of the adapter -- putting it on PATH would not
    # be reached, and that is the property we want to keep true.
    cp "$SCRIPTS_DIR/spec-gate-pr.sh" "$FIX/bin/spec-gate-pr.sh"
    ADAPTER="$FIX/bin/spec-gate-pr.sh"

    GATE_LOG="$FIX/gate.log"
    export GATE_LOG

    # Gate stub: records the forwarded argv and the three env vars it was handed.
    # No log file at all means it was never invoked -- the fail-closed assertion.
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

    # gh stub: STUB_PR_JSON is the answer, STUB_GH_RC forces the failure path.
    cat > "$FIX/bin/gh" <<'STUB'
#!/usr/bin/env bash
if [[ "${STUB_GH_RC:-0}" -ne 0 ]]; then
    printf 'GraphQL: Could not resolve to a PullRequest\n' >&2
    exit "$STUB_GH_RC"
fi
printf '%s' "$STUB_PR_JSON"
STUB
    chmod +x "$FIX/bin/gh"

    PATH="$FIX/bin:$PATH"
    export PATH
    export STUB_PR_JSON='{"labels":[],"body":null,"author":{"login":"mlorentedev"}}'
}

teardown() {
    cd / || true
    rm -rf "$FIX"
}

@test "spec-gate-pr: derives labels, body and author from the live gh read" {
    export STUB_PR_JSON='{"labels":[{"name":"skip-archive"},{"name":"bug"}],"body":"## Archive skip rationale\n\nThe spec stays active on purpose.","author":{"login":"mlorentedev"}}'
    run "$ADAPTER" --pr 877 --base-ref origin/main --head-ref HEAD --explain
    [ "$status" -eq 0 ]
    grep -qF 'SDD_LABELS=[skip-archive,bug]' "$GATE_LOG"
    grep -qF 'SDD_PR_AUTHOR=[mlorentedev]' "$GATE_LOG"
    grep -qF 'Archive skip rationale' "$GATE_LOG"
}

@test "spec-gate-pr: forwards every non-pr argument to the gate verbatim" {
    run "$ADAPTER" --pr 877 --base-ref origin/main --head-ref HEAD --explain \
        --adjacency-issues /tmp/adjacency.tsv
    [ "$status" -eq 0 ]
    grep -qF 'ARGS=[--base-ref origin/main --head-ref HEAD --explain --adjacency-issues /tmp/adjacency.tsv]' "$GATE_LOG"
}

@test "spec-gate-pr: a failed live read exits 2 and never invokes the gate" {
    export STUB_GH_RC=1
    run "$ADAPTER" --pr 877 --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 2 ]
    [[ "$output" == *"could not read live metadata"* ]]
    [[ "$output" == *"Refusing to fall back to the event payload"* ]]
    [ ! -f "$GATE_LOG" ]
}

@test "spec-gate-pr: a null body and no labels become empty, not the string null" {
    export STUB_PR_JSON='{"labels":[],"body":null,"author":{"login":"dependabot[bot]"}}'
    run "$ADAPTER" --pr 1 --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 0 ]
    grep -qF 'SDD_LABELS=[]' "$GATE_LOG"
    grep -qF 'SDD_PR_BODY=[]' "$GATE_LOG"
    grep -qF 'SDD_PR_AUTHOR=[dependabot[bot]]' "$GATE_LOG"
}

@test "spec-gate-pr: propagates the gate verdict instead of masking it" {
    export STUB_GATE_RC=1
    run "$ADAPTER" --pr 1 --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 1 ]
}

@test "spec-gate-pr: exits 2 without --pr" {
    run "$ADAPTER" --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 2 ]
    [[ "$output" == *"--pr is required"* ]]
    [ ! -f "$GATE_LOG" ]
}

@test "spec-gate-pr: exits 2 when there is nothing to forward" {
    run "$ADAPTER" --pr 877
    [ "$status" -eq 2 ]
    [[ "$output" == *"nothing to forward"* ]]
}

@test "spec-gate-pr: --help exits 0 and documents the forwarding" {
    run "$ADAPTER" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"forwarded"* ]]
}

@test "spec-gate-pr: --gate runs the named checker beside it instead of check-spec-gate.sh" {
    cat > "$FIX/bin/check-knowledge-gate.sh" <<'STUB'
#!/usr/bin/env bash
printf 'KNOWLEDGE ARGS=[%s] BODY=[%s]\n' "$*" "${SDD_PR_BODY-<unset>}" >> "$GATE_LOG"
STUB
    chmod +x "$FIX/bin/check-knowledge-gate.sh"
    export STUB_PR_JSON='{"labels":[],"body":"## Knowledge","author":{"login":"mlorentedev"}}'
    run "$ADAPTER" --pr 877 --gate check-knowledge-gate.sh --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 0 ]
    grep -qF 'KNOWLEDGE ARGS=[--base-ref origin/main --head-ref HEAD] BODY=[## Knowledge]' "$GATE_LOG"
    run grep -c '^ARGS=' "$GATE_LOG"
    [ "$output" -eq 0 ]
}

@test "spec-gate-pr: a --gate value with a path separator exits 2 and runs nothing" {
    run "$ADAPTER" --pr 877 --gate ../bin/check-spec-gate.sh --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 2 ]
    [[ "$output" == *"--gate"* ]]
    [ ! -f "$GATE_LOG" ]
}

@test "spec-gate-pr: a --gate naming no executable beside the adapter exits 2" {
    run "$ADAPTER" --pr 877 --gate no-such-gate.sh --base-ref origin/main --head-ref HEAD
    [ "$status" -eq 2 ]
    [ ! -f "$GATE_LOG" ]
}

@test "spec-gate-pr: --gate without a value exits 2" {
    run "$ADAPTER" --pr 877 --base-ref origin/main --head-ref HEAD --gate
    [ "$status" -eq 2 ]
    [ ! -f "$GATE_LOG" ]
}

# End to end through the REAL knowledge checker: proves the env names the
# adapter exports are the ones the checker reads, and that its verdict survives.
_knowledge_repo() {
    cp "$SCRIPTS_DIR/check-knowledge-gate.sh" "$FIX/bin/"
    mkdir -p "$FIX/repo" && cd "$FIX/repo" || return 1
    git init -q -b main && git config user.email t@t && git config user.name t
    git config commit.gpgsign false
    echo seed > README.md && git add -A && git commit -q -m seed
    git checkout -q -b feature
    mkdir -p docs/lessons && echo new > docs/lessons/lesson-002-new.md
    git add -A && git commit -q -m lesson
}

@test "spec-gate-pr: a live body with its Knowledge section passes the real checker" {
    _knowledge_repo
    export STUB_PR_JSON='{"labels":[],"body":"## Knowledge\r\n- Lesson: docs/lessons/lesson-002-new.md\r\n- ADR: none: no decision\r\n- Runbook: none: no procedure","author":{"login":"mlorentedev"}}'
    run "$ADAPTER" --pr 1 --gate check-knowledge-gate.sh --base-ref main --head-ref feature
    [ "$status" -eq 0 ]
    [[ "$output" == *"[OK] knowledge-gate"* ]]
}

@test "spec-gate-pr: a live body without the section fails the real checker" {
    _knowledge_repo
    export STUB_PR_JSON='{"labels":[],"body":null,"author":{"login":"mlorentedev"}}'
    run "$ADAPTER" --pr 1 --gate check-knowledge-gate.sh --base-ref main --head-ref feature
    [ "$status" -eq 1 ]
    [[ "$output" == *"## Knowledge"* ]]
}

@test "spec-gate workflow: no PR metadata is sourced from the event payload" {
    run grep -c 'SDD_LABELS\|SDD_PR_BODY\|SDD_PR_AUTHOR' "$WORKFLOW"
    [ "$output" -eq 0 ]
}

@test "spec-gate workflow: metadata-only events do not cancel a run in flight" {
    grep -qF 'cancel-in-progress: ${{ !contains(' "$WORKFLOW"
    grep -qF '"labeled"' "$WORKFLOW"
    grep -qF '"unlabeled"' "$WORKFLOW"
    grep -qF '"edited"' "$WORKFLOW"
    run grep -c 'cancel-in-progress: true' "$WORKFLOW"
    [ "$output" -eq 0 ]
}

@test "spec-gate workflow: grants the pull-requests scope the live read needs" {
    grep -qF 'pull-requests: read' "$WORKFLOW"
}

@test "spec-gate workflow: the adjacency feed it collects is actually consumed" {
    grep -qF -- '--adjacency-issues' "$WORKFLOW"
}
