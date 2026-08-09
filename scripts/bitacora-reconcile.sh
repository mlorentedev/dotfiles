#!/usr/bin/env bash

# bitacora-reconcile.sh: classify a --backfill-only reconciler run (OPS-023, #809).
#
# This is the error-handling half of .github/workflows/bitacora-reconcile.yml,
# extracted from the workflow's `run:` block by BUG-063.
#
# WHY IT IS A SCRIPT AND NOT AN INLINE BLOCK
#
# GitHub Actions invokes a `run:` block as `bash -e {0}` — the -e arrives on the
# SHELL INVOCATION, and `set -uo pipefail` inside the script does not clear it.
# The inline version captured the rollout output with a bare
#
#     out=$(./scripts/bitacora-rollout.sh --backfill-only ...)
#
# so the instant the rollout exited non-zero the step died ON THAT LINE and every
# branch below it — the rate-limit soft-pass AND the self-reporting `gh issue
# create` — was unreachable. Two scheduled runs (2026-08-08, 2026-08-09) went red
# and silent: no output, no issue filed, and `$out` destroyed before it was ever
# printed, taking the diagnosis with it.
#
# As a separate script the injected -e governs only the one-line invocation in the
# workflow, where exit 0/1 is exactly the green/red mapping we want. The
# classification lives here, where it is reachable AND executable by a test.
#
# Usage:  bitacora-reconcile.sh
#
# Env:
#   TARGET_REPOS       Space-separated repo names (empty = every eligible repo)
#   GH_TOKEN, GH_REPO  Passed through to gh by the workflow
#   BITACORA_ROLLOUT   Override the rollout script path (tests inject a stub)
#
# Exit:
#   0  reconciled, or hit the rate limit it exists to tolerate
#   1  the backstop itself is broken (an issue has been filed)
#   2  usage error (a refused repo name)

set -uo pipefail

_SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]:-$0}")" && pwd)
ROLLOUT="${BITACORA_ROLLOUT:-$_SCRIPT_DIR/bitacora-rollout.sh}"
ISSUE_TITLE="Bitácora reconciler is failing"

# TARGET_REPOS is a workflow_dispatch input, so it is attacker-shaped even though
# dispatching needs write access. It must be word-split to pass several repos, and
# an unquoted expansion is exactly how a workflow injection lands — so every token
# is validated against the GitHub repo-name charset first and anything else is
# refused outright.
#
# The unquoted expansion is deliberate (it must word-split) but it also invites
# PATHNAME expansion, so `set -f` first: without it TARGET_REPOS='*' expands to
# whatever the checkout root happens to contain and those filenames sail through
# the charset check as repo names. The refusal below only ever sees literals.
repos=()
set -f
for r in ${TARGET_REPOS:-}; do
    case "$r" in
        *[!A-Za-z0-9._-]*)
            printf '::error::refusing repo name %s — only [A-Za-z0-9._-] is accepted\n' "$r"
            exit 2
            ;;
    esac
    repos+=("$r")
done
set +f

# The capture that started it all. `|| rc=$?` makes the exit status explicit
# rather than depending on -e being off, so this line is correct under any flag
# state the caller happens to impose.
rc=0
out=$("$ROLLOUT" --backfill-only ${repos[@]+"${repos[@]}"} 2>&1) || rc=$?

# Print BEFORE classifying, unconditionally. The inline version only reached its
# print on the paths that never failed, which is how the evidence for the real
# failure was lost. The log is the only artifact a scheduled run leaves behind.
printf '%s\n' "$out"

if [ "$rc" -eq 0 ]; then
    printf '::notice::bitácora reconciled — every open item is on the board\n'
    exit 0
fi

# The healer hitting the very limit it heals is expected, not an incident:
# tomorrow's run picks it up. Green, but never silent.
case "$out" in
    *"API rate limit exceeded"*|*"API rate limit already exceeded"*)
        printf '::warning::reconciler hit the primary GraphQL pool — some items may still be off the board; the next daily run will retry\n'
        exit 0
        ;;
esac

# Anything else means the backstop itself is broken. File it where it will be seen
# rather than leaving it in a run log nobody opens.
gh label create bitacora-reconcile \
    --description "The bitácora reconciler failed (OPS-023)" \
    --color D93F0B --force >/dev/null 2>&1 || true

# NOT a bare assignment. Under the -e Actions injects, a failing `gh issue list`
# would abort the script HERE — before the ::error:: below, before anything is
# filed — which is precisely the defect this change exists to remove. It was
# reproduced once already in this very block; `|| lookup_rc=$?` is what keeps the
# reporting path independent of the caller's flags.
lookup_rc=0
existing=$(gh issue list --state open --label bitacora-reconcile \
    --json number,title \
    --jq "[.[] | select(.title==\"${ISSUE_TITLE}\") | .number] | first // empty" 2>/dev/null) || lookup_rc=$?

# Dedupe by stable title, so a daily cron comments instead of spamming.
if [ "$lookup_rc" -ne 0 ]; then
    # Filing blind would duplicate the stable-title issue the dedupe exists to
    # prevent, and a daily cron would keep duplicating it. A late report beats a
    # spammed board; the ::error:: below still makes the run red either way.
    printf '::warning::could not query existing bitacora-reconcile issues — not filing, to avoid a duplicate\n'
elif [ -n "${existing}" ]; then
    if gh issue comment "${existing}" \
        --body "Still failing on $(date -u +%F). Run: ${GITHUB_SERVER_URL:-}/${GITHUB_REPOSITORY:-}/actions/runs/${GITHUB_RUN_ID:-}"; then
        printf '::notice::updated existing issue #%s\n' "${existing}"
    else
        printf '::warning::could not comment on issue #%s\n' "${existing}"
    fi
elif gh issue create --title "${ISSUE_TITLE}" --label bitacora-reconcile --body \
"\`bitacora-reconcile.yml\` failed, so items dropped by the event-driven add are no longer being healed.

This is the backstop for OPS-023 (#809). While it is down, a board gap is silent again.

Run: ${GITHUB_SERVER_URL:-}/${GITHUB_REPOSITORY:-}/actions/runs/${GITHUB_RUN_ID:-}

Filed automatically by \`scripts/bitacora-reconcile.sh\`."; then
    printf '::warning::opened a bitacora-reconcile issue\n'
else
    printf '::warning::could not open a bitacora-reconcile issue\n'
fi

printf '::error::bitácora reconciliation failed for a non-rate-limit reason\n'
exit 1
