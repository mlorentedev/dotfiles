#!/usr/bin/env bats
# Byte-parity between `dotf vault crystallize` and the shell oracle (CLI-021).
#
# The claim increment 1 has to support is "the Go output is byte-identical to the
# shell for every fixture". That claim is only meaningful if ONE set of
# expectations judges both, so this reuses the very goldens captured from the
# shell in tests/golden/crystallize/ and the very same runner (lib.sh) — it does
# not re-derive anything in Go. A parity test that built its own expectations
# would pass while the two implementations drifted.
#
# Skips (never fails) when the Go toolchain is absent, so a shell-only checkout
# still runs the rest of the suite. CI installs Go for the `test` job precisely so
# this does not silently skip there — the failure mode #807 and BUG-055 are both
# about a check that skips instead of running.

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

# Build once per test file run, into a cached location, so 13 cases do not pay 13
# compilations.
_build_dotf() {
    command -v go >/dev/null 2>&1 || skip "go toolchain not installed"
    GC_DOTF_BIN="${BATS_FILE_TMPDIR:-/tmp}/dotf-parity"
    export GC_DOTF_BIN
    if [ ! -x "$GC_DOTF_BIN" ]; then
        ( cd "$BATS_TEST_DIRNAME/../cli" && go build -o "$GC_DOTF_BIN" ./cmd/dotf ) \
            || skip "go build failed"
    fi
    export GC_IMPL_MODE=go
}

assert_parity() {
    local name="$1" case_dir="$HERE/cases/$1"
    _build_dotf
    gc_run_case "$case_dir" "$ACTUAL/$name"
    local artefact
    for artefact in exit stdout memory.md; do
        if ! diff -u "$case_dir/expected/$artefact" "$ACTUAL/$name/$artefact"; then
            printf 'go/shell parity broken: %s/%s\n' "$name" "$artefact" >&2
            return 1
        fi
    done
}

@test "parity: fresh file with no markers gains both sections" {
    assert_parity fresh-no-markers
}

@test "parity: currentDate present, Last Crystallized absent" {
    assert_parity currentdate-only
}

@test "parity: Last Crystallized present, currentDate absent" {
    assert_parity lastcrystallized-only
}

@test "parity: both markers already present are updated in place" {
    assert_parity both-markers
}

@test "parity: duplicate currentDate blocks collapse to one" {
    assert_parity duplicate-currentdate
}

@test "parity: HARNESS-029 - the handoff block stays last" {
    assert_parity handoff-no-markers
}

@test "parity: dedup relocates the currentDate block ahead of the handoff" {
    assert_parity handoff-with-duplicates
}

@test "parity: a currentDate marker with no date line (oracle no-op, ticketed)" {
    assert_parity marker-without-dateline
}

@test "parity: over the 150-line limit warns but still stamps" {
    assert_parity over-line-limit
}

@test "parity: BUG-062 - a YAML-wrapped file is refused, exit 1" {
    assert_parity yaml-wrapped
}

@test "parity: no MEMORY.md for the project exits 0 with guidance" {
    assert_parity no-memory-file
}

@test "parity: a second run changes nothing" {
    assert_parity idempotent-twice
}

@test "parity: --all counts a refusal as skipped, not processed" {
    assert_parity all-mixed
}

# `help` is deliberately absent from the parity set. The shell hand-rolls its
# usage text; cobra generates its own. That difference is a property of the CLI
# framework, not a port defect, and pretending otherwise would either freeze
# cobra's help output as a contract or force the shell's format onto it.

@test "the parity set covers every case except the documented help exclusion" {
    local d name missing=""
    for d in "$HERE"/cases/*/; do
        name="$(basename "$d")"
        [ "$name" = "help" ] && continue
        grep -q "assert_parity $name\$" "$BATS_TEST_DIRNAME/knowledge-crystallize-go-parity.bats" \
            || missing="$missing $name"
    done
    [ -z "$missing" ] || {
        printf 'cases with a golden but no parity test —%s\n' "$missing" >&2
        return 1
    }
}
