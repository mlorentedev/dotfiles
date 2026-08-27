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

@test "TEST-003: a post-setup dotf doctor step fails the job on a non-zero exit" {
    job="$(test_windows_job)"
    printf '%s\n' "$job" | grep -qF 'Post-setup doctor gate'
    printf '%s\n' "$job" | grep -qF 'dotf doctor'
    printf '%s\n' "$job" | grep -qF 'if ($LASTEXITCODE -ne 0) { throw'
}

@test "TEST-003: the source build is kept by Install-Dotf (DOTF_VERSION=dev on the setup step)" {
    job="$(test_windows_job)"
    printf '%s\n' "$job" | grep -qF 'DOTF_VERSION: dev'
}
