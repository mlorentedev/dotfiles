#!/bin/bash

# check-backlog-merged.sh: detect "merged-but-open" backlog drift in vault task files.
#
# Semantic sibling of check-backlog-integrity.sh (SDD-012). That guard catches
# STRUCTURAL drift (one id on 2+ lines, an id marked both [ ] and [x]). It cannot
# catch the SEMANTIC class: a `- [ ]` entry whose work is already shipped — a
# stale-merged tick is perfectly well-formed text. This guard closes that gap,
# implementing the cross-reference SDD-012 deferred as "merged-but-open".
#
# Done-signal (v1): an ARCHIVED SPEC for the entry's FULL id —
# `<repo>/specs/archive/<full-id>/`. The SDD lifecycle archives a spec at merge
# (see pattern-three-layer-proposal-lifecycle), so its presence is a reliable,
# local, network-free "this shipped" marker. Keyed on the FULL id (the bold
# token between ** **), NOT the bare ticket number — that is what dodges the
# number-reuse trap (dotfiles has had two BUG-024s, two HERMES-002s, etc.).
#
# Matching merged PRs by number is intentionally OUT OF SCOPE here: PR titles
# cite the number, not the slug, so the match is heuristic and number-reuse makes
# it noisy. That stays a manual/advisory step — see pattern-verify-against-
# source-of-truth.md "Automating the Reconciliation". This guard ships the
# reliable, zero-false-positive half.
#
# ADVISORY by contract: a finding is a "verify + tick" candidate, not proof.
# vault-health.sh treats a non-zero exit as a WARN, not a failure.
#
# Usage:
#   check-backlog-merged.sh <tasks-file> [--repo <repo-dir>]
#     <tasks-file>   a vault 11-tasks.md
#     --repo <dir>   repo holding specs/archive/. Default: inferred from the
#                    tasks path — <vault>/10_projects/<proj>/11-tasks.md maps to
#                    $HOME/Projects/<proj> (the workstation convention).
#
# Exit:
#   0  clean (no stale-open candidates, or repo not found -> nothing to check)
#   1  one or more open entries have an archived spec (likely done -> verify+tick)
#   2  usage error / tasks file missing

set -euo pipefail

TASKS=""
REPO=""

while [ "$#" -gt 0 ]; do
    case "$1" in
        --repo) REPO="${2:-}"; shift 2 ;;
        -h|--help)
            cat <<'EOF'
Usage: check-backlog-merged.sh <tasks-file> [--repo <repo-dir>]
  Flags open `- [ ]` entries whose FULL ticket id has an archived spec
  (<repo>/specs/archive/<id>/) — work shipped but the tick is stale.
  --repo defaults to $HOME/Projects/<proj> inferred from the tasks path.
  Advisory: findings are "verify + tick" candidates, not failures.
Exit: 0 clean | 1 stale-open candidates | 2 usage / missing file
EOF
            exit 0 ;;
        -*) printf '  ERROR: unknown option: %s\n' "$1" >&2; exit 2 ;;
        *)
            if [ -z "$TASKS" ]; then TASKS="$1"; shift
            else printf '  ERROR: only one tasks-file accepted\n' >&2; exit 2; fi ;;
    esac
done

if [ -z "$TASKS" ]; then
    cat >&2 <<'EOF'
Usage: check-backlog-merged.sh <tasks-file> [--repo <repo-dir>]
  Flags open `- [ ]` entries whose FULL ticket id has an archived spec
  (<repo>/specs/archive/<id>/) — i.e. the work shipped but the tick is stale.
  Advisory: findings are "verify + tick" candidates. See SDD-012b and
  pattern-verify-against-source-of-truth.md.
Exit: 0 clean | 1 stale-open candidates | 2 usage / missing file
EOF
    exit 2
fi

if [ ! -f "$TASKS" ]; then
    printf '  ERROR: not found or unreadable: %s\n' "$TASKS" >&2
    exit 2
fi

# Infer the repo from the tasks-file path when not given explicitly.
# <vault>/10_projects/<proj>/11-tasks.md -> $HOME/Projects/<proj>
if [ -z "$REPO" ]; then
    proj="$(basename "$(dirname "$TASKS")")"
    REPO="$HOME/Projects/$proj"
fi

# No repo -> no archived specs to cross-reference. Advisory skip, exit clean:
# the guard's job is to FLAG drift, not to fail when it cannot look.
if [ ! -d "$REPO/specs/archive" ]; then
    printf '  SKIP: no %s/specs/archive — nothing to cross-reference\n' "$REPO" >&2
    exit 0
fi

# Collect OPEN entries' FULL ids. Open = status char ' ', '~', or '-'
# (not 'x'/'X'). The full id is the bold token between the first ** ** pair,
# mirroring check-backlog-integrity.sh's identity rule.
open_ids="$(awk '
    /^- \[[ ~-]\] \*\*[^*]+\*\*/ {
        s = $0; sub(/^[^*]*\*\*/, "", s)
        id = s;  sub(/\*\*.*$/, "", id)
        print id
    }
' "$TASKS")"

candidates=0
while IFS= read -r id; do
    [ -z "$id" ] && continue
    if [ -d "$REPO/specs/archive/$id" ]; then
        printf '  STALE-OPEN: %s — archived spec exists (specs/archive/%s/); verify + tick [x]\n' "$id" "$id"
        candidates=$((candidates + 1))
    fi
done <<EOF
$open_ids
EOF

if [ "$candidates" -gt 0 ]; then
    cat >&2 <<EOF

$candidates open entr$([ "$candidates" -eq 1 ] && echo y || echo ies) in $TASKS have an archived spec — the work likely shipped but the
backlog tick is stale. Verify against git/PR history and mark [x] with the PR link.
Advisory only (archived spec is a strong but not absolute signal). See
pattern-verify-against-source-of-truth.md "Automating the Reconciliation".
EOF
    exit 1
fi
exit 0
