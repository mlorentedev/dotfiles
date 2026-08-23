#!/usr/bin/env bats
# Tests for POLISH-003: PSScriptAnalyzer full repository coverage

load 'lib/refute'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export CI_YML="$DOTFILES_DIR/.github/workflows/ci.yml"
}

@test "POLISH-003: lint-powershell uses Get-ChildItem for dynamic discovery" {
    grep -q 'Get-ChildItem' "$CI_YML"
}

@test "POLISH-003: lint-powershell does not hardcode static script list" {
    refute_grep_fixed '$scripts = @(' "$CI_YML"
}

@test "POLISH-003: repo contains at least 20 powershell scripts subject to analysis" {
    count=$(find "$DOTFILES_DIR" -type f \( -name "*.ps1" -o -name "*.psm1" \) -not -path "*/.git/*" | wc -l)
    [ "$count" -ge 20 ]
}
