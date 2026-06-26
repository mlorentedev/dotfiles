#!/bin/bash
#
# check-bats-names.sh: catch bats @test names the runner silently fails to
# register.
#
# bats translates each `@test "<name>"` into a shell function name. Names with
# non-ASCII characters (em-dash, <=, ...) or duplicated within a file cause the
# test to be skipped, or the whole file to fail parsing -- while the suite still
# exits 0. The result is a "green" test that never ran.
#
# Detected by the auto-curation analyzer (CURATOR-001, knowledge#135) after the
# class recurred: duplicate names (docs/lessons.md 2026-06-25) + an em-dash that
# silently skipped a test in #607. See dotfiles#615.
#
# Usage:
#   check-bats-names.sh <path>...   files, or directories scanned for *.bats
#
# Exit:
#   0  clean
#   1  one or more @test names would be silently dropped by bats
#   2  usage error

set -euo pipefail

if [ "$#" -eq 0 ]; then
    echo "Usage: check-bats-names.sh <path>...  (.bats files or directories)" >&2
    exit 2
fi

# Collect .bats files from the args: files as-is, directories scanned recursively.
files=()
for p in "$@"; do
    if [ -d "$p" ]; then
        while IFS= read -r f; do files+=("$f"); done < <(find "$p" -type f -name '*.bats' | sort)
    elif [ -f "$p" ]; then
        files+=("$p")
    fi
done

if [ "${#files[@]}" -eq 0 ]; then
    echo "check-bats-names: no .bats files found"
    exit 0
fi

rc=0
for f in "${files[@]}"; do
    # (a) non-ASCII byte anywhere inside a double-quoted @test name. LC_ALL=C makes
    #     the class match by byte, so any UTF-8 multibyte char (em-dash, <=) hits.
    while IFS= read -r ln; do
        [ -n "$ln" ] || continue
        printf '%s:%s: non-ASCII character in @test name (bats silently fails to register it)\n' "$f" "$ln"
        rc=1
    done < <(LC_ALL=C grep -nP '^[[:space:]]*@test[[:space:]]+"[^"]*[^\x00-\x7F]' "$f" 2>/dev/null | cut -d: -f1)

    # (b) duplicate @test names within the file (bats refuses to parse it).
    while IFS= read -r name; do
        [ -n "$name" ] || continue
        lines=$(grep -nF -- "@test \"${name}\"" "$f" | cut -d: -f1 | paste -sd, -)
        printf '%s: duplicate @test name "%s" (lines %s) -- bats refuses to parse the file\n' "$f" "$name" "$lines"
        rc=1
    done < <(grep -oE '^[[:space:]]*@test[[:space:]]+"[^"]*"' "$f" \
                | sed -E 's/^[[:space:]]*@test[[:space:]]+"(.*)"$/\1/' \
                | sort | uniq -d)
done

if [ "$rc" -eq 0 ]; then
    echo "check-bats-names: OK (${#files[@]} file(s) clean)"
fi
exit "$rc"
