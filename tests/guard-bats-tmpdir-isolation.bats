#!/usr/bin/env bats
# GUARD: no bats file builds a fixed path under $BATS_TMPDIR (CI-004 AC6).
#
# $BATS_TMPDIR reads like a per-run directory and is not: it is $TMPDIR, or
# /tmp, shared by every bats process on the machine. A fixture at
# "$BATS_TMPDIR/<name>" is therefore one directory for every concurrent run,
# and a teardown that removes it removes the other run's fixture mid-test.
#
# Measured 2026-09-27: two worktrees ran the suite at the same time, and
# check-doc-paths.bats (4 tests) and docs-drift.bats (1) failed with a
# different subset in each run; `diff: /tmp/sdd-005/ai.txt: No such file or
# directory`. CI never showed it: one run per runner, and within a run
# `--no-parallelize-within-files` keeps a file from racing itself. This repo
# runs parallel sessions in parallel worktrees daily, which is where it bites,
# and CI-004's own fixture audit had missed it for the same reason.
#
# Use $BATS_TEST_TMPDIR (per test, removed by bats), $BATS_FILE_TMPDIR (per
# file), or `mktemp -d "$BATS_TMPDIR/<name>.XXXXXX"`, which is unique.

@test "guard: every \$BATS_TMPDIR path in the suite is made unique by mktemp" {
    local offenders
    # Two steps, not a pipe: a pipe would report the second grep's status.
    grep -nE 'BATS_TMPDIR' "$BATS_TEST_DIRNAME"/*.bats "$BATS_TEST_DIRNAME"/*.bash \
        >"$BATS_TEST_TMPDIR/uses" 2>/dev/null || true
    offenders="$(grep -vE 'mktemp' "$BATS_TEST_TMPDIR/uses" | grep -vF "$BATS_TEST_FILENAME" || true)"
    [ -z "$offenders" ] || {
        printf 'fixed path under the shared $BATS_TMPDIR:\n%s\n' "$offenders"
        return 1
    }
}
