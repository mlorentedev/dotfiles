#!/usr/bin/env bats
# Tests for scripts/init-repo-github-defaults.ps1
# The .sh twin was deleted in CLI-014 (ported to `dotf init github`). The .ps1
# is retained as the Windows fallback until a Windows dotf install path exists
# (#380); these structural asserts lock its contract.

setup() {
    export REPO_DIR="$BATS_TEST_DIRNAME/.."
    export PS_SCRIPT="$REPO_DIR/scripts/init-repo-github-defaults.ps1"
}

@test "init-repo-github-defaults.ps1 exists" {
    [ -f "$PS_SCRIPT" ]
}

@test "init-repo-github-defaults.ps1 is ASCII-only (PSScriptAnalyzer rule)" {
    # No em dashes, arrows, smart quotes, ellipsis -- the recurring CI failure.
    ! grep -nP '[^\x00-\x7F]' "$PS_SCRIPT"
}

@test "init-repo-github-defaults.ps1 PATCHes delete_branch_on_merge=true" {
    grep -qE "delete_branch_on_merge=true" "$PS_SCRIPT"
}

@test "init-repo-github-defaults.ps1 is idempotent (early-exit on already-enabled)" {
    grep -q "Already enabled" "$PS_SCRIPT"
}

@test "init-repo-github-defaults.ps1 supports -DryRun switch" {
    grep -qE '\[switch\]\$DryRun' "$PS_SCRIPT"
}

@test "init-repo-github-defaults.ps1 derives owner/name from origin" {
    grep -qE 'remote get-url origin' "$PS_SCRIPT"
}
