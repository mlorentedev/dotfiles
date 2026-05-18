#!/usr/bin/env bats
# Tests for scripts/init-repo-github-defaults.sh

setup() {
    export REPO_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPT="$REPO_DIR/scripts/init-repo-github-defaults.sh"
    export PS_SCRIPT="$REPO_DIR/scripts/init-repo-github-defaults.ps1"
}

@test "init-repo-github-defaults.sh exists and is executable" {
    [ -x "$SCRIPT" ]
}

@test "init-repo-github-defaults.sh passes bash syntax" {
    bash -n "$SCRIPT"
}

@test "init-repo-github-defaults.sh passes zsh syntax" {
    zsh -n "$SCRIPT"
}

@test "init-repo-github-defaults.sh uses set -euo pipefail" {
    grep -q 'set -euo pipefail' "$SCRIPT"
}

@test "init-repo-github-defaults.sh PATCHes delete_branch_on_merge=true" {
    grep -qE 'delete_branch_on_merge=true' "$SCRIPT"
}

@test "init-repo-github-defaults.sh is idempotent (early-exit on already-enabled)" {
    grep -q 'Already enabled, nothing to do' "$SCRIPT"
}

@test "init-repo-github-defaults.sh supports --dry-run" {
    grep -qE '\-\-dry-run' "$SCRIPT"
}

@test "init-repo-github-defaults.sh supports --repo override" {
    grep -qE '\-\-repo' "$SCRIPT"
}

@test "init-repo-github-defaults.sh --help works" {
    run "$SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage:"* ]]
}

@test "init-repo-github-defaults.sh rejects unknown args" {
    run "$SCRIPT" --bogus
    [ "$status" -ne 0 ]
}

@test "init-repo-github-defaults.sh derives owner/name from SSH origin URL" {
    # Sanity-check the sed pipeline that strips git@github.com: / .git
    grep -qF 'git@github' "$SCRIPT"
    grep -qF 'https?://github' "$SCRIPT"
    grep -qF '.git$' "$SCRIPT"
}

# --- PowerShell parity ---

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
