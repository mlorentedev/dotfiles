#!/usr/bin/env bash
# Regenerate the crystallize golden corpus from the SHELL ORACLE (CLI-021 / #672).
#
# Run this ONLY to (re)capture against the shell, never to make a failing Go port
# go green — that inverts the whole point. The goldens are the executable contract
# the port must satisfy; if `dotf vault crystallize` disagrees with a golden, the
# port is wrong until proven otherwise, and a deliberate behaviour change gets a
# recapture in its own commit with the reason in the message.
#
# Usage:
#   tests/golden/crystallize/capture.sh [case ...]     (default: every case)

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
REPO_ROOT="$(cd "$HERE/../../.." && pwd)"

export GC_ORACLE_SH="$REPO_ROOT/scripts/knowledge-crystallize.sh"
# shellcheck source=lib.sh disable=SC1091
. "$HERE/lib.sh"

cases=("$@")
if [ "${#cases[@]}" -eq 0 ]; then
    for d in "$HERE"/cases/*/; do
        cases+=("$(basename "$d")")
    done
fi

for name in "${cases[@]}"; do
    case_dir="$HERE/cases/$name"
    [ -d "$case_dir" ] || { printf 'no such case: %s\n' "$name" >&2; exit 1; }
    rm -rf "${case_dir:?}/expected"
    gc_run_case "$case_dir" "$case_dir/expected"
    printf 'captured %s (exit %s)\n' "$name" "$(cat "$case_dir/expected/exit")"
done

# Pin the oracle per file rather than to a branch tip: the tip moves for unrelated
# reasons, and "which revision produced these bytes" is the only question the
# record has to answer.
{
    printf '# Oracle revisions for the crystallize golden corpus.\n'
    printf '# Regenerate with tests/golden/crystallize/capture.sh\n'
    printf '# Captured: %s\n\n' "$(date -u +%Y-%m-%d)"
    for f in scripts/knowledge-crystallize.sh scripts/knowledge-crystallize.ps1; do
        printf '%s  %s\n' "$(git -C "$REPO_ROOT" log -1 --format=%H -- "$f")" "$f"
    done
} > "$HERE/ORACLE"

printf '\noracle:\n'
cat "$HERE/ORACLE"
