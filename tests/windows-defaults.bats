#!/usr/bin/env bats
# Tests for scripts/windows-defaults.ps1 (WIN-005 -- HKCU engineering defaults,
# the mathiasbynens .macos analog). Structural + PSScriptAnalyzer here;
# behavioral (sandboxed registry writes, idempotency) in
# tests/windows-defaults.Tests.ps1 (Pester, Windows-only).

load 'winpath'
load 'lib/refute'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export PS1_SCRIPT="$SCRIPTS_DIR/windows-defaults.ps1"
    export SETUP_SCRIPT="$DOTFILES_DIR/setup-windows.ps1"
}

# --- File presence + doc block ---

@test "windows-defaults.ps1 exists" {
    [[ -f "$PS1_SCRIPT" ]]
}

@test "windows-defaults.ps1 has .SYNOPSIS block" {
    grep -q '\.SYNOPSIS' "$PS1_SCRIPT"
}

@test "windows-defaults.ps1 has .DESCRIPTION block" {
    grep -q '\.DESCRIPTION' "$PS1_SCRIPT"
}

@test "windows-defaults.ps1 has .EXAMPLE block" {
    grep -q '\.EXAMPLE' "$PS1_SCRIPT"
}

@test "windows-defaults.ps1 uses Set-StrictMode" {
    grep -q 'Set-StrictMode -Version Latest' "$PS1_SCRIPT"
}

@test "windows-defaults.ps1 has ErrorActionPreference Stop" {
    grep -q "ErrorActionPreference = 'Stop'" "$PS1_SCRIPT"
}

# --- HKCU-only invariant (AC: all writes target HKCU:\ only) ---

@test "windows-defaults.ps1 never references HKLM (HKCU-only invariant)" {
    run grep -in 'HKLM' "$PS1_SCRIPT"
    [ "$status" -eq 1 ] || { printf 'HKCU-only invariant broken:\n%s\n' "$output" >&2; return 1; }
}

@test "windows-defaults.ps1 validates -Root stays inside HKCU at runtime" {
    grep -q 'Root -notmatch' "$PS1_SCRIPT"
}

@test "windows-defaults.ps1 -Root defaults to HKCU:" {
    grep -qE "\\\$Root = 'HKCU:'" "$PS1_SCRIPT"
}

# --- R2: registry paths as a named table, not scattered literals ---

@test "windows-defaults.ps1 declares the defaults as a named table (R2)" {
    grep -q '$Defaults = @(' "$PS1_SCRIPT"
}

# --- R3: Win10 vs Win11 branch on OS build, without touching HKLM ---

@test "windows-defaults.ps1 branches on OS build for version-specific keys (R3)" {
    grep -q 'OSVersion' "$PS1_SCRIPT"
    grep -q '22000' "$PS1_SCRIPT"
}

# --- R1 + anti-scope: document explorer restart, never perform it ---

@test "windows-defaults.ps1 tells the user to restart Explorer (R1)" {
    grep -qi 'restart' "$PS1_SCRIPT"
}

@test "windows-defaults.ps1 never restarts explorer.exe itself (anti-scope)" {
    refute_grep 'Stop-Process|taskkill|Restart-Computer' "$PS1_SCRIPT"
}

# --- setup-windows.ps1 integration (AC: -WithDefaults flag, OFF by default) ---

@test "setup-windows.ps1 declares the -WithDefaults switch" {
    grep -q '\[switch\]$WithDefaults' "$SETUP_SCRIPT"
}

@test "setup-windows.ps1 gates the invocation on the flag (opt-in, R5)" {
    grep -q 'if ($WithDefaults)' "$SETUP_SCRIPT"
}

@test "setup-windows.ps1 deploys windows-defaults.ps1 to ScriptsDir" {
    grep -q 'windows-defaults.ps1' "$SETUP_SCRIPT"
    grep -qE 'Copy-Item \$winDefaultsSource' "$SETUP_SCRIPT"
}

@test "setup-windows.ps1 BUG-005 re-exec forwards -WithDefaults to pwsh" {
    # The re-exec block warned: '@args is sufficient' only while param() was
    # empty. With a named switch, the flag must be forwarded explicitly or a
    # PS 5.1 invocation silently drops it.
    grep -q "forward += '-WithDefaults'" "$SETUP_SCRIPT"
}

# --- README documents the opt-in flag ---

@test "README documents -WithDefaults" {
    grep -q 'WithDefaults' "$DOTFILES_DIR/README.md"
}

# --- PSScriptAnalyzer + syntax (strict variant: catch exits 1, never 0) ---

@test "windows-defaults.ps1 passes PSScriptAnalyzer (if pwsh available)" {
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

@test "windows-defaults.ps1 valid PowerShell syntax (if pwsh available)" {
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
