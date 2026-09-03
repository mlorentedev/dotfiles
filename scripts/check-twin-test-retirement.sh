#!/usr/bin/env bash

# check-twin-test-retirement.sh: enforce ADR-020 §5's second half.
#
# The ADR has said this since it was written:
#
#   "Port a script into `dot` only when it is next touched; IN THE SAME PR,
#    delete the .sh + .ps1 pair AND THEIR BATS/PESTER TESTS. Never leave the
#    three coexisting (no triple-maintenance)."
#
# Nothing enforced it. Both halves of that sentence were executed by hand in
# OPS-043 and CLI-072 because the agent happened to remember; a port that forgot
# would leave a bats or Pester file testing a script that no longer exists, and
# nothing would say so. That is the shape this repo keeps meeting: a rule that is
# written, agreed, and unenforced.
#
# The check is DIFF-based on purpose. A state-based version was measured and
# rejected: most of tests/*.bats covers invariants, docs and CI config rather
# than a same-stem script, so "a test whose script is missing" is ~90% false
# positives. The ADR says "in the same PR", so the PR's diff is the right unit.
#
# Logic lives here rather than inline in a workflow `run:` block for the reason
# recorded in tests/bitacora-reconcile.bats: inline logic is unreachable by tests
# and went red-and-silent twice (BUG-063).
#
# Usage:
#   check-twin-test-retirement.sh --base <ref>        # deletions from git
#   check-twin-test-retirement.sh --deleted-from FILE # one path per line (tests)
#
# Exit:
#   0  no retired script left its test twin behind
#   1  at least one did; the offenders are named
#   2  usage error

set -euo pipefail

usage() {
    printf '%s\n' "usage: check-twin-test-retirement.sh (--base <ref> | --deleted-from <file>)" >&2
    exit 2
}

BASE=""
DELETED_FROM=""
while [ $# -gt 0 ]; do
    case "$1" in
        --base) [ $# -ge 2 ] || usage; BASE="$2"; shift 2 ;;
        --deleted-from) [ $# -ge 2 ] || usage; DELETED_FROM="$2"; shift 2 ;;
        -h|--help) usage ;;
        *) printf 'unknown argument: %s\n' "$1" >&2; usage ;;
    esac
done

if [ -n "$BASE" ] && [ -n "$DELETED_FROM" ]; then usage; fi
if [ -z "$BASE" ] && [ -z "$DELETED_FROM" ]; then usage; fi

deleted_paths() {
    if [ -n "$DELETED_FROM" ]; then
        [ -f "$DELETED_FROM" ] || { printf 'no such file: %s\n' "$DELETED_FROM" >&2; exit 2; }
        cat "$DELETED_FROM"
        return 0
    fi
    # Three-dot form: what THIS branch deleted relative to the merge-base, not
    # what the base has since changed. Fails closed -- an unreachable merge-base
    # is an error, never an empty deletion list, because an empty list is
    # indistinguishable from "nothing was retired".
    git diff --diff-filter=D --name-only "$BASE...HEAD"
}

# test_twins_for <scripts/NAME.(sh|ps1)>: the test files that would cover it.
# Both spellings of the Pester name are checked: the repo carries
# install-dotf-ps1.Tests.ps1 as well as windows-defaults.Tests.ps1.
test_twins_for() {
    stem="$(basename "$1")"
    stem="${stem%.sh}"
    stem="${stem%.ps1}"
    printf '%s\n' "tests/$stem.bats" "tests/$stem.Tests.ps1" "tests/$stem-ps1.Tests.ps1"
}

# Resolve the deletion list ONCE, into a variable, checking the status.
#
# It used to be read as `$(deleted_paths)` inside the loop's heredoc, and that
# was a false-green generator: the command substitution runs in a subshell, so
# `exit 2` on a missing file — or a failing `git diff` on an unreachable
# merge-base — terminated the subshell and left the outer script iterating an
# EMPTY list, which exits 0. The guard's own header claims it fails closed; it
# did not, and its own test caught it.
if ! DELETED="$(deleted_paths)"; then
    printf '%s\n' "[FAIL] could not determine which files were deleted — refusing to report a clean tree" >&2
    exit 2
fi

offenders=""
while IFS= read -r path; do
    [ -n "$path" ] || continue
    case "$path" in
        scripts/*.sh|scripts/*.ps1) ;;
        *) continue ;;
    esac

    # A twin pair is only retired when BOTH halves are gone. Deleting just the
    # .ps1 (a Windows-only cleanup) must not demand the bats file's head.
    stem="$(basename "$path")"; stem="${stem%.sh}"; stem="${stem%.ps1}"
    if [ -f "scripts/$stem.sh" ] || [ -f "scripts/$stem.ps1" ]; then
        continue
    fi

    while IFS= read -r twin; do
        # Present at HEAD means it was NOT deleted in this change.
        if [ -f "$twin" ]; then
            offenders="$offenders$twin (its subject scripts/$stem.* was retired)
"
        fi
    done <<EOF
$(test_twins_for "$path")
EOF
done <<EOF
$DELETED
EOF

if [ -n "$offenders" ]; then
    printf '%s\n' "[FAIL] ADR-020 §5: a retired script left its test twin behind." >&2
    # Deduplicated: a retired PAIR deletes two scripts, and both resolve to the
    # same surviving test file, so the offender would otherwise be named twice.
    printf '%s' "$offenders" | sort -u | sed 's/^/  /' >&2
    printf '%s\n' "" >&2
    printf '%s\n' "Port the cases into the Go suite and delete the file in this PR." >&2
    printf '%s\n' "Leaving all three (shell, PowerShell, Go) is the triple-maintenance" >&2
    printf '%s\n' "the ADR forbids, and the stale suite is the one nobody runs." >&2
    exit 1
fi

printf '%s\n' "[OK] no retired script left a bats or Pester twin behind"
