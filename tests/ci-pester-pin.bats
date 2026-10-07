#!/usr/bin/env bats
# test-windows must run the Pester that versions.conf declares, and a Gallery
# outage must read as one (#1605).
#
# It installed `Pester -MinimumVersion 5.5.0` from the PowerShell Gallery on
# every run, with no retry: whatever the Gallery called latest that day was the
# test runner, and one Gallery blip failed the job before a single test ran
# (run 35799905245, 2026-09-22). The install also cost ~20 s of every run.
#
# Asserted on the workflow text, because nothing here can run GitHub's runner.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    CI="$REPO/.github/workflows/ci.yml"
    # The Pester steps only: from the pin step to the suite run, comments dropped
    # so a comment naming the old form cannot satisfy or trip a check.
    STEPS=$(awk '/name: Resolve the Pester pin/,/Invoke-Pester -Path tests -CI/' "$CI" | grep -vE '^[[:space:]]*#')
}

@test "versions.conf pins Pester" {
    grep -qE '^PESTER_VERSION=[0-9]+\.[0-9]+\.[0-9]+$' "$REPO/versions.conf"
}

@test "ci: test-windows takes the Pester version from versions.conf, never an open minimum" {
    [ -n "$STEPS" ] || { echo "no 'Resolve the Pester pin' step in ci.yml" >&2; return 1; }
    printf '%s\n' "$STEPS" | grep -q 'PESTER_VERSION'
    if grep -vE '^[[:space:]]*#' "$CI" | grep -nE 'Pester.*-MinimumVersion'; then
        echo "a floating Pester minimum: pin it with -RequiredVersion from versions.conf" >&2
        return 1
    fi
    printf '%s\n' "$STEPS" | grep -qE 'Import-Module Pester -RequiredVersion'
}

@test "ci: the Pester install is cached, retried, and fails naming the Gallery" {
    printf '%s\n' "$STEPS" | grep -qE 'uses: actions/cache@[0-9a-f]{40}'
    printf '%s\n' "$STEPS" | grep -qE 'key: pester-.*PESTER|key: pester-.*steps\.pester\.outputs\.version'
    printf '%s\n' "$STEPS" | grep -qE 'for \(\$attempt'
    printf '%s\n' "$STEPS" | grep -qE '::error::.*PowerShell Gallery'
}
