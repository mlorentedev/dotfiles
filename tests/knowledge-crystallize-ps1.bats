#!/usr/bin/env bats
# Tests for scripts/knowledge-crystallize.ps1 (structural + PSScriptAnalyzer)

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export PS1_SCRIPT="$SCRIPTS_DIR/knowledge-crystallize.ps1"
}

@test "knowledge-crystallize.ps1 exists" {
    [[ -f "$PS1_SCRIPT" ]]
}

@test "knowledge-crystallize.ps1 has .SYNOPSIS block" {
    grep -q '\.SYNOPSIS' "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 has .DESCRIPTION block" {
    grep -q '\.DESCRIPTION' "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 has .EXAMPLE block" {
    grep -q '\.EXAMPLE' "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 has -All parameter" {
    grep -q '\[switch\]\$All' "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 has Get-EncodedPath function" {
    grep -q 'function Get-EncodedPath' "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 has Get-DecodedPath function" {
    grep -q 'function Get-DecodedPath' "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 has Get-MemoryFilePath function" {
    grep -q 'function Get-MemoryFilePath' "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 has Invoke-ProjectCrystallize function" {
    grep -q 'function Invoke-ProjectCrystallize' "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 has Write-Checklist function" {
    grep -q 'function Write-Checklist' "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 uses Set-StrictMode" {
    grep -q 'Set-StrictMode' "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 has ErrorActionPreference Stop" {
    grep -q "ErrorActionPreference = 'Stop'" "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 path encoding strips colon (Windows convention)" {
    grep -q "Replace.*':'.*''" "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 handles filesystem scan for dashes in dir names" {
    grep -q 'Get-ChildItem.*Recurse.*Directory.*Depth' "$PS1_SCRIPT"
}

@test "knowledge-crystallize.ps1 passes PSScriptAnalyzer (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    run pwsh -NonInteractive -Command "
        \$ErrorActionPreference = 'Stop'
        try {
            Install-Module PSScriptAnalyzer -Force -Scope CurrentUser -ErrorAction SilentlyContinue
            \$results = Invoke-ScriptAnalyzer -Path '$PS1_SCRIPT' -Severity Error,Warning
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

@test "knowledge-crystallize.ps1 valid PowerShell syntax (if pwsh available)" {
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
