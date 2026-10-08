#!/usr/bin/env bats
# scripts/run-bats.sh when its environment is broken. The runner is what turns a
# missing dependency into an annotated failure instead of an empty or a bats-level
# error, so each case here builds the broken environment and asserts the message.
# The working-environment cases drive the real tools in tests/run-bats-real.bats.

# bats file_tags=os-sensitive

load 'lib/refute'

setup() {
    RUN="$BATS_TEST_DIRNAME/../scripts/run-bats.sh"
    BIN="$BATS_TEST_TMPDIR/bin"
    mkdir -p "$BIN"
}

# Links the real tools the script itself needs into $BIN, so a PATH of only $BIN
# is a working shell environment with exactly one thing missing.
link_basics() {
    local t
    for t in bash dirname head grep getconf; do
        ln -s "$(command -v "$t")" "$BIN/$t"
    done
}

@test "run-bats: a missing GNU parallel fails with the CI annotation, not a bats error" {
    link_basics
    run env PATH="$BIN" "$BIN/bash" "$RUN" --print-jobs
    [ "$status" -eq 1 ]
    [[ "$output" == "::error::run-bats: GNU parallel is missing"* ]] || false
}

@test "run-bats: a parallel that is not GNU parallel (moreutils) is refused by name" {
    link_basics
    printf '#!/bin/sh\necho "parallel from moreutils"\n' > "$BIN/parallel"
    chmod +x "$BIN/parallel"
    run env PATH="$BIN" "$BIN/bash" "$RUN" --print-jobs
    [ "$status" -eq 1 ]
    [[ "$output" == *"is not GNU parallel"* ]] || false
}

@test "run-bats: an unknown option is a usage error" {
    run "$RUN" --no-such-flag
    [ "$status" -eq 2 ]
}

@test "run-bats: an option given an empty value is a usage error, never an unfiltered run" {
    # `--filter-tags "$UNSET"` would otherwise read as no filter at all and run the
    # whole suite in place of the tier (#2146 review).
    local opt
    for opt in --filter-tags --expect-bash --report-dir; do
        # --print-jobs exits before any test runs, so a regression fails fast
        # instead of starting the suite.
        run "$RUN" "$opt" "" --print-jobs
        [ "$status" -eq 2 ] || { printf '%s exited %s: %s\n' "$opt" "$status" "$output" >&2; false; }
        [[ "$output" == *"$opt needs a non-empty value"* ]] || false
    done
}

@test "ci.yml counts CPUs through run-bats.sh, not nproc, which macOS does not have" {
    refute_grep '(\$\(|`)[[:space:]]*nproc' "$BATS_TEST_DIRNAME/../.github/workflows/ci.yml"
}

@test "ci.yml: every job that runs the bats suite builds dotf once and exports DOTF_BIN" {
    run python3 -c "
import sys, yaml
jobs = yaml.safe_load(open('$BATS_TEST_DIRNAME/../.github/workflows/ci.yml'))['jobs']
bad = []
for name, job in jobs.items():
    steps = job.get('steps', [])
    runs_suite = any('run-bats.sh' in (s.get('run') or '') for s in steps)
    builds = any('DOTF_BIN=' in (s.get('run') or '') for s in steps)
    if runs_suite and not builds:
        bad.append(name)
if bad:
    print('no build-once DOTF_BIN step in: ' + ', '.join(bad)); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}
