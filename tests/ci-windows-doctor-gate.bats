#!/usr/bin/env bats
# TEST-003 (#1298): the test-windows job installed dotf into ~/.local/bin and
# never put it on PATH, so every dotf-gated setup block skipped with a warning,
# dotf doctor never ran in CI, and the job stayed green over a box that failed
# four doctor checks. These guards keep the three masking layers from returning:
# dotf reachable before setup, the deploy dir left at its real-box default, and
# a post-setup doctor step that fails the job.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export CI_YML="$DOTFILES_DIR/.github/workflows/ci.yml"
}

# The test-windows job body only.
test_windows_job() {
    awk '/^  test-windows:/{flag=1; next} /^  [a-z0-9_-]+:/{flag=0} flag' "$CI_YML"
}

@test "TEST-003: test-windows builds dotf from the PR and puts ~/.local/bin on GITHUB_PATH before setup" {
    job="$(test_windows_job)"
    printf '%s\n' "$job" | grep -qF 'go build -o "$bin\dotf.exe" ./cmd/dotf'
    printf '%s\n' "$job" | grep -qF 'Add-Content $env:GITHUB_PATH $bin'
    build_line=$(printf '%s\n' "$job" | grep -n 'Build dotf from this PR' | head -1 | cut -d: -f1)
    setup_line=$(printf '%s\n' "$job" | grep -n 'run: .\\setup-windows.ps1' | head -1 | cut -d: -f1)
    [ -n "$build_line" ] && [ -n "$setup_line" ]
    [ "$build_line" -lt "$setup_line" ]
}

@test "TEST-003: setup runs with the real-box deploy dir, not DOTFILES_DIR overridden to the workspace" {
    run grep -F 'DOTFILES_DIR: ${{ github.workspace }}' "$CI_YML"
    [ "$status" -ne 0 ]
}

@test "TEST-003: a post-setup doctor gate step runs the gate script, which exits non-zero on an unlisted or stale FAIL" {
    job="$(test_windows_job)"
    printf '%s\n' "$job" | grep -qF 'Post-setup doctor gate'
    printf '%s\n' "$job" | grep -qF 'pwsh -NoProfile -File .github/scripts/doctor-gate.ps1'
    grep -qF '& dotf doctor' "$DOTFILES_DIR/.github/scripts/doctor-gate.ps1"
    grep -qF 'exit 1' "$DOTFILES_DIR/.github/scripts/doctor-gate.ps1"
}

@test "TEST-003: the gate refreshes PATH like a fresh terminal before running doctor" {
    grep -qF "[Environment]::GetEnvironmentVariable('PATH', 'User')" "$DOTFILES_DIR/.github/scripts/doctor-gate.ps1"
}

@test "TEST-003: every known-failure entry names its owning ticket" {
    # An entry without a ticket is an allow rule nobody owns.
    awk 'BEGIN{ok=1} /^[[:space:]]*$/{seen=0; next} /^#/{ if ($0 ~ /#[0-9]+/) seen=1; next } { if (!seen) { print "no ticket before: " $0; ok=0 } seen=0 } END{exit !ok}' \
        "$DOTFILES_DIR/.github/scripts/doctor-gate-known-failures.txt"
}

@test "test-windows refreshes PATH and declares copilot before Invoke-Pester (TEST-006)" {
    # The Pester step inherits the runner's original PATH; without the refresh
    # every tool-gated suite skipped as green. The declaration turns an absent
    # copilot on the runner into a FAIL inside the suite.
    grep -B12 'Invoke-Pester -Path tests -CI' "$CI_YML" | grep -qF "pwsh -NoProfile -Command '. ./scripts/utils.ps1; Sync-SessionPath; \$env:PATH'"
    grep -B12 'Invoke-Pester -Path tests -CI' "$CI_YML" | grep -qF 'Sync-SessionPath'
    grep -B12 'Invoke-Pester -Path tests -CI' "$CI_YML" | grep -qF "DOTFILES_CI_EXPECT_COPILOT = '1'"
    grep -qF 'DOTFILES_CI_EXPECT_COPILOT' "$BATS_TEST_DIRNAME/copilot-native-skills.Tests.ps1"
}

# #2013 X4: the runner has no identity, and doctor's machine scope declares
# that. The gate used to be fed a fabricated age key, vault skeleton and
# obsidian stub instead, plus a flag that covered Bitwarden alone; fabrication
# is a second source of truth for what a runner is, and it hid the vault
# checks' real behaviour on a machine without one.
@test "test-windows: the doctor gate runs the machine scope and the job fabricates no identity (#2013 X4)" {
    grep -qF '& dotf doctor --scope machine' "$DOTFILES_DIR/.github/scripts/doctor-gate.ps1"
    run grep -F '$env:DOTFILES_DOCTOR_NO_IDENTITY' "$DOTFILES_DIR/.github/scripts/doctor-gate.ps1"
    [ "$status" -eq 1 ] || false
    job="$(test_windows_job)"
    for fabricated in 'age-keygen -o' 'Projects\knowledge' 'obsidian.cmd'; do
        run grep -F "$fabricated" <<<"$job"
        [ "$status" -eq 1 ] || { echo "test-windows still fabricates: $fabricated"; false; }
    done
    # Least privilege: the doctor step needs no token under the machine scope.
    gate="$(printf '%s\n' "$job" | awk '/- name: Post-setup doctor gate/{f=1} f && /^      - / && !/Post-setup doctor gate/{exit} f')"
    [ -n "$gate" ] || false
    run grep -E '^[^#]*GH_TOKEN' <<<"$gate"
    [ "$status" -eq 1 ] || { echo "the doctor gate step carries GH_TOKEN again"; false; }
}

# The machine scope skips doctor's gh checks too, and spec-issue-state was the
# one that verified something on the runner (CI-004 AC8): a spec whose issue
# closed must still turn the job red. The audit is its own step with the token.
@test "test-windows: the spec issue-state audit runs with GH_TOKEN after the doctor gate (CI-004 AC8)" {
    job="$(test_windows_job)"
    step="$(printf '%s\n' "$job" | awk '/- name: Spec issue-state audit/{f=1} f && /^      - / && !/Spec issue-state audit/{exit} f')"
    [ -n "$step" ] || false
    printf '%s\n' "$step" | grep -qF 'GH_TOKEN: ${{ github.token }}'
    printf '%s\n' "$step" | grep -qF 'dotf spec audit'
    printf '%s\n' "$step" | grep -qF 'if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }'
    gate_line=$(printf '%s\n' "$job" | grep -n 'Post-setup doctor gate' | head -1 | cut -d: -f1)
    audit_line=$(printf '%s\n' "$job" | grep -n 'Spec issue-state audit' | head -1 | cut -d: -f1)
    [ "$gate_line" -lt "$audit_line" ] || false
}
