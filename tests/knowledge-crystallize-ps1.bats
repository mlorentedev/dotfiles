#!/usr/bin/env bats
# Tests for scripts/knowledge-crystallize.ps1 (structural + PSScriptAnalyzer)

load 'winpath'

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

@test "knowledge-crystallize.ps1 sources the shared project-key helper (no local encoder)" {
    # #689: the local Get-EncodedPath re-implemented the key encoding and drifted
    # (deleting ':' -> the wrong single-dash key). It now sources utils.ps1 and
    # resolves the key via Get-ClaudeProjectKey (dotf-backed single source).
    grep -q "utils.ps1" "$PS1_SCRIPT"
    grep -q 'Get-ClaudeProjectKey' "$PS1_SCRIPT"
    ! grep -q 'function Get-EncodedPath' "$PS1_SCRIPT"
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

@test "knowledge-crystallize.ps1 does not strip the drive colon (the #689 regression)" {
    # The bug mapped ':' to '' (delete), producing C-Users-... which Claude never
    # reads. The correct key maps ':' to '-' (C--Users-...). Guard both: the buggy
    # delete pattern is gone, and the decoder expects the double-dash drive key.
    ! grep -q "Replace.*':'.*''" "$PS1_SCRIPT"
    grep -q "A-Za-z\].*--" "$PS1_SCRIPT"
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
            \$results = Invoke-ScriptAnalyzer -Path '$(_winpath "$PS1_SCRIPT")' -Settings '$(_winpath "$DOTFILES_DIR/.PSScriptAnalyzerSettings.psd1")' -Severity Error,Warning
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
            '$(_winpath "$PS1_SCRIPT")', [ref]\$null, [ref]\$errors
        ) | Out-Null
        if (\$errors) {
            \$errors | ForEach-Object { Write-Error \$_.Message }
            exit 1
        }
        Write-Host 'Syntax OK'
    "
    [[ "$status" -eq 0 ]]
}
