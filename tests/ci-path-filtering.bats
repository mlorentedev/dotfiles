#!/usr/bin/env bats
# Tests for HARNESS-041: CI matrix path filtering (Issue #552)

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export CI_YML="$DOTFILES_DIR/.github/workflows/ci.yml"
}

@test "HARNESS-041: ci.yml defines a changes job with dorny/paths-filter" {
    grep -q 'changes:' "$CI_YML"
    grep -q 'dorny/paths-filter' "$CI_YML"
}

@test "HARNESS-041: matrix jobs depend on changes job" {
    for job in lint lint-powershell test test-windows integration; do
        awk -v j="$job:" '$0 ~ "^  " j {flag=1; next} /^  [a-z0-9_-]+:/ {flag=0} flag {print}' "$CI_YML" | grep -q 'changes'
    done
}

@test "HARNESS-041: heavy steps carry changes conditional filter" {
    grep -q 'needs.changes.outputs.code' "$CI_YML"
    grep -q 'needs.changes.outputs.powershell' "$CI_YML"
}

@test "HARNESS-041: critical deploy and agent paths are included in code filter" {
    for path in 'ai/**' 'cli/**' 'sensitive/**' '.zshrc' '.bashrc' '.gitconfig' 'setup-linux.sh' 'setup-windows.ps1'; do
        grep -q -- "$path" "$CI_YML"
    done
}
