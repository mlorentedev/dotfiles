#!/usr/bin/env bats
# OPS-023 (#809): the bitácora board silently lost items whenever the
# event-driven add hit an API failure. Nothing in this machinery had any test
# coverage, which is precisely why a drop could stay invisible for three days
# across two repos.
#
# The execution cases drive a STUB `gh` first on PATH: what is under test is the
# script's own decisions and the commands it builds, not GitHub's behaviour, and
# a stub is the only way to assert the second thing at all. The remaining cases
# pin structure in the workflow files, which cannot be executed here — following
# the same convention the hive-upgrade guard uses.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    ROLLOUT="$REPO/scripts/bitacora-rollout.sh"
    ADD_WF="$REPO/.github/workflows/add-to-project.yml"
    RECONCILE_WF="$REPO/.github/workflows/bitacora-reconcile.yml"
    WORK="$(mktemp -d)"
    STUB="$WORK/stub"
    GH_LOG="$WORK/gh-calls"
    mkdir -p "$STUB"
}

teardown() { [ -z "${WORK:-}" ] || rm -rf "$WORK"; }

# A `gh` that records every invocation and answers the few queries the script
# makes, so a run completes without network and every call is assertable.
stub_gh() {
    {
        echo '#!/usr/bin/env bash'
        printf 'printf "%%s\\n" "$*" >> %s\n' "$GH_LOG"
        cat <<'SH'
case "$*" in
    "repo list"*)         printf 'demo\n' ;;
    "issue list"*)        printf 'https://github.com/mlorentedev/demo/issues/1\n' ;;
    "pr list"*)           printf 'https://github.com/mlorentedev/demo/pull/2\n' ;;
    "project list"*)      printf 'demo\n' ;;
    *)                    : ;;
esac
exit 0
SH
    } > "$STUB/gh"
    chmod +x "$STUB/gh"
    # An `age` that fails loudly: --backfill-only must never reach the decrypt
    # step, so if it does, the test fails instead of silently passing.
    printf '#!/usr/bin/env bash\necho "age must not be called" >&2\nexit 1\n' > "$STUB/age"
    chmod +x "$STUB/age"
}

run_backfill_only() {
    run env PATH="$STUB:$PATH" BITACORA_PAT="" bash "$ROLLOUT" --backfill-only --check demo
}

@test "OPS-023: --backfill-only ensures open issues and PRs are on the board" {
    stub_gh
    run_backfill_only
    [ "$status" -eq 0 ]
    [[ "$output" == *"backfill"* ]]
    [[ "$output" == *"2 open item(s) ensured on board"* ]]
}

@test "OPS-023: --backfill-only does not provision: no link, no secret, no workflow push" {
    # Reconciliation must not be able to push workflow files or rotate a secret.
    # If it could, a scheduled job would be silently deploying code.
    stub_gh
    run_backfill_only
    [ "$status" -eq 0 ]
    [[ "$output" != *"link to project"* ]]
    [[ "$output" != *"BITACORA_PAT set"* ]]
    run cat "$GH_LOG"
    [[ "$output" != *"secret set"* ]]
    [[ "$output" != *"project link"* ]]
}

@test "OPS-023: --backfill-only never needs the age key, so it can run in CI" {
    # The stub `age` exits 1 with a message. Reaching it at all fails this test —
    # which is the point: CI has the Actions secret but no age key, so a decrypt
    # attempt would make the reconciler undeployable there.
    stub_gh
    run_backfill_only
    [ "$status" -eq 0 ]
    [[ "$output" != *"age must not be called"* ]]
    [[ "$output" != *"cannot decrypt"* ]]
}

@test "OPS-023: a full run still provisions (the flag narrows scope, not the tool)" {
    stub_gh
    run env PATH="$STUB:$PATH" BITACORA_PAT="token" bash "$ROLLOUT" --check demo
    [ "$status" -eq 0 ]
    [[ "$output" == *"link to project"* || "$output" == *"already linked"* ]]
    [[ "$output" == *"backfill"* ]]
}

# The header of these workflows explains the design at length, and that prose
# names every construct the assertions below look for. Grepping the raw file
# would match the explanation as readily as the code — the same mistake that made
# an earlier plugin audit report live plugins as dead (docs/lessons.md,
# 2026-08-06). So the structural cases read the file with comment lines removed.
uncommented() { grep -v '^[[:space:]]*#' "$1"; }

@test "OPS-023: the add workflow classifies a rate limit apart from other failures" {
    # The whole design: a primary-pool exhaustion soft-fails for the reconciler
    # to heal, while any other error stays red. Collapsing them back into one
    # branch would restore the silent drop — or hide a dead token.
    run bash -c "uncommented() { grep -v '^[[:space:]]*#' \"\$1\"; }; uncommented '$ADD_WF' | grep -c 'API rate limit'"
    [ "$output" -ge 1 ]
    # A dedicated soft-fail exit code is what keeps the two apart at the call site.
    run bash -c "grep -c 'RATE_LIMITED' '$ADD_WF'"
    [ "$output" -ge 3 ]
    run bash -c "uncommented() { grep -v '^[[:space:]]*#' \"\$1\"; }; uncommented '$ADD_WF' | grep -c '::error::'"
    [ "$output" -ge 2 ]
}

@test "OPS-023: the add workflow never blanket-swallows failures" {
    # The YAML key, not the bare word: the header discusses `continue-on-error`
    # precisely to explain why it is not used.
    run grep -c 'continue-on-error:' "$ADD_WF"
    [ "$output" -eq 0 ]
}

@test "OPS-023: the add workflow takes the node id from the payload, not a lookup" {
    # Removing the lookup is what let both event types share one retrying call
    # path; a reintroduced query would also spend an extra point of the very
    # pool whose exhaustion this ticket is about.
    run grep -c 'node_id' "$ADD_WF"
    [ "$output" -ge 1 ]
    # `uses:`, not the bare name — the header cites the retired action by name.
    run grep -c 'uses: actions/add-to-project' "$ADD_WF"
    [ "$output" -eq 0 ]
}

@test "OPS-023: the reconciler validates its repo-name input before word-splitting" {
    # A workflow_dispatch input reaching a command line unquoted is the classic
    # workflow-injection shape. The charset guard must stay, and the raw value
    # must never be passed through unvalidated.
    run grep -c 'A-Za-z0-9._-' "$RECONCILE_WF"
    [ "$output" -ge 1 ]
    run grep -c 'backfill-only \$TARGET_REPOS' "$RECONCILE_WF"
    [ "$output" -eq 0 ]
}

@test "OPS-023: the reconciler is scheduled and goes loud when it cannot run" {
    # Healed drift is a notice; the backstop failing is an issue plus a red job,
    # mirroring pat-expiry.yml. A silent backstop is the fault this ticket names.
    run grep -c 'schedule:' "$RECONCILE_WF"
    [ "$output" -ge 1 ]
    run grep -c 'gh issue create' "$RECONCILE_WF"
    [ "$output" -ge 1 ]
    run grep -c '::error::' "$RECONCILE_WF"
    [ "$output" -ge 1 ]
}
