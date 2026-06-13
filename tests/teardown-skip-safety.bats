#!/usr/bin/env bats
# Guard: a teardown()'s LAST statement classifies even *skipped* tests in bats.
#
# A bare `[ -n "$VAR" ] && rm -rf "$VAR"` as the final teardown statement exits
# non-zero when VAR is unset -- e.g. a setup() that called `skip` before it
# assigned VAR. bats then reclassifies a clean `ok # skip` into a confusing
# `not ok # skip` (looks like a failure; CI stayed green only because the skip
# condition was false there). See docs/lessons.md [2026-06-13].
#
# Safe idiom: `[ -z "${VAR:-}" ] || rm -rf "$VAR"` -- both branches exit 0.

setup() {
    TESTS_DIR="$BATS_TEST_DIRNAME"
}

# Emit "<file>: <last teardown statement>" for every *.bats whose teardown ends
# in the skip-fragile `[ -n ... ] && ...` form.
_offenders() {
    for f in "$TESTS_DIR"/*.bats; do
        awk '
            /^[[:space:]]*teardown\(\)[[:space:]]*\{/ { inb = 1; last = ""; next }
            inb && /^[[:space:]]*\}/                 { inb = 0; check = 1 }
            inb && NF && $0 !~ /^[[:space:]]*#/       { last = $0 }
            check {
                check = 0
                if (last ~ /\[[[:space:]]+-n[[:space:]].*\][[:space:]]*&&/) {
                    sub(/^[[:space:]]+/, "", last)
                    print FILENAME ": " last
                }
            }
        ' "$f"
    done
}

@test "no teardown ends with a skip-fragile [ -n ... ] && guard" {
    run _offenders
    [ "$status" -eq 0 ]
    if [ -n "$output" ]; then
        printf 'Skip-fragile teardown(s) found -- the last statement exits\n' >&2
        printf 'non-zero when the var is unset, turning `ok # skip` into\n' >&2
        printf '`not ok # skip`. Use `[ -z "${VAR:-}" ] || rm -rf "$VAR"`:\n\n' >&2
        printf '%s\n' "$output" >&2
        false
    fi
}
