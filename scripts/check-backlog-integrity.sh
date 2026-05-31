#!/bin/bash

# check-backlog-integrity.sh: detect backlog drift in vault task files.
#
# A hand-maintained `11-tasks.md` accumulates drift when the same ticket lives in
# two places (e.g. a sprint overview + a detailed list) that fall out of sync:
# the count inflates and ticks go stale (an id shows `[ ]` in one view, `[x]` in
# another). This guard enforces the structural invariant that prevents it:
# ONE ticket = ONE entry.
#
# Per file it flags:
#   - DUPLICATE      the same FULL ticket id on 2+ entry lines (true drift)
#   - CONTRADICTION  the same FULL id marked BOTH open ([ ]/[~]/[-]) and done ([x])
#   - NOTE           one ticket NUMBER reused by different full ids — advisory
#                    only, NOT a failure (two distinct tickets sharing a number)
#
# Identity is the FULL bold id between ** ** (so BUG-020-pwsh-profile is distinct
# from BUG-020-splitpath, and WIN-002 from WIN-002a). Matching by full id catches
# genuine "same ticket listed twice" drift without false-positiving two different
# tickets that merely share a number — the latter is reported as an advisory NOTE
# (renumbering archived tickets would desync them from their PR/commit history).
# No Obsidian dependency — pure text parsing, safe in CI and fixture tests.
# Sibling of check-md-escapes.sh (SDD-006); see SDD-012-tasks-integrity-guard.
#
# Usage:
#   check-backlog-integrity.sh <tasks-file>...
#
# Exit:
#   0  clean (or only advisory NOTEs)
#   1  one or more files have duplicate full ids / status contradictions
#   2  usage error / missing file

set -euo pipefail

if [ "$#" -eq 0 ]; then
    cat >&2 <<'EOF'
Usage: check-backlog-integrity.sh <tasks-file>...
  Flags, per file: duplicate FULL ticket ids (same id on 2+ entry lines) and
  status contradictions (same id marked both [ ] and [x]). One ticket = one
  entry. Number reuse across different tickets is an advisory NOTE, not a
  failure. See SDD-012-tasks-integrity-guard (sibling of SDD-006 check-md-escapes).
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

    # All parsing in awk (no sed): the full id is the bold token between ** **,
    # the status char is column 4 ("- [X] ..."). DUPLICATE/CONTRADICTION key off
    # the full id; the number prefix only drives the advisory NOTE.
    out="$(awk '
        /^- \[[ xX~-]\] \*\*[^*]+\*\*/ {
            line = $0
            st = substr(line, 4, 1)
            s = line; sub(/^[^*]*\*\*/, "", s)
            fid = s; sub(/\*\*.*$/, "", fid)
            count[fid]++
            if (st == "x" || st == "X") done_[fid] = 1; else open_[fid] = 1
            if (!(fid in seen)) { seen[fid] = 1; order[++n] = fid }
            num = fid
            if (match(num, /^[A-Z]+-[0-9]+/)) {
                num = substr(num, 1, RLENGTH)
                if (!((num SUBSEP fid) in spair)) {
                    spair[num SUBSEP fid] = 1
                    ncount[num]++
                    nex[num] = nex[num] (ncount[num] > 1 ? ", " : "") fid
                    if (!(num in snum)) { snum[num] = 1; norder[++m] = num }
                }
            }
        }
        END {
            for (i = 1; i <= n; i++) {
                fid = order[i]
                if (count[fid] >= 2) {
                    if (done_[fid] && open_[fid])
                        printf "  CONTRADICTION: %s — %d entries, marked BOTH open and done\n", fid, count[fid]
                    else
                        printf "  DUPLICATE: %s — %d entries\n", fid, count[fid]
                }
            }
            for (i = 1; i <= m; i++) {
                num = norder[i]
                if (ncount[num] > 1)
                    printf "  NOTE: number %s reused by %d different tickets (%s) — advisory, not drift\n", num, ncount[num], nex[num]
            }
        }
    ' "$f")" || true

    if [ -n "$out" ]; then
        printf '%s:\n%s\n' "$f" "$out"
    fi
    # Only DUPLICATE / CONTRADICTION are hard failures; NOTE (number reuse) is advisory.
    if printf '%s\n' "$out" | grep -qE 'DUPLICATE|CONTRADICTION'; then
        issues=$((issues + 1))
    fi
done

if [ "$issues" -gt 0 ]; then
    cat >&2 <<EOF

$issues file(s) have backlog-integrity issues (duplicate full ids / status contradictions).
One ticket = one entry — consolidate so each full id appears once with a single status.
(NOTE lines about number reuse are advisory, not failures.)
See AGENTS.md "incident -> guard" and specs/archive/SDD-012-tasks-integrity-guard/.
EOF
    exit 1
fi
exit 0
