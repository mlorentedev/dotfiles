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
    # The lint-powershell job's step scripts, parsed rather than cut by line
    # range: a range ending at the next job's key silently spans the rest of
    # the file the day that job is renamed or moved.
    JOB=$(python3 - "$CI" <<'PY'
import sys, yaml
job = yaml.safe_load(open(sys.argv[1]))["jobs"]["lint-powershell"]
print("\n".join(s.get("run", "") for s in job["steps"]))
PY
)
}

@test "versions.conf pins PSScriptAnalyzer" {
    grep -qE '^PSSCRIPTANALYZER_VERSION=[0-9]+\.[0-9]+\.[0-9]+$' "$REPO/versions.conf"
}

@test "no test file runs PSScriptAnalyzer: lint-powershell is its one owner" {
    # Every test file at any depth. The scan is asserted to have read files, so
    # a wrong root or an empty glob cannot pass as "nothing found".
    local n hits rc
    n=$(find "$REPO/tests" -type f \( -name '*.bats' -o -name '*.Tests.ps1' -o -name '*.bash' \) | grep -c .)
    [ "$n" -gt 50 ] || { echo "the scan found $n test files under $REPO/tests" >&2; return 1; }
    # grep exits 1 on no match, which is the pass; 2 is an error and fails. This
    # file is excluded: its header quotes the removed install.
    hits=$(grep -rlE --include='*.bats' --include='*.Tests.ps1' --include='*.bash' \
        --exclude='ci-pssa-single-owner.bats' \
        'Invoke-ScriptAnalyzer|Install-Module.*PSScriptAnalyzer' "$REPO/tests") && rc=0 || rc=$?
    [ "$rc" -le 1 ] || { echo "the scan itself failed (grep exit $rc)" >&2; return 1; }
    [ -z "$hits" ] || { printf 'PSScriptAnalyzer run outside lint-powershell:\n%s\n' "$hits" >&2; return 1; }
}

@test "ci: a versions.conf change runs lint-powershell, so a pin bump lints on its own PR" {
    python3 - "$CI" <<'PY'
import sys, yaml
steps = yaml.safe_load(open(sys.argv[1]))["jobs"]["changes"]["steps"]
filters = yaml.safe_load(next(s for s in steps if s.get("id") == "filter")["with"]["filters"])
sys.exit(0 if "versions.conf" in filters["powershell"] else 1)
PY
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
