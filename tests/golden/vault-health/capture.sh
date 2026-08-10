#!/usr/bin/env bash
# Regenerate the vault-health golden corpus from the SHELL oracle.
#
# Run this ONLY as a deliberate act, with the reason in the commit message. A
# golden that goes red means the port diverged from the oracle; regenerating to
# turn it green destroys the only thing the corpus is for.
#
# Usage: tests/golden/vault-health/capture.sh [case ...]   (default: every case)

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
REPO_ROOT="$(cd "$HERE/../../.." && pwd)"

GVH_ORACLE_SH="$REPO_ROOT/scripts/vault-health.sh"
export GVH_ORACLE_SH

# The oracle is the SHELL, always — capturing from the port would make the
# corpus agree with whatever the port happens to do.
export GVH_IMPL_MODE=shell

# shellcheck source=lib.sh disable=SC1091
. "$HERE/lib.sh"

cases=("$@")
if [ "${#cases[@]}" -eq 0 ]; then
    cases=()
    for d in "$HERE"/cases/*/; do cases+=("$(basename "$d")"); done
fi

for name in "${cases[@]}"; do
    case_dir="$HERE/cases/$name"
    [ -d "$case_dir" ] || { printf 'no such case: %s\n' "$name" >&2; exit 1; }
    rm -rf "${case_dir:?}/expected"
    gvh_run_case "$case_dir" "$case_dir/expected"
    printf 'captured %s (exit %s)\n' "$name" "$(cat "$case_dir/expected/exit")"
done

# ── ORACLE ───────────────────────────────────────────────────────────────────
# Pinned by CONTENT HASH, not by commit SHA. A SHA is unavailable under CI's
# shallow checkout and changes on every rebase without a byte changing; a
# content hash answers the only question that matters — is the script these
# goldens were captured from still the script in the tree? (docs/lessons.md,
# 2026-08-09.)
{
    printf '# Oracle for the vault-health golden corpus (CLI-021 increment 2).\n'
    printf '#\n'
    printf '# sha256 of each file the goldens were captured from. A test fails the\n'
    printf '# suite when the working tree drifts from these, so a recapture is always\n'
    printf '# a deliberate, reviewable act.\n'
    printf '#\n'
    printf '# Captured at git revision %s (informational only - never assert on it,\n' "$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)"
    printf '# CI checks out shallow and rebases change it for free).\n'
    printf '\n'
    for f in scripts/vault-health.sh scripts/check-backlog-integrity.sh scripts/check-backlog-merged.sh; do
        printf '%s  %s\n' "$(sha256sum "$REPO_ROOT/$f" | cut -d' ' -f1)" "$f"
    done
} > "$HERE/ORACLE"

printf 'ORACLE written\n'
