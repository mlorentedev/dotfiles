#!/usr/bin/env bats
# Shape of .github/workflows/knowledge-gate.yml (HARNESS-024). Declarative config
# cannot be executed, so it is asserted against spec-gate.yml, whose triggers and
# concurrency already carry BUG-066's fix: a copy that drifts from them would
# bring the bug back to the gate that is cleared by editing the body.

setup() {
    WF_DIR="$BATS_TEST_DIRNAME/../.github/workflows"
    KNOWLEDGE="$WF_DIR/knowledge-gate.yml"
    SPEC="$WF_DIR/spec-gate.yml"
}

_line() {
    grep -E "^[[:space:]]*$1" "$2"
}

@test "knowledge-gate workflow: triggers equal spec-gate's, edited included" {
    [ "$(_line 'types:' "$KNOWLEDGE")" = "$(_line 'types:' "$SPEC")" ]
    [ "$(_line 'branches:' "$KNOWLEDGE")" = "$(_line 'branches:' "$SPEC")" ]
    _line 'types:.*edited' "$KNOWLEDGE"
}

@test "knowledge-gate workflow: metadata-only events do not cancel a run in flight" {
    [ -n "$(_line 'cancel-in-progress:' "$SPEC")" ]
    [ "$(_line 'cancel-in-progress:' "$KNOWLEDGE")" = "$(_line 'cancel-in-progress:' "$SPEC")" ]
}

@test "knowledge-gate workflow: reads the PR live through the adapter, never from the payload" {
    grep -qF './scripts/spec-gate-pr.sh' "$KNOWLEDGE"
    grep -qF -- '--gate check-knowledge-gate.sh' "$KNOWLEDGE"
    run grep -c 'SDD_LABELS\|SDD_PR_BODY\|SDD_PR_AUTHOR\|pull_request.body' "$KNOWLEDGE"
    [ "$output" -eq 0 ]
}

@test "knowledge-gate workflow: grants only the scopes the live read needs" {
    grep -qF 'pull-requests: read' "$KNOWLEDGE"
    grep -qF 'contents: read' "$KNOWLEDGE"
    run grep -c ': write' "$KNOWLEDGE"
    [ "$output" -eq 0 ]
}

@test "knowledge-gate workflow: the job is named knowledge-gate, the check context protection will name" {
    grep -qE '^  knowledge-gate:$' "$KNOWLEDGE"
}
