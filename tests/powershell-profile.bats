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

# --- Aider tier functions ---

@test "profile.ps1 defines ai function for daily tier" {
    grep -q 'function ai {' "$PROFILE_SCRIPT"
}

@test "profile.ps1 ai function calls aider" {
    grep 'function ai {' "$PROFILE_SCRIPT" | grep -q 'aider'
}

@test "profile.ps1 defines aic function for coding tier" {
    grep -q 'function aic {' "$PROFILE_SCRIPT"
}

@test "profile.ps1 aic function uses qwen coder model" {
    grep 'function aic {' "$PROFILE_SCRIPT" | grep -q 'qwen.*coder'
}

@test "profile.ps1 defines aia function for architecture tier" {
    grep -q 'function aia {' "$PROFILE_SCRIPT"
}

@test "profile.ps1 aia function uses architect mode" {
    grep 'function aia {' "$PROFILE_SCRIPT" | grep -q '\-\-architect'
}

@test "profile.ps1 aia function uses deepseek speciale" {
    grep 'function aia {' "$PROFILE_SCRIPT" | grep -q 'speciale'
}

# --- Parity check: same models in bash and PowerShell ---

@test "parity: aic uses same model in aliases.zsh and profile.ps1" {
    bash_model=$(grep 'alias aic=' "$DOTFILES_DIR/.zsh/aliases.zsh" | grep -oP 'openrouter/[^ "]+')
    ps_model=$(grep 'function aic {' "$PROFILE_SCRIPT" | grep -oP 'openrouter/[^ }]+')
    [[ "$bash_model" = "$ps_model" ]]
}

@test "parity: aia uses same model in aliases.zsh and profile.ps1" {
    bash_model=$(grep 'alias aia=' "$DOTFILES_DIR/.zsh/aliases.zsh" | grep -oP 'openrouter/[^ "]+')
    ps_model=$(grep 'function aia {' "$PROFILE_SCRIPT" | grep -oP 'openrouter/[^ }]+')
    [[ "$bash_model" = "$ps_model" ]]
}

# --- Other functions ---

@test "profile.ps1 has project-init function" {
    grep -q 'function project-init' "$PROFILE_SCRIPT"
}

@test "profile.ps1 has prompt function" {
    grep -q 'function prompt' "$PROFILE_SCRIPT"
}

@test "profile.ps1 valid PowerShell syntax (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    pwsh -NoProfile -Command "Get-Content '$PROFILE_SCRIPT' | Out-Null; \$null = [System.Management.Automation.PSParser]::Tokenize((Get-Content '$PROFILE_SCRIPT' -Raw), [ref]\$null)"
}
