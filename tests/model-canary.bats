#!/usr/bin/env bats
# Tests for scripts/model-canary.sh and .github/workflows/model-canary.yml
# (AI-045 AC8, #1860).
#
# The script maps the canary's exit status onto ONE issue. These execute it,
# under `bash -e` as Actions does, with a stub canary and a stub gh, because the
# exit-1 branch is exactly what an inline `run:` block makes unreachable
# (BUG-063).

load 'lib/refute'

setup() {
    SCRIPT="$BATS_TEST_DIRNAME/../scripts/model-canary.sh"
    WORKFLOW="$BATS_TEST_DIRNAME/../.github/workflows/model-canary.yml"
    FIX="/tmp/bats_canary_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$FIX/bin"

    GH_LOG="$FIX/gh-calls.log"
    export GH_LOG
    export GH_REPO="owner/repo"
    unset GITHUB_STEP_SUMMARY GITHUB_RUN_ID GITHUB_SERVER_URL

    # gh stub: records every call; `issue list` answers STUB_EXISTING, and the
    # subcommand named in STUB_FAIL (e.g. "issue close") exits 1.
    cat > "$FIX/bin/gh" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$GH_LOG"
[ "$1 $2" = "${STUB_FAIL:-}" ] && exit 1
case "$1 $2" in
    "issue list") printf '%s' "${STUB_EXISTING:-}" ;;
esac
exit 0
STUB
    chmod +x "$FIX/bin/gh"
    PATH="$FIX/bin:$PATH"
    export PATH

    # Canary stub: writes a report where --report points, exits STUB_RC.
    CANARY="$FIX/bin/dotf"
    cat > "$CANARY" <<'STUB'
#!/usr/bin/env bash
while [ $# -gt 0 ]; do
    if [ "$1" = "--report" ]; then printf '| `mimo-v2.5` | refused (HTTP 401) |\n' > "$2"; fi
    shift
done
exit "${STUB_RC:-0}"
STUB
    chmod +x "$CANARY"
}

teardown() {
    rm -rf "$FIX"
}

@test "canary: a refused model opens the issue and the run goes red" {
    STUB_RC=1 run bash -e "$SCRIPT" "$CANARY" .
    [ "$status" -eq 1 ]
    grep -q '^issue create .*--label model-canary' "$GH_LOG"
    refute_grep_fixed 'api -X PATCH' "$GH_LOG"
}

@test "canary: a second failing run rewrites the open issue instead of filing another" {
    STUB_RC=1 STUB_EXISTING=42 run bash -e "$SCRIPT" "$CANARY" .
    [ "$status" -eq 1 ]
    grep -q '^api -X PATCH repos/owner/repo/issues/42 ' "$GH_LOG"
    refute_grep_fixed 'issue create' "$GH_LOG"
}

@test "canary: a clean run closes the open issue" {
    STUB_RC=0 STUB_EXISTING=42 run bash -e "$SCRIPT" "$CANARY" .
    [ "$status" -eq 0 ]
    grep -q '^issue comment 42 ' "$GH_LOG"
    grep -q '^issue close 42$' "$GH_LOG"
}

@test "canary: a clean run whose issue cannot be closed says so instead of exiting bare" {
    STUB_RC=0 STUB_EXISTING=42 STUB_FAIL="issue close" run bash -e "$SCRIPT" "$CANARY" .
    [ "$status" -eq 2 ]
    [[ "$output" == *"::error::every bound model answered, but #42 could not be closed"* ]] || false
}

@test "canary: a failing run whose issue cannot be written says so instead of exiting bare" {
    STUB_RC=1 STUB_EXISTING=42 STUB_FAIL="api -X" run bash -e "$SCRIPT" "$CANARY" .
    [ "$status" -eq 2 ]
    [[ "$output" == *"::error::a bound model is not answering, but #42 could not be updated"* ]] || false
}

@test "canary: a clean run with no open issue touches nothing" {
    STUB_RC=0 run bash -e "$SCRIPT" "$CANARY" .
    [ "$status" -eq 0 ]
    refute_grep_fixed 'issue create' "$GH_LOG"
    refute_grep_fixed 'issue close' "$GH_LOG"
}

@test "canary: a canary that could not run leaves the issue alone and keeps its status" {
    STUB_RC=2 STUB_EXISTING=42 run bash -e "$SCRIPT" "$CANARY" .
    [ "$status" -eq 2 ]
    # Not even listed: a run that could not look says nothing about the models.
    [ ! -f "$GH_LOG" ]
}

@test "canary: the report reaches the job summary" {
    export GITHUB_STEP_SUMMARY="$FIX/summary.md"
    STUB_RC=1 run bash -e "$SCRIPT" "$CANARY" .
    grep -qF 'mimo-v2.5' "$GITHUB_STEP_SUMMARY"
}

@test "canary workflow: daily and on demand, NaN key, issue writes, script as the one-line step" {
    run python3 - "$WORKFLOW" <<'PY'
import sys, yaml
w = yaml.safe_load(open(sys.argv[1]))
on = w.get("on") or w.get(True)
steps = w["jobs"]["canary"]["steps"]
probe = [s for s in steps if "model-canary.sh" in s.get("run", "")]
print(bool(on.get("schedule")), "workflow_dispatch" in on,
      w["permissions"].get("issues"),
      len(probe) == 1 and probe[0]["env"].get("NAN_API_KEY") == "${{ secrets.NAN_API_KEY }}",
      len(probe) == 1 and "\n" not in probe[0]["run"].strip())
PY
    [ "$status" -eq 0 ]
    [ "$output" = "True True write True True" ]
}
