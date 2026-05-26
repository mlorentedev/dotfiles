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

@test "profile.ps1 has agy alias (g -> agy, post-SDD-007 Antigravity migration)" {
    # gemini-cli replaced by Google Antigravity CLI ('agy'). The single-letter
    # alias 'g' carries forward — same finger pattern, new binary.
    grep -q 'Set-Alias.*-Name g -Value agy' "$PROFILE_SCRIPT"
}

# --- OpenCode alias (admin-conditional on Windows; aider sunset) ---

@test "profile.ps1 defines conditional oc function for opencode (--pure)" {
    # Guarded by Get-Command — non-admin Windows machines without opencode
    # skip the alias rather than producing broken references. Function form
    # (not Set-Alias) so the --pure flag can be passed; mirrors bash/zsh.
    grep -qE 'if \(Get-Command opencode' "$PROFILE_SCRIPT"
    grep -qE 'function oc\s+\{\s*opencode --pure' "$PROFILE_SCRIPT"
}

@test "profile.ps1 no longer defines aider tier functions (sunset)" {
    ! grep -qE '^function (ai|aic|aia) ' "$PROFILE_SCRIPT"
}

# --- Secrets sourcing (cross-OS parity: .bashrc sources load-secrets.sh) ---

@test "profile.ps1 sources load-secrets.ps1 (cross-OS parity with .bashrc)" {
    # .bashrc:63 sources load-secrets.sh. PowerShell parity requires profile.ps1
    # to dot-source load-secrets.ps1 so $env:NAN_API_KEY / OPENROUTER_API_KEY /
    # etc. are populated for opencode (reads {env:NAN_API_KEY} from opencode.jsonc),
    # agy, and copilot. Without this, opencode.jsonc references resolve to empty.
    grep -qE 'load-secrets\.ps1' "$PROFILE_SCRIPT"
    grep -qF 'Test-Path -LiteralPath' "$PROFILE_SCRIPT"
}

@test "parity: load-secrets sourced in both .bashrc and profile.ps1" {
    grep -qE 'load-secrets\.sh' "$DOTFILES_DIR/.bashrc"
    grep -qE 'load-secrets\.ps1' "$PROFILE_SCRIPT"
}

# --- Parity check: oc alias exists in both shells ---

@test "parity: oc with --pure defined in both aliases.zsh and profile.ps1" {
    # Cross-OS parity is the architectural invariant: same finger pattern,
    # same effective command. Zsh uses alias; PowerShell uses a function
    # because Set-Alias cannot pass arguments. Both invoke `opencode --pure`.
    grep -qE '^alias oc="opencode --pure"' "$DOTFILES_DIR/.zsh/aliases.zsh"
    grep -qE 'function oc\s+\{\s*opencode --pure' "$PROFILE_SCRIPT"
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
