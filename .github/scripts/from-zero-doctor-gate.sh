#!/usr/bin/env bash
# from-zero-doctor-gate.sh -- the doctor gate of the from-zero job (#2013 X1).
#
# Usage: from-zero-doctor-gate.sh <os> <doctor-exit-status> <doctor-output> [known-failures]
#
# On a fresh runner `dotf doctor --scope machine` still FAILs on findings the
# epic owns, each with its ledger row. They are listed in
# from-zero-known-failures.txt, so the job can be green while they are open
# and still catch a regression. The gate fails on a [FAIL] line no entry for
# <os> matches (new) and on an entry for <os> that matches no [FAIL] line
# (stale: its fix landed, so it leaves the list, which only shrinks). A doctor
# that exits non-zero without one [FAIL] line fails too: that is a crash, not a
# finding. Same contract as test-windows' doctor-gate.ps1 (TEST-003), with
# fixed strings instead of regexes, so GNU and BSD tools read them alike.
set -eu

if [ "$#" -lt 3 ] || [ "$#" -gt 4 ]; then
    echo "usage: $0 <os> <doctor-exit-status> <doctor-output> [known-failures]" >&2
    exit 2
fi
os=$1
rc=$2
out=$3
known=${4:-$(dirname "$0")/from-zero-known-failures.txt}

# Preflight: this gate reports absence (no new failure, no stale entry), so a
# missing tool must stop it rather than read as a clean result.
for tool in awk grep sed; do
    command -v "$tool" >/dev/null 2>&1 || { echo "::error::from-zero doctor gate: $tool not found" >&2; exit 2; }
done
for f in "$out" "$known"; do
    [ -r "$f" ] || { echo "::error::from-zero doctor gate: cannot read $f" >&2; exit 2; }
done

# A malformed entry is the list's defect, not the machine's: name it and stop.
malformed=$(awk '!/^[[:space:]]*(#|$)/ && !/^(darwin|linux) [^[:space:]]/ { print FILENAME ":" FNR ": " $0 }' "$known")
if [ -n "$malformed" ]; then
    printf '::error::from-zero doctor gate: an entry is "<darwin|linux> <text>":\n%s\n' "$malformed" >&2
    exit 2
fi

esc=$(printf '\033')
failures=$(sed "s/${esc}\[[0-9;]*m//g" "$out" | grep -F '[FAIL]' || true)
patterns=$(awk -v os="$os" '$1 == os { sub(/^[^ ]+ /, ""); print }' "$known")

bad=0
if [ "$rc" != 0 ] && [ -z "$failures" ]; then
    echo "::error::dotf doctor exited $rc without a [FAIL] line; see its output above"
    bad=1
fi

if [ -n "$failures" ]; then
    while IFS= read -r line; do
        hit=0
        if [ -n "$patterns" ]; then
            while IFS= read -r pat; do
                case "$line" in *"$pat"*) hit=1 ;; esac
            done <<EOF
$patterns
EOF
        fi
        if [ "$hit" = 0 ]; then
            echo "::error::new doctor failure on $os, not in $known (fix it, or file it and list it with its owner): $line"
            bad=1
        fi
    done <<EOF
$failures
EOF
fi

if [ -n "$patterns" ]; then
    while IFS= read -r pat; do
        case "$failures" in
            *"$pat"*) ;;
            *)
                echo "::error::stale entry for $os in $known, no [FAIL] line matches it any more; remove it: $pat"
                bad=1
                ;;
        esac
    done <<EOF
$patterns
EOF
fi

[ "$bad" = 0 ] || exit 1
n=0
[ -z "$failures" ] || n=$(printf '%s\n' "$failures" | awk 'END { print NR }')
echo "from-zero doctor gate ($os): $n known FAIL line(s), 0 new, 0 stale"
