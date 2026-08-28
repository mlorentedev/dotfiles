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
