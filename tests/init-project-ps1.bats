#!/usr/bin/env bats
# Tests for scripts/init-project.ps1 (structural + PSScriptAnalyzer)

load 'winpath'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export PS1_SCRIPT="$DOTFILES_DIR/scripts/init-project.ps1"
}

@test "init-project.ps1 exists" {
    [[ -f "$PS1_SCRIPT" ]]
}

# --- Structural checks ---

@test "init-project.ps1 has .SYNOPSIS block" {
    grep -q '\.SYNOPSIS' "$PS1_SCRIPT"
}

@test "init-project.ps1 has .DESCRIPTION block" {
    grep -q '\.DESCRIPTION' "$PS1_SCRIPT"
}

@test "init-project.ps1 has .EXAMPLE block" {
    grep -q '\.EXAMPLE' "$PS1_SCRIPT"
}

@test "init-project.ps1 has ProjectName parameter" {
    grep -q 'ProjectName' "$PS1_SCRIPT"
}

@test "init-project.ps1 has Stack parameter" {
    grep -q 'Stack' "$PS1_SCRIPT"
}

@test "init-project.ps1 supports python stack" {
    grep -q '"python"' "$PS1_SCRIPT"
}

@test "init-project.ps1 supports go stack" {
    grep -q '"go"' "$PS1_SCRIPT"
}

@test "init-project.ps1 supports node stack" {
    grep -q '"node"' "$PS1_SCRIPT"
}

@test "init-project.ps1 supports ts stack" {
    grep -q '"ts"' "$PS1_SCRIPT"
}

@test "init-project.ps1 supports none stack" {
    grep -q '"none"' "$PS1_SCRIPT"
}

@test "init-project.ps1 creates Knowledge Vault entries" {
    grep -q 'Knowledge Vault' "$PS1_SCRIPT"
}

@test "init-project.ps1 creates 11-tasks.md" {
    grep -q '11-tasks.md' "$PS1_SCRIPT"
}

@test "init-project.ps1 does not seed 90-lessons.md (knowledge-placement)" {
    # Lessons live in the repo docs/lessons.md, not the vault — the script must
    # not Set-Content a vault 90-lessons.md (the guidance text mentioning it is fine).
    ! grep -qE 'Set-Content.*90-lessons\.md' "$PS1_SCRIPT"
}

@test "init-project.ps1 injects CLAUDE.md" {
    grep -q 'CLAUDE.md' "$PS1_SCRIPT"
}

@test "init-project.ps1 injects AGY.md (post-SDD-007: agy replaces gemini-cli)" {
    grep -q 'AGY.md' "$PS1_SCRIPT"
    # Asserting the OLD file is NOT referenced (regression guard):
    ! grep -qE 'Injected GEMINI\.md|GEMINI\.md.*not found' "$PS1_SCRIPT"
}

@test "init-project.ps1 initializes git" {
    grep -q 'git init' "$PS1_SCRIPT"
}

@test "init-project.ps1 creates .gitignore" {
    grep -q '\.gitignore' "$PS1_SCRIPT"
}

# --- Parity checks ---

@test "parity: init-project.ps1 poetry includes typer" {
    grep 'poetry add' "$PS1_SCRIPT" | grep -q 'typer'
}

@test "parity: init-project.ps1 poetry includes rich" {
    grep 'poetry add' "$PS1_SCRIPT" | grep -q 'rich'
}

@test "parity: init-project.ps1 poetry includes pydantic" {
    grep 'poetry add' "$PS1_SCRIPT" | grep -q 'pydantic'
}

# --- REFACTOR-004 wiring. init-project.ps1 is the Windows fallback for `dotf
# init` until a Windows dotf install path exists (#380). The .sh twins were
# deleted in CLI-014; these asserts lock the surviving .ps1 wiring and guard the
# two features removed from the repo-init path: standards (dropped, 0/6
# adoption) and --work-sdk (vault work, moved to dotf vault, #388). ---

@test "init-project.ps1 declares -SkipAgents/-SkipGithub switches" {
    grep -qF '[switch]$SkipAgents' "$PS1_SCRIPT"
    grep -qF '[switch]$SkipGithub' "$PS1_SCRIPT"
}

@test "init-project.ps1 invokes the 2 kept init-repo-* helpers" {
    grep -qF 'init-repo-agents.ps1' "$PS1_SCRIPT"
    grep -qF 'init-repo-github-defaults.ps1' "$PS1_SCRIPT"
}

@test "init-project.ps1 github invocation guarded by origin remote check" {
    grep -qF 'remote.origin.url' "$PS1_SCRIPT"
}

@test "regression: init-project.ps1 no longer wires init-repo-standards (dropped)" {
    ! grep -qF 'init-repo-standards.ps1' "$PS1_SCRIPT"
    ! grep -qF 'SkipStandards' "$PS1_SCRIPT"
}

@test "regression: init-project.ps1 no longer carries -WorkSdk (moved to dotf vault #388)" {
    ! grep -qiF 'WorkSdk' "$PS1_SCRIPT"
}

# --- PSScriptAnalyzer ---

@test "init-project.ps1 passes PSScriptAnalyzer (if pwsh available)" {
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

@test "init-project.ps1 valid PowerShell syntax (if pwsh available)" {
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
