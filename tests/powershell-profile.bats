#!/usr/bin/env bats
# Tests for powershell/profile.ps1 (structural)

load 'winpath'

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

@test "profile.ps1 defines conditional oc alias for opencode (plain, no --pure)" {
    # Guarded by Get-Command -- non-admin Windows machines without opencode
    # skip the alias rather than producing broken references. Plain Set-Alias
    # (no --pure) because empirically opencode launched bare from a Windows
    # terminal works correctly with MCPs+plugins (2026-05-26). Linux keeps
    # --pure kept from the abandoned DX-003 terminal-hang investigation.
    grep -qE 'if \(Get-Command opencode' "$PROFILE_SCRIPT"
    grep -qF 'Set-Alias -Name oc -Value opencode' "$PROFILE_SCRIPT"
    # Regression guard: ensure --pure isn't quietly reintroduced for Windows.
    ! grep -qE 'function oc\s+\{\s*opencode --pure' "$PROFILE_SCRIPT"
}

@test "profile.ps1 no longer defines aider tier functions (sunset)" {
    ! grep -qE '^function (ai|aic|aia) ' "$PROFILE_SCRIPT"
}

# --- Cross-machine path resolution (ADR-025): no hardcoded repo literal ---

@test "profile.ps1 dbg resolves nan-debug.sh via DOTFILES_REPO_DIR, not a hardcoded Projects\\dotfiles literal" {
    # Mirror of the Linux AC4 (shell-wrapper-dedup.bats): a relocated clone must
    # still find the repo script. The old literal 'Projects\dotfiles\scripts'
    # broke on any machine whose repo is not under ~/Projects/dotfiles.
    grep -qF '$env:DOTFILES_REPO_DIR' "$PROFILE_SCRIPT"
    ! grep -qF "'Projects\\dotfiles\\scripts\\nan-debug.sh'" "$PROFILE_SCRIPT"
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

# --- Cross-OS oc behaviour (intentional asymmetry, documented) ---

@test "oc is defined on both POSIX and PowerShell (intentional asymmetry on --pure)" {
    # Same finger pattern, different underlying invocation:
    #   Linux : oc() { opencode --pure ...} in .zsh/functions.sh (DX-003 hang workaround)
    #   Windows: Set-Alias -Name oc -Value opencode  (no hang observed)
    # Linux oc moved to functions.sh (shared bash/zsh) per REFACTOR-010; the
    # intentional --pure asymmetry is preserved. Once the Linux hang root cause is
    # found (DX-003, abandoned), the --pure workaround drops everywhere.
    grep -qE '^oc\(\) \{ opencode --pure' "$DOTFILES_DIR/.zsh/functions.sh"
    grep -qF 'Set-Alias -Name oc -Value opencode' "$PROFILE_SCRIPT"
}

# --- Other functions ---

@test "profile.ps1 has project-init function" {
    grep -q 'function project-init' "$PROFILE_SCRIPT"
}

@test "profile.ps1 has prompt function" {
    grep -q 'function prompt' "$PROFILE_SCRIPT"
}

# --- WIN-001: hc function (healthcheck alias mirror of Linux `hc`) ---

@test "profile.ps1 defines hc function (CLI-018: dotf doctor alias)" {
    grep -qE '^function hc' "$PROFILE_SCRIPT"
}

@test "profile.ps1 hc function wraps dotf doctor (CLI-018)" {
    # hc was a healthcheck.ps1 launcher; after CLI-018 it runs dotf doctor.
    grep -A6 '^function hc' "$PROFILE_SCRIPT" | grep -qF 'dotf doctor'
    ! grep -qF 'healthcheck.ps1' "$PROFILE_SCRIPT"
}

@test "profile.ps1 valid PowerShell syntax (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    pwsh -NoProfile -Command "Get-Content '$(_winpath "$PROFILE_SCRIPT")' | Out-Null; \$null = [System.Management.Automation.PSParser]::Tokenize((Get-Content '$(_winpath "$PROFILE_SCRIPT")' -Raw), [ref]\$null)"
}
