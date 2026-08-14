#!/usr/bin/env bash

# spec-gate-pr.sh: CI adapter that runs check-spec-gate.sh against LIVE PR metadata.
#
# check-spec-gate.sh reads its PR context from SDD_LABELS / SDD_PR_BODY /
# SDD_PR_AUTHOR. The workflow used to fill those from github.event.pull_request.*,
# which is the event payload — and a re-run REPLAYS the original payload. So the
# gate re-evaluated the PR as it was BEFORE the operator fixed it, and the gate's
# own documented escape ("skip-archive" label + a "## Archive skip rationale"
# section in the body) could not be applied to a live PR without pushing an
# unrelated commit (BUG-066, observed on #877).
#
# This adapter reads labels/body/author from the API at run time instead, which
# makes "Re-run failed jobs" the recovery it already looks like.
#
# It lives here rather than inline in the workflow for the reason recorded in
# tests/bitacora-reconcile.bats: logic inside a `run:` block is unreachable by
# tests and went red-and-silent twice (BUG-063). The token stays in the
# workflow: this adapter needs GH_TOKEN to read a PR it does not own (CI's
# actor). scripts/spec-gate-prepush.sh (BUG-061/#854) is the local pre-push
# equivalent and needs none, reading a PR under the developer's own `gh auth`
# instead — see its header for why its failure mode is deliberately the
# opposite of this one's fail-closed behaviour below.
#
# Usage:
#   spec-gate-pr.sh --pr N [args forwarded verbatim to check-spec-gate.sh]
#
# Env:
#   GH_TOKEN / GH_REPO  consumed by `gh` (set by the workflow)
#
# Exit:
#   0/1  whatever check-spec-gate.sh returned (OK / Discipline Gate violation)
#   2    usage error, or the live metadata read failed

# Explicit, not inherited: Actions injects -e via `bash -e {0}`, but bats does
# not. Relying on the injected flag is precisely the BUG-063 trap.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

usage() {
    cat <<'EOF'
Usage: spec-gate-pr.sh --pr N [args forwarded to check-spec-gate.sh]

  --pr N     Pull request number whose labels/body/author to read live
  -h, --help Show this help

Every other argument is forwarded verbatim to check-spec-gate.sh, e.g.
  spec-gate-pr.sh --pr 877 --base-ref origin/main --head-ref HEAD --explain
EOF
}

PR_NUMBER=""
FORWARD=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --pr) PR_NUMBER="${2:-}"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) FORWARD+=("$1"); shift ;;
    esac
done

if [[ -z "$PR_NUMBER" ]]; then
    printf '[ERROR] --pr is required\n' >&2
    usage >&2
    exit 2
fi

# Guarded rather than relying on "${FORWARD[@]}" expanding to nothing under
# set -u: an empty forward list is a wiring mistake worth naming here, not a
# usage error surfaced two scripts away.
if [[ ${#FORWARD[@]} -eq 0 ]]; then
    printf '[ERROR] nothing to forward to check-spec-gate.sh (need at least --base-ref/--head-ref)\n' >&2
    exit 2
fi

if ! meta=$(gh pr view "$PR_NUMBER" --json labels,body,author 2>&1); then
    printf '[ERROR] could not read live metadata for PR #%s:\n%s\n' "$PR_NUMBER" "$meta" >&2
    printf '[ERROR] Refusing to fall back to the event payload — a blocking gate must not\n' >&2
    printf '        decide on the stale source this adapter exists to replace (BUG-066).\n' >&2
    printf '        Re-run the job: the read is retried against current state.\n' >&2
    exit 2
fi

# `// ""` on both scalars: gh returns JSON null for an empty body, and the
# literal string "null" reaching the gate would match headings and label
# substrings that were never there.
SDD_LABELS=$(printf '%s' "$meta" | jq -r '[.labels[].name] | join(",")')
SDD_PR_BODY=$(printf '%s' "$meta" | jq -r '.body // ""')
SDD_PR_AUTHOR=$(printf '%s' "$meta" | jq -r '.author.login // ""')
export SDD_LABELS SDD_PR_BODY SDD_PR_AUTHOR

exec "$SCRIPT_DIR/check-spec-gate.sh" "${FORWARD[@]}"
