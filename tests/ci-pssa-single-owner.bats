#!/usr/bin/env bats
# lint-powershell is the one owner of PowerShell lint, and it runs the
# PSScriptAnalyzer versions.conf declares (#2052).
#
# Three bats tests re-linted files lint-powershell already covers. Each ran
# `Install-Module PSScriptAnalyzer -Force` against the Gallery: 226 s and 155 s
# on run 37581508879 against a ~20 s baseline. Two of them also passed when
# the module could not be loaded (`catch { ...; exit 0 }`), so a Gallery outage
# read as "lint clean".
#
# Asserted on the text, because nothing here can run GitHub's runner.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    CI="$REPO/.github/workflows/ci.yml"
    # The lint-powershell job only, comments dropped.
    JOB=$(awk '/^  lint-powershell:/,/^  test:/' "$CI" | grep -vE '^[[:space:]]*#')
}

@test "versions.conf pins PSScriptAnalyzer" {
    grep -qE '^PSSCRIPTANALYZER_VERSION=[0-9]+\.[0-9]+\.[0-9]+$' "$REPO/versions.conf"
}

@test "no test file runs PSScriptAnalyzer: lint-powershell is its one owner" {
    # This file is excluded: its header quotes the removed install.
    hits=$(grep -lE 'Invoke-ScriptAnalyzer|Install-Module.*PSScriptAnalyzer' \
        "$REPO"/tests/*.bats "$REPO"/tests/*.Tests.ps1 2>/dev/null |
        grep -v '/ci-pssa-single-owner\.bats$' || true)
    [ -z "$hits" ] || { printf 'PSScriptAnalyzer run outside lint-powershell:\n%s\n' "$hits" >&2; return 1; }
}

@test "ci: lint-powershell installs and imports the pinned PSScriptAnalyzer, never a floating one" {
    [ -n "$JOB" ] || { echo "no lint-powershell job in ci.yml" >&2; return 1; }
    printf '%s\n' "$JOB" | grep -q 'PSSCRIPTANALYZER_VERSION='
    printf '%s\n' "$JOB" | grep -qE 'Install-Module PSScriptAnalyzer -RequiredVersion'
    printf '%s\n' "$JOB" | grep -qE 'Import-Module PSScriptAnalyzer -RequiredVersion'
    if printf '%s\n' "$JOB" | grep -E 'Install-Module' | grep -vq -- '-RequiredVersion'; then
        echo "an Install-Module without -RequiredVersion: the Gallery's latest would be the linter" >&2
        return 1
    fi
}

@test "ci: the PSScriptAnalyzer install is retried and fails naming the Gallery" {
    printf '%s\n' "$JOB" | grep -qF "for (\$attempt"
    printf '%s\n' "$JOB" | grep -qE '::error::PSScriptAnalyzer .*PowerShell Gallery'
}
