#!/usr/bin/env bats
# CI-012 (#2059): tests/lib/dotf-bin.bash decides which `dotf` a bats file runs,
# and what happens when there is none. Three files did `go build ... || skip`,
# so a compile error in cli/ read as skipped tests and a green `test` job. These
# cases pin the rule by what the helper DOES (run it, read its exit status), and
# the guard at the bottom keeps the fail-open pattern from coming back.

load 'lib/refute'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    BASH_BIN="$(command -v bash)"
    # A PATH that holds only what the helper itself needs. `go` is added per
    # test, so "no toolchain" is real rather than depending on the host.
    STUBS="$BATS_TEST_TMPDIR/bin"
    mkdir -p "$STUBS"
    ln -s "$(command -v mkdir)" "$STUBS/mkdir"
    ln -s "$(command -v dirname)" "$STUBS/dirname"
    ln -s "$(command -v chmod)" "$STUBS/chmod"
    FALLBACK="$BATS_TEST_TMPDIR/cache/dotf"
    CALLS="$BATS_TEST_TMPDIR/go-calls"
}

# Run dotf_bin_resolve in a clean shell. `skip` is replaced by a marker and a
# distinct status, because the real one only means something inside bats.
_resolve() {
    # Hermetic: the caller's DOTF_BIN and CI (the job exports both) must not leak
    # in, so each case names the ones it wants through CASE_DOTF_BIN / CASE_CI.
    run env -i PATH="$STUBS" BATS_TEST_DIRNAME="$REPO/tests" ${CASE_DOTF_BIN:+DOTF_BIN="$CASE_DOTF_BIN"} \
        ${CASE_CI:+CI="$CASE_CI"} "$BASH_BIN" -c '
        skip() { printf "SKIPPED: %s\n" "$1"; exit 77; }
        . "$BATS_TEST_DIRNAME/lib/dotf-bin.bash"
        dotf_bin_resolve "$1" || exit 1
        printf "RESOLVED: %s\n" "$DOTF_BIN"
    ' _ "$FALLBACK"
}

# A fake toolchain: `go build -o <path> ...` records the call and writes an
# executable to <path>.
_fake_go_ok() {
    cat > "$STUBS/go" <<EOF2
#!/bin/sh
echo call >> "$CALLS"
out=
while [ \$# -gt 0 ]; do [ "\$1" = "-o" ] && out="\$2"; shift; done
printf '#!/bin/sh\n' > "\$out"; chmod +x "\$out"
EOF2
    chmod +x "$STUBS/go"
}

# A toolchain that is present and cannot build: the compile-error case.
_fake_go_fail() {
    cat > "$STUBS/go" <<EOF2
#!/bin/sh
echo call >> "$CALLS"
exit 1
EOF2
    chmod +x "$STUBS/go"
}

@test "dotf-bin: DOTF_BIN set to an executable is used, with no toolchain needed" {
    printf '#!/bin/sh\n' > "$BATS_TEST_TMPDIR/prebuilt"; chmod +x "$BATS_TEST_TMPDIR/prebuilt"
    CASE_DOTF_BIN="$BATS_TEST_TMPDIR/prebuilt" _resolve
    [ "$status" -eq 0 ]
    [ "$output" = "RESOLVED: $BATS_TEST_TMPDIR/prebuilt" ]
}

@test "dotf-bin: DOTF_BIN set but missing fails, and never falls through to a build" {
    _fake_go_ok
    CASE_DOTF_BIN="$BATS_TEST_TMPDIR/nope" _resolve
    [ "$status" -eq 1 ]
    [[ "$output" == *"DOTF_BIN is set to $BATS_TEST_TMPDIR/nope"* ]] || false
    [ ! -e "$CALLS" ]
}

@test "dotf-bin: DOTF_BIN naming a directory is not an executable and fails" {
    CASE_DOTF_BIN="$BATS_TEST_TMPDIR" _resolve
    [ "$status" -eq 1 ]
}

@test "dotf-bin: no toolchain on a developer machine skips" {
    _resolve
    [ "$status" -eq 77 ]
    [[ "$output" == *"SKIPPED: go toolchain not installed"* ]] || false
}

@test "dotf-bin: no toolchain in CI fails rather than skipping" {
    CASE_CI=true _resolve
    [ "$status" -eq 1 ]
    [[ "$output" != *SKIPPED* ]] || false
    [[ "$output" == *"missing in CI"* ]] || false
}

@test "dotf-bin: a build that fails fails, locally as well as in CI" {
    _fake_go_fail
    _resolve
    [ "$status" -eq 1 ]
    [[ "$output" != *SKIPPED* ]] || false
    [[ "$output" == *"go build ./cmd/dotf failed"* ]] || false
}

@test "dotf-bin: with no DOTF_BIN it builds once into the fallback and reuses it" {
    _fake_go_ok
    _resolve
    [ "$status" -eq 0 ]
    [ "$output" = "RESOLVED: $FALLBACK" ]
    _resolve
    [ "$status" -eq 0 ]
    [ "$(wc -l < "$CALLS" | tr -d ' ')" -eq 1 ]
}

# --- the fail-open pattern, and the files that must not have it ---

# Prints "file:line" for every `go build` that is followed within three lines by
# a `skip`: the shape of "build failed, so skip". It sees only that layout: a
# build moved into a wrapper, with the `|| skip` at the call site, is out of its
# reach, which is what _builds_outside_the_helper below is for.
_fail_open_builds() {
    awk '
        FNR == 1 { last = -99 }
        /^[[:space:]]*#/ { next }
        /go build/ && !/grep/ { last = FNR }
        /skip[[:space:]]+"/ && FNR - last <= 3 && last > 0 { print FILENAME ":" FNR; last = -99 }
    ' "$@"
}

@test "guard: the detector flags a build that skips on failure (negative control)" {
    cat > "$BATS_TEST_TMPDIR/bad.bats" <<'EOF2'
setup() {
    ( cd cli && go build -o "$BIN" ./cmd/dotf ) \
        || skip "go build failed"
}
EOF2
    run _fail_open_builds "$BATS_TEST_TMPDIR/bad.bats"
    [ -n "$output" ]
}

# Prints "file:line" for every line that RUNS `go build`, anywhere under tests/
# but the helper and this file. A build that lives only in the helper cannot be
# wrapped by a caller that skips on its failure, whatever the layout: the caller
# has nothing to wrap. Lines that merely assert about someone else's build (a
# grep over a Dockerfile or a workflow) and comments are not builds.
_builds_outside_the_helper() {
    awk '
        /^[[:space:]]*#/ { next }
        /go build/ && !/grep/ { print FILENAME ":" FNR }
    ' "$@"
}

@test "guard: no bats file skips right after a go build" {
    # Not this file: its negative control above spells the pattern on purpose.
    local files=() f
    for f in "$REPO"/tests/*.bats; do
        [ "${f##*/}" = "${BATS_TEST_FILENAME##*/}" ] || files+=("$f")
    done
    run _fail_open_builds "${files[@]}"
    [ -z "$output" ]
}

@test "guard: the detector flags a build hidden in a wrapper (negative control)" {
    cat > "$BATS_TEST_TMPDIR/wrapped.bats" <<'EOF2'
build_dotf() { ( cd cli && go build -o "$BIN" ./cmd/dotf ); }

# ... other setup, four or more lines away ...
BIN=x
OTHER=y
setup() {
    build_dotf || skip "build failed"
}
EOF2
    run _builds_outside_the_helper "$BATS_TEST_TMPDIR/wrapped.bats"
    [ -n "$output" ]
    # and the window detector is blind to exactly that layout, which is why the
    # two guards exist side by side
    run _fail_open_builds "$BATS_TEST_TMPDIR/wrapped.bats"
    [ -z "$output" ]
}

@test "guard: no test file builds dotf outside the helper, so no wrapper can skip on its failure" {
    local files=() f
    for f in "$REPO"/tests/*.bats "$REPO"/tests/lib/*.bash "$REPO"/tests/golden/*/lib.sh; do
        [ -f "$f" ] || continue
        case "${f##*/}" in
            dotf-bin-helper.bats | dotf-bin.bash) continue ;;
        esac
        files+=("$f")
    done
    run _builds_outside_the_helper "${files[@]}"
    [ -z "$output" ]
}

@test "guard: every file that ran its own build now resolves the binary through the helper" {
    local f
    for f in compile-harness-real \
             vault-maintenance-weekly dotf-agent-run; do
        grep -qF "load 'lib/dotf-bin'" "$REPO/tests/$f.bats"
        grep -qF 'dotf_bin_resolve' "$REPO/tests/$f.bats"
        refute_grep 'go build' "$REPO/tests/$f.bats"
    done
}

# The CI half: bats files only read DOTF_BIN, so the job has to build and export
# it before they run, and has to keep the timing report that justifies the row.
_test_job() {
    awk '/^  test:/{flag=1; next} /^  [a-z0-9_-]+:/{flag=0} flag' "$REPO/.github/workflows/ci.yml"
}

@test "ci: the Linux test job builds dotf and exports DOTF_BIN before the bats step" {
    job="$(_test_job)"
    build=$(printf '%s\n' "$job" | grep -n 'go build -o "$RUNNER_TEMP/dotf" ./cmd/dotf' | head -1 | cut -d: -f1)
    export_line=$(printf '%s\n' "$job" | grep -n 'DOTF_BIN=$RUNNER_TEMP/dotf" >> "$GITHUB_ENV"' | head -1 | cut -d: -f1)
    run_line=$(printf '%s\n' "$job" | grep -n 'run-bats.sh' | head -1 | cut -d: -f1)
    [ -n "$build" ] && [ -n "$export_line" ] && [ -n "$run_line" ]
    [ "$build" -lt "$run_line" ]
    [ "$export_line" -lt "$run_line" ]
}

@test "ci: the bats step keeps its parallelism flags and adds a junit timing report that is uploaded" {
    job="$(_test_job)"
    # The flags live in scripts/run-bats.sh, the one place bats runs on every OS
    # (PLAT-001d); the job passes the report directory through it.
    grep -qF -- '--no-parallelize-within-files' "$REPO/scripts/run-bats.sh"
    grep -qF -- '--report-formatter junit --timing --output' "$REPO/scripts/run-bats.sh"
    printf '%s\n' "$job" | grep -qF -- 'run-bats.sh --report-dir "$RUNNER_TEMP/bats-report"'
    printf '%s\n' "$job" | grep -qE 'uses: actions/upload-artifact@[0-9a-f]{40}'
    printf '%s\n' "$job" | grep -qF 'path: ${{ runner.temp }}/bats-report/report.xml'
    # A red run is the one whose timings are wanted; a plain `success()` upload
    # would drop them.
    printf '%s\n' "$job" | grep -qF '!cancelled()'
}
