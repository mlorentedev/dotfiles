#!/usr/bin/env bats
# Tests for powershell/profile.ps1 (structural)

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export PROFILE_SCRIPT="$DOTFILES_DIR/powershell/profile.ps1"
}

@test "profile.ps1 exists" {
    [[ -f "$PROFILE_SCRIPT" ]]
}

# --- AI tool aliases ---

@test "profile.ps1 has claude alias" {
    grep -q 'Set-Alias.*-Name c -Value claude' "$PROFILE_SCRIPT"
}

@test "profile.ps1 has gemini alias" {
    grep -q 'Set-Alias.*-Name g -Value gemini' "$PROFILE_SCRIPT"
}

# --- OpenCode alias (admin-conditional on Windows; aider sunset) ---

@test "profile.ps1 defines conditional oc alias for opencode" {
    # Guarded by Get-Command — non-admin Windows machines without opencode
    # skip the alias rather than producing broken references.
    grep -qE 'if \(Get-Command opencode' "$PROFILE_SCRIPT"
    grep -qE 'Set-Alias -Name oc -Value opencode' "$PROFILE_SCRIPT"
}

@test "profile.ps1 no longer defines aider tier functions (sunset)" {
    ! grep -qE '^function (ai|aic|aia) ' "$PROFILE_SCRIPT"
}

# --- Parity check: oc alias exists in both shells ---

@test "parity: oc alias defined in both aliases.zsh and profile.ps1" {
    grep -qE '^alias oc=' "$DOTFILES_DIR/.zsh/aliases.zsh"
    grep -qE 'Set-Alias -Name oc' "$PROFILE_SCRIPT"
}

# --- Other functions ---

@test "profile.ps1 has project-init function" {
    grep -q 'function project-init' "$PROFILE_SCRIPT"
}

@test "profile.ps1 has prompt function" {
    grep -q 'function prompt' "$PROFILE_SCRIPT"
}

# --- WIN-001: hc function (healthcheck alias mirror of Linux `hc`) ---

@test "profile.ps1 defines hc function (WIN-001 healthcheck alias)" {
    grep -qE '^function hc' "$PROFILE_SCRIPT"
}

@test "profile.ps1 hc function references SCRIPTS_DIR\\healthcheck.ps1" {
    grep -qF 'healthcheck.ps1' "$PROFILE_SCRIPT"
}

@test "profile.ps1 valid PowerShell syntax (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    pwsh -NoProfile -Command "Get-Content '$PROFILE_SCRIPT' | Out-Null; \$null = [System.Management.Automation.PSParser]::Tokenize((Get-Content '$PROFILE_SCRIPT' -Raw), [ref]\$null)"
}
