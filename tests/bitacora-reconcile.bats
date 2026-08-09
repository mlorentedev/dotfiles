#!/usr/bin/env bats
# Tests for scripts/bitacora-reconcile.sh (BUG-063).
#
# The defect these exist to prevent: the classification used to live inline in
# .github/workflows/bitacora-reconcile.yml, which Actions runs as `bash -e {0}`.
# `set -uo pipefail` does not clear that injected -e, so the step died at the
# rollout capture and every branch below — the rate-limit soft-pass and the
# self-reporting issue — was unreachable dead code. It went red and silent twice.
#
# So these tests do not inspect the workflow's text; they EXECUTE the classifier,
# and the load-bearing ones execute it under `bash -e` (see "the regression").

setup() {
    SCRIPT="$BATS_TEST_DIRNAME/../scripts/bitacora-reconcile.sh"
    FIX="/tmp/bats_reconcile_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$FIX/bin"

    GH_LOG="$FIX/gh-calls.log"
    export GH_LOG

    # gh stub: records every invocation, and answers `issue list` from
    # STUB_EXISTING so the dedupe branch can be steered.
    cat > "$FIX/bin/gh" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$GH_LOG"
case "$1 $2" in
    "issue list") printf '%s' "${STUB_EXISTING:-}" ;;
esac
exit 0
STUB
    chmod +x "$FIX/bin/gh"
    PATH="$FIX/bin:$PATH"
    export PATH

    # Rollout stub: exit code and output are the two axes the classifier reads.
    ROLLOUT="$FIX/bin/rollout.sh"
    cat > "$ROLLOUT" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "${STUB_OUT:-rollout output}"
exit "${STUB_RC:-0}"
STUB
    chmod +x "$ROLLOUT"
    export BITACORA_ROLLOUT="$ROLLOUT"
}

teardown() {
    rm -rf "$FIX"
}

# `grep -s` so this holds when gh was never called at all and the log is absent —
# which is itself the assertion on the soft-pass paths.
refute_issue_filed() {
    ! grep -qs "issue create" "$GH_LOG"
}

@test "success: notice and exit 0" {
    STUB_RC=0 run "$SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" == *"::notice::bitácora reconciled"* ]]
    refute_issue_filed
}

@test "rate limit: warning, exit 0, and no issue filed" {
    STUB_RC=1 STUB_OUT="gh: API rate limit exceeded for user" run "$SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" == *"::warning::reconciler hit the primary GraphQL pool"* ]]
    refute_issue_filed
}

@test "rate limit: the 'already exceeded' wording is matched too" {
    STUB_RC=1 STUB_OUT="API rate limit already exceeded" run "$SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" == *"::warning::"* ]]
}

@test "other failure with no existing issue: files one and exits 1" {
    STUB_RC=1 STUB_OUT="something else broke" STUB_EXISTING="" run "$SCRIPT"
    [ "$status" -eq 1 ]
    [[ "$output" == *"::error::bitácora reconciliation failed"* ]]
    grep -q "issue create" "$GH_LOG"
}

@test "other failure with an existing issue: comments instead of creating" {
    STUB_RC=1 STUB_OUT="something else broke" STUB_EXISTING="42" run "$SCRIPT"
    [ "$status" -eq 1 ]
    grep -q "issue comment 42" "$GH_LOG"
    refute_issue_filed
}

# --- the regression -------------------------------------------------------
#
# Actions supplies the -e; the script never sees it in its own `set` line. Each
# of these invokes the classifier exactly the way the runner does. Against the
# pre-BUG-063 inline logic every one of them fails: the step aborts at the
# capture, printing nothing and filing nothing.

@test "the regression: under bash -e, a failing rollout still reaches the classifier" {
    STUB_RC=1 STUB_OUT="something else broke" run bash -e "$SCRIPT"
    [ "$status" -eq 1 ]
    [[ "$output" == *"::error::bitácora reconciliation failed"* ]]
    grep -q "issue create" "$GH_LOG"
}

@test "the regression: under bash -e, the rate limit still soft-passes green" {
    STUB_RC=1 STUB_OUT="API rate limit exceeded" run bash -e "$SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" == *"::warning::"* ]]
}

@test "the regression: rollout output is printed before the verdict, so the log keeps the evidence" {
    STUB_RC=1 STUB_OUT="DIAGNOSTIC MARKER" run bash -e "$SCRIPT"
    [[ "$output" == *"DIAGNOSTIC MARKER"* ]]
    # Evidence first, verdict second — the ordering that was lost.
    [[ "${output%%::error::*}" == *"DIAGNOSTIC MARKER"* ]]
}

# --- input validation -----------------------------------------------------

@test "a repo name outside the allowed charset is refused" {
    TARGET_REPOS='good; rm -rf /' run bash -e "$SCRIPT"
    [ "$status" -eq 2 ]
    [[ "$output" == *"refusing repo name"* ]]
}

@test "an empty TARGET_REPOS reconciles every repo without tripping set -u" {
    STUB_RC=0 TARGET_REPOS="" run bash -e "$SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" == *"::notice::"* ]]
}

@test "several valid repo names are passed through as separate arguments" {
    cat > "$BITACORA_ROLLOUT" <<'STUB'
#!/usr/bin/env bash
printf 'args:%s\n' "$*"
exit 0
STUB
    chmod +x "$BITACORA_ROLLOUT"
    TARGET_REPOS="dotfiles kubelab" run bash -e "$SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" == *"args:--backfill-only dotfiles kubelab"* ]]
}

# --- the reporting path must survive -e too (CodeRabbit on PR #870) ----------
#
# The first cut of this script reproduced the very defect it fixes: the dedupe
# lookup was a bare `existing=$(gh issue list ...)`, so under `bash -e` a failing
# gh aborted the script before the ::error:: and before anything was filed. The
# original suite missed it because every stub gh SUCCEEDED.

_gh_failing_on() {
    cat > "$FIX/bin/gh" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "\$GH_LOG"
case "\$1 \$2" in
    "$1") printf 'gh: simulated failure\n' >&2; exit 1 ;;
    "issue list") printf '%s' "\${STUB_EXISTING:-}" ;;
esac
exit 0
STUB
    chmod +x "$FIX/bin/gh"
}

@test "under bash -e, a failing issue-list lookup still reaches the error verdict" {
    _gh_failing_on "issue list"
    STUB_RC=1 STUB_OUT="something else broke" run bash -e "$SCRIPT"
    [ "$status" -eq 1 ]
    [[ "$output" == *"::error::bitácora reconciliation failed"* ]]
}

@test "a failed lookup refuses to file, so the stable-title issue is never duplicated" {
    _gh_failing_on "issue list"
    STUB_RC=1 STUB_OUT="something else broke" run bash -e "$SCRIPT"
    [[ "$output" == *"not filing, to avoid a duplicate"* ]]
    refute_issue_filed
}

@test "a failed issue create reports the failure instead of claiming success" {
    _gh_failing_on "issue create"
    STUB_RC=1 STUB_OUT="something else broke" STUB_EXISTING="" run bash -e "$SCRIPT"
    [ "$status" -eq 1 ]
    [[ "$output" == *"could not open a bitacora-reconcile issue"* ]]
    [[ "$output" != *"::warning::opened a bitacora-reconcile issue"* ]]
}

@test "a failed issue comment reports the failure instead of claiming success" {
    _gh_failing_on "issue comment"
    STUB_RC=1 STUB_OUT="something else broke" STUB_EXISTING="42" run bash -e "$SCRIPT"
    [ "$status" -eq 1 ]
    [[ "$output" == *"could not comment on issue #42"* ]]
    [[ "$output" != *"::notice::updated existing issue"* ]]
}

# --- TARGET_REPOS must not glob (CodeRabbit on PR #870) ---------------------

@test "TARGET_REPOS='*' is refused, not expanded against the working directory" {
    mkdir -p "$FIX/cwd" && touch "$FIX/cwd/dotfiles" "$FIX/cwd/kubelab"
    cd "$FIX/cwd" || return 1
    TARGET_REPOS='*' run bash -e "$SCRIPT"
    [ "$status" -eq 2 ]
    [[ "$output" == *"refusing repo name"* ]]
}
