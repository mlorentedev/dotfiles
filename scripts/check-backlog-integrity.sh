#!/bin/bash

# check-backlog-integrity.sh: detect backlog drift in vault task files.
#
# A hand-maintained `11-tasks.md` accumulates drift when the same ticket lives in
# two places (e.g. a sprint overview + a detailed list) that fall out of sync:
# the count inflates and ticks go stale (an ID shows `[ ]` in one view, `[x]` in
# another, or is `[ ]` everywhere despite being merged). This guard enforces the
# structural invariant that prevents it: ONE ticket = ONE entry.
#
# Per file it flags:
#   - DUPLICATE      a ticket ID on 2+ entry lines
#   - CONTRADICTION  a ticket ID marked BOTH open ([ ]/[~]/[-]) and done ([x])
#
# Ticket IDs are `[A-Z]+-[0-9]+` with an optional single-letter sub-id suffix
# (WIN-002 vs WIN-002a are distinct, not duplicates). No Obsidian dependency —
# pure text parsing, safe in CI and fixture tests. Sibling of check-md-escapes.sh
# (SDD-006); see specs/SDD-012-tasks-integrity-guard/.
#
# Usage:
#   check-backlog-integrity.sh <tasks-file>...
#
# Exit:
#   0  clean
#   1  one or more files have duplicate IDs / status contradictions
#   2  usage error / missing file

set -euo pipefail

if [ "$#" -eq 0 ]; then
    cat >&2 <<'EOF'
Usage: check-backlog-integrity.sh <tasks-file>...
  Flags, per file: duplicate ticket IDs (same ID on 2+ entry lines) and status
  contradictions (same ID marked both [ ] and [x]). One ticket = one entry.
  See specs/SDD-012-tasks-integrity-guard/ (sibling of SDD-006 check-md-escapes).
Exit: 0 clean | 1 issues found | 2 usage / missing file
EOF
    exit 2
fi

issues=0
for f in "$@"; do
    if [ ! -f "$f" ]; then
        printf '  ERROR: not found or unreadable: %s\n' "$f" >&2
        exit 2
    fi

    # Extract "<status-char>\t<ticket-id>" per checkbox entry line. The greedy
    # [a-z]? keeps WIN-002a whole while leaving WIN-002 intact before a "-slug".
    out="$(sed -nE 's/^- \[([ xX~-])\] \*\*([A-Z]+-[0-9]+[a-z]?).*/\1	\2/p' "$f" \
        | awk -F'\t' '
            {
                id=$2; st=$1
                count[id]++
                if (st=="x" || st=="X") done_[id]=1; else open_[id]=1
                if (!(id in seen)) { seen[id]=1; order[++n]=id }
            }
            END {
                for (i=1; i<=n; i++) {
                    id=order[i]
                    if (count[id] >= 2) {
                        if (done_[id] && open_[id])
                            printf "  CONTRADICTION: %s — %d entries, marked BOTH open and done\n", id, count[id]
                        else
                            printf "  DUPLICATE: %s — %d entries\n", id, count[id]
                    }
                }
            }')"

    if [ -n "$out" ]; then
        printf '%s:\n%s\n' "$f" "$out"
        issues=$((issues + 1))
    fi
done

if [ "$issues" -gt 0 ]; then
    cat >&2 <<EOF

$issues file(s) have backlog-integrity issues (duplicate IDs / status contradictions).
One ticket = one entry — consolidate the file so each ID appears once with a single
status. See AGENTS.md "incident -> guard" and specs/SDD-012-tasks-integrity-guard/.
EOF
    exit 1
fi
exit 0
