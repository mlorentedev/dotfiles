#!/usr/bin/env bats
# Tests for scripts/orca-hook-tune.ps1 (DX-006 Orca/Copilot hook fix).
# Structural + PSScriptAnalyzer + parse. Behavioural coverage (apply/idempotent/
# -Check/backup) lives in tests/orca-hook-tune.Tests.ps1 (Pester).

load 'winpath'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export PS1_SCRIPT="$SCRIPTS_DIR/orca-hook-tune.ps1"
}

# --- File presence + doc comment block ---

@test "orca-hook-tune.ps1 exists" {
    [[ -f "$PS1_SCRIPT" ]]
}

@test "orca-hook-tune.ps1 has .SYNOPSIS block" {
    grep -q '\.SYNOPSIS' "$PS1_SCRIPT"
}

@test "orca-hook-tune.ps1 has .DESCRIPTION block" {
    grep -q '\.DESCRIPTION' "$PS1_SCRIPT"
}

@test "orca-hook-tune.ps1 has .EXAMPLE block" {
    grep -q '\.EXAMPLE' "$PS1_SCRIPT"
}

@test "orca-hook-tune.ps1 references DX-006" {
    grep -qF 'DX-006' "$PS1_SCRIPT"
}

# --- PowerShell conventions (mirror healthcheck.ps1 / windows-defaults.ps1) ---

@test "orca-hook-tune.ps1 has [CmdletBinding()]" {
    grep -qF '[CmdletBinding()]' "$PS1_SCRIPT"
}

@test "orca-hook-tune.ps1 uses Set-StrictMode -Version Latest" {
    grep -qF 'Set-StrictMode -Version Latest' "$PS1_SCRIPT"
}

@test "orca-hook-tune.ps1 sets ErrorActionPreference to Stop" {
    grep -qF "ErrorActionPreference = 'Stop'" "$PS1_SCRIPT"
}

# --- Test seams + safety invariants ---

@test "orca-hook-tune.ps1 exposes HookConfig/HookScript/TimeoutSec/Check params" {
    grep -qF '$HookConfig' "$PS1_SCRIPT"
    grep -qF '$HookScript' "$PS1_SCRIPT"
    grep -qF '$TimeoutSec' "$PS1_SCRIPT"
    grep -qF '$Check' "$PS1_SCRIPT"
}

@test "orca-hook-tune.ps1 backs up before writing" {
    grep -qF 'Backup-HookFile' "$PS1_SCRIPT"
}

@test "orca-hook-tune.ps1 targets Orca's generated hook files" {
    grep -qF '.copilot\hooks\orca.json' "$PS1_SCRIPT"
    grep -qF '.orca\agent-hooks\copilot-hook.ps1' "$PS1_SCRIPT"
}

@test "orca-hook-tune.ps1 swaps Invoke-WebRequest for HttpWebRequest" {
    grep -qF 'HttpWebRequest' "$PS1_SCRIPT"
    grep -qF 'Invoke-WebRequest' "$PS1_SCRIPT"
}

# --- PSScriptAnalyzer ---

@test "orca-hook-tune.ps1 passes PSScriptAnalyzer (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    run pwsh -NonInteractive -NoProfile -Command "
        \$ErrorActionPreference = 'Stop'
        try {
            if (-not (Get-Module -ListAvailable PSScriptAnalyzer)) {
                Install-Module PSScriptAnalyzer -Force -Scope CurrentUser -ErrorAction SilentlyContinue
            }
            \$results = Invoke-ScriptAnalyzer -Path '$(_winpath "$PS1_SCRIPT")' -Settings '$(_winpath "$DOTFILES_DIR/.PSScriptAnalyzerSettings.psd1")' -Severity Error,Warning
            if (\$results) {
                \$results | Format-Table -AutoSize
                exit 1
            }
            Write-Host 'PSScriptAnalyzer: OK'
        } catch {
            Write-Host \"PSScriptAnalyzer error: \$_\"
            exit 1
        }
    "
    [[ "$status" -eq 0 ]] || {
        echo "$output"
        return 1
    }
}

@test "orca-hook-tune.ps1 parses as valid PowerShell (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    run pwsh -NonInteractive -NoProfile -Command "
        try {
            [scriptblock]::Create((Get-Content -Raw '$(_winpath "$PS1_SCRIPT")')) | Out-Null
            Write-Host 'PARSE OK'
        } catch {
            Write-Host \"PARSE FAIL: \$_\"
            exit 1
        }
    "
    [[ "$status" -eq 0 ]]
}
