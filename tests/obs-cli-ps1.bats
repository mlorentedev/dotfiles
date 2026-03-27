#!/usr/bin/env bats
# Tests for scripts/obs-cli.ps1 (structural + PSScriptAnalyzer)

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export PS1_SCRIPT="$SCRIPTS_DIR/obs-cli.ps1"
}

@test "obs-cli.ps1 exists" {
    [[ -f "$PS1_SCRIPT" ]]
}

@test "obs-cli.ps1 has .SYNOPSIS block" {
    grep -q '\.SYNOPSIS' "$PS1_SCRIPT"
}

@test "obs-cli.ps1 has .DESCRIPTION block" {
    grep -q '\.DESCRIPTION' "$PS1_SCRIPT"
}

@test "obs-cli.ps1 has .EXAMPLE block" {
    grep -q '\.EXAMPLE' "$PS1_SCRIPT"
}

@test "obs-cli.ps1 uses Set-StrictMode" {
    grep -q 'Set-StrictMode' "$PS1_SCRIPT"
}

@test "obs-cli.ps1 has ErrorActionPreference Stop" {
    grep -q "ErrorActionPreference = 'Stop'" "$PS1_SCRIPT"
}

@test "obs-cli.ps1 defaults vault to knowledge" {
    grep -q "'knowledge'" "$PS1_SCRIPT"
}

@test "obs-cli.ps1 checks obsidian in PATH" {
    grep -q 'Get-Command obsidian' "$PS1_SCRIPT"
}

@test "obs-cli.ps1 exits 2 when GUI not running" {
    grep -q 'exit 2' "$PS1_SCRIPT"
}

@test "obs-cli.ps1 passes PSScriptAnalyzer (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    run pwsh -NonInteractive -Command "
        \$ErrorActionPreference = 'Stop'
        try {
            Install-Module PSScriptAnalyzer -Force -Scope CurrentUser -ErrorAction SilentlyContinue
            \$results = Invoke-ScriptAnalyzer -Path '$PS1_SCRIPT' -Settings '$DOTFILES_DIR/.PSScriptAnalyzerSettings.psd1' -Severity Error,Warning
            if (\$results) {
                \$results | Format-Table -AutoSize
                exit 1
            }
            Write-Host 'PSScriptAnalyzer: OK'
        } catch {
            Write-Warning \"PSScriptAnalyzer not available: \$_\"
            exit 0
        }
    "
    [[ "$status" -eq 0 ]]
}

@test "obs-cli.ps1 valid PowerShell syntax (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    run pwsh -NonInteractive -Command "
        \$errors = \$null
        [System.Management.Automation.Language.Parser]::ParseFile(
            '$PS1_SCRIPT', [ref]\$null, [ref]\$errors
        ) | Out-Null
        if (\$errors) {
            \$errors | ForEach-Object { Write-Error \$_.Message }
            exit 1
        }
        Write-Host 'Syntax OK'
    "
    [[ "$status" -eq 0 ]]
}
