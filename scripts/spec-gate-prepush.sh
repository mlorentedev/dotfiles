#!/usr/bin/env bash

# spec-gate-prepush.sh: local pre-push adapter that resolves LIVE PR metadata,
# when there is a PR to resolve, before running check-spec-gate.sh.
#
# BUG-061: check-spec-gate.sh's archive-on-merge credit
# (_mandated_archive_ids) only fires when SDD_PR_BODY names a closing keyword.
# Locally that variable is empty by design ("check-spec-gate.sh stays offline"
# — see its ADJACENCY_ISSUES comment), so a PR that correctly archives its spec
# in the same change could not be pushed: the LOC gate saw no active-spec touch
# and rejected a change the author had done exactly right. The documented
# escape (`SDD_PR_BODY=... git push`) is not one anybody would guess, so the
# discoverable "fix" was --no-verify.
#
# This mirrors scripts/spec-gate-pr.sh's shape but not its failure mode: that
# adapter fails CLOSED on a bad live read because in CI the alternative is a
# stale event-payload replay (BUG-066) — deciding on wrong data is worse than
# refusing to decide. Here there is no stale fallback to guard against: "no PR
# context" is simply the ordinary state before a PR exists (first push of a
# branch), so any failure to resolve one — no `gh`, no `jq`, unauthenticated,
# no PR open yet for this branch — falls THROUGH to running check-spec-gate.sh
# with nothing set, i.e. today's exact behaviour. Once a PR exists for the
# branch, local pre-push becomes as strict as CI: archive-on-merge and the
# label-gated skips now apply locally too, not only after push.
#
# No token required: `gh` uses the developer's own local auth, unlike
# spec-gate-pr.sh's CI run, which is the one that needs GH_TOKEN.
#
# Usage:
#   spec-gate-prepush.sh [args forwarded verbatim to check-spec-gate.sh]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

usage() {
    cat <<'EOF'
Usage: spec-gate-prepush.sh [args forwarded to check-spec-gate.sh]

Resolves the current branch's PR labels/body/author live via `gh`, when one
exists, then forwards every argument to check-spec-gate.sh. Falls through to
running the gate with no PR context (today's local behaviour) when `gh`/`jq`
are unavailable, the developer is unauthenticated, or no PR is open yet for
this branch — never fails on that account. e.g.
  spec-gate-prepush.sh --base-ref origin/main --head-ref HEAD
EOF
}

case "${1:-}" in
    -h|--help) usage; exit 0 ;;
esac

_run_gate() {
    exec "$SCRIPT_DIR/check-spec-gate.sh" "$@"
}

command -v gh >/dev/null 2>&1 || _run_gate "$@"
command -v jq >/dev/null 2>&1 || _run_gate "$@"

if ! meta=$(gh pr view --json labels,body,author 2>/dev/null); then
    # No PR for this branch yet, or the developer is unauthenticated. Both are
    # the ordinary local baseline, not an error this wrapper reports on.
    _run_gate "$@"
fi

# `// ""` on both scalars: gh returns JSON null for an empty body, and the
# literal string "null" reaching the gate would match headings and label
# substrings that were never there.
SDD_LABELS=$(printf '%s' "$meta" | jq -r '[.labels[].name] | join(",")')
SDD_PR_BODY=$(printf '%s' "$meta" | jq -r '.body // ""')
SDD_PR_AUTHOR=$(printf '%s' "$meta" | jq -r '.author.login // ""')
export SDD_LABELS SDD_PR_BODY SDD_PR_AUTHOR

_run_gate "$@"
