#!/usr/bin/env bats
# Golden characterization corpus for knowledge-crystallize.sh (CLI-021, #672).
#
# The shell is the ORACLE (revisions pinned in tests/golden/crystallize/ORACLE).
# These lock its observable behaviour — stdout, exit status and the resulting
# MEMORY.md, byte for byte — so the Go port (#490) can be proved equivalent
# rather than asserted to be.
#
# This is a different instrument from the existing suites and does not replace
# them: tests/knowledge-crystallize.bats is mostly REPRESENTATION (has function X,
# uses set -euo pipefail) which a correct port would fail on principle, and
# tests/knowledge-crystallize-yaml-guard.bats asserts invariants. Only a golden
# says "these exact bytes".
#
# Two behaviours captured here are OracLE DEFECTS, reproduced faithfully and
# ticketed separately per the CLI-021 proposal — a port that "improves while
# translating" cannot be characterization-tested. See the cases:
#   marker-without-dateline  logs "Updated currentDate" having written no date
#   handoff-with-duplicates  dedup relocates the currentDate block

setup() {
    HERE="$BATS_TEST_DIRNAME/golden/crystallize"
    export GC_ORACLE_SH="$BATS_TEST_DIRNAME/../scripts/knowledge-crystallize.sh"
    # shellcheck source=golden/crystallize/lib.sh disable=SC1091
    . "$HERE/lib.sh"
    ACTUAL="$(mktemp -d)"
}

teardown() {
    rm -rf "$ACTUAL"
}

# Re-run a case through the oracle and require byte-equality with the golden.
assert_golden() {
    local name="$1" case_dir="$HERE/cases/$1"
    [ -d "$case_dir/expected" ] || {
        printf 'no golden captured for %s — run tests/golden/crystallize/capture.sh\n' "$name" >&2
        return 1
    }
    gc_run_case "$case_dir" "$ACTUAL/$name"
    local artefact
    for artefact in exit stdout memory.md; do
        if ! diff -u "$case_dir/expected/$artefact" "$ACTUAL/$name/$artefact"; then
            printf 'golden mismatch: %s/%s\n' "$name" "$artefact" >&2
            return 1
        fi
    done
}

@test "golden: fresh file with no markers gains both sections" {
    assert_golden fresh-no-markers
}

@test "golden: currentDate present, Last Crystallized absent" {
    assert_golden currentdate-only
}

@test "golden: Last Crystallized present, currentDate absent" {
    assert_golden lastcrystallized-only
}

@test "golden: both markers already present are updated in place" {
    assert_golden both-markers
}

@test "golden: duplicate currentDate blocks collapse to one" {
    assert_golden duplicate-currentdate
}

@test "golden: HARNESS-029 - the handoff block stays last" {
    assert_golden handoff-no-markers
}

@test "golden: dedup relocates the currentDate block ahead of the handoff" {
    assert_golden handoff-with-duplicates
}

# BUG-060's second seed: a second pass must not duplicate a section nor move the
# handoff block. The golden is byte-identical to the single-run case by design.
@test "golden: a second run changes nothing" {
    assert_golden idempotent-twice
}

@test "golden: a currentDate marker with no date line (oracle no-op, ticketed)" {
    assert_golden marker-without-dateline
}

@test "golden: over the 150-line limit warns but still stamps" {
    assert_golden over-line-limit
}

@test "golden: BUG-062 - a YAML-wrapped file is refused, exit 1" {
    assert_golden yaml-wrapped
}

@test "golden: no MEMORY.md for the project exits 0 with guidance" {
    assert_golden no-memory-file
}

@test "golden: --help" {
    assert_golden help
}

@test "golden: --all counts a refusal as skipped, not processed" {
    assert_golden all-mixed
}

# --- corpus hygiene ---------------------------------------------------------

@test "every case directory has a golden and a test that exercises it" {
    local missing=""
    local d name
    for d in "$HERE"/cases/*/; do
        name="$(basename "$d")"
        [ -d "$d/expected" ] || missing="$missing golden:$name"
        grep -q "assert_golden $name\$" "$BATS_TEST_DIRNAME/knowledge-crystallize-golden.bats" \
            || missing="$missing test:$name"
    done
    [ -z "$missing" ] || {
        printf 'corpus drift —%s\n' "$missing" >&2
        return 1
    }
}

@test "the oracle revisions are recorded and still match the working tree" {
    [ -f "$HERE/ORACLE" ]
    local f recorded current
    while read -r recorded f; do
        case "$recorded" in '#'*|'') continue ;; esac
        current="$(git -C "$BATS_TEST_DIRNAME/.." log -1 --format=%H -- "$f")"
        if [ "$recorded" != "$current" ]; then
            printf 'oracle moved for %s: corpus captured at %s, tree has %s — recapture deliberately\n' \
                "$f" "$recorded" "$current" >&2
            return 1
        fi
    done < "$HERE/ORACLE"
}
