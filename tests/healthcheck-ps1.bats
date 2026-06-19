#!/usr/bin/env bats
# Tests for scripts/healthcheck.ps1 (structural + PSScriptAnalyzer).
# The Linux .sh twin was ported to `dotf doctor` (CLI-012) and deleted; its
# behavioural intent now lives in cli/internal/doctor (go test). healthcheck.ps1
# is kept until the Windows port lands dotf on PATH there.

load 'winpath'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export PS1_SCRIPT="$SCRIPTS_DIR/healthcheck.ps1"
}

# --- File presence + doc comment block ---

@test "healthcheck.ps1 exists" {
    [[ -f "$PS1_SCRIPT" ]]
}

@test "healthcheck.ps1 has .SYNOPSIS block" {
    grep -q '\.SYNOPSIS' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 has .DESCRIPTION block" {
    grep -q '\.DESCRIPTION' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 has .EXAMPLE block" {
    grep -q '\.EXAMPLE' "$PS1_SCRIPT"
}

# --- PowerShell conventions (mirror doctor.ps1 / knowledge-crystallize.ps1) ---

@test "healthcheck.ps1 has [CmdletBinding()]" {
    grep -q '\[CmdletBinding()\]' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 uses Set-StrictMode -Version Latest" {
    grep -q 'Set-StrictMode -Version Latest' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 sets ErrorActionPreference" {
    grep -q "ErrorActionPreference = 'Continue'" "$PS1_SCRIPT"
}

# --- Output helpers (pass/fail/skip/section) ---

@test "healthcheck.ps1 defines Write-Pass" {
    grep -q '^function Write-Pass' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 defines Write-Fail" {
    grep -q '^function Write-Fail' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 defines Write-Skip" {
    grep -q '^function Write-Skip' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 defines Write-Section" {
    grep -q '^function Write-Section' "$PS1_SCRIPT"
}

# --- 13 sections present (12 -> 13 after DX-006 re-added a section) ---

@test "healthcheck.ps1 has all 13 sections numbered 1/13..13/13" {
    for n in 1 2 3 4 5 6 7 8 9 10 11 12 13; do
        grep -qF "Write-Section '${n}/13'" "$PS1_SCRIPT" || {
            echo "Missing: Write-Section '${n}/13'"
            return 1
        }
    done
}

@test "healthcheck.ps1 section 1/13 covers Core Tools in PATH" {
    grep -qF "Write-Section '1/13' 'Core Tools in PATH'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 2/13 covers Versioned Tool Paths" {
    grep -qF "Write-Section '2/13' 'Versioned Tool Paths'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 3/13 covers Version Match" {
    grep -qF "Write-Section '3/13' 'Version Match (versions.conf)'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 4/13 covers Key Files / Junctions (Windows-specific naming)" {
    grep -qF "Write-Section '4/13' 'Key Files / Junctions'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 5/13 covers Environment Variables" {
    grep -qF "Write-Section '5/13' 'Environment Variables'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 6/13 covers Optional Tools" {
    grep -qF "Write-Section '6/13' 'Optional Tools'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 7/13 covers Knowledge Vault" {
    grep -qF "Write-Section '7/13' 'Knowledge Vault'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 8/13 covers Secrets Integrity" {
    grep -qF "Write-Section '8/13' 'Secrets Integrity'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 9/13 covers tmux" {
    grep -qF "Write-Section '9/13' 'tmux'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 10/13 covers OpenCode" {
    grep -qF "Write-Section '10/13' 'OpenCode'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 11/13 covers drift" {
    grep -qF "Write-Section '11/13' 'Repo - Deploy-Dir Drift'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 12/13 covers Antigravity CLI Health (SDD-007)" {
    grep -qF "Write-Section '12/13' 'Antigravity CLI Health'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 13/13 covers Orca Copilot Hook (DX-006)" {
    grep -qF "Write-Section '13/13' 'Orca Copilot Hook (DX-006)'" "$PS1_SCRIPT"
}

# --- Linux-only sections emit SKIP with explanation (per WIN-001 design) ---

@test "healthcheck.ps1 section 9/13 (tmux) emits SKIP with explanation" {
    grep -qF "Linux-only by design" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 section 11/13 invokes diff-check.ps1 (REFACTOR-003)" {
    # Pre-REFACTOR-003 this was a SKIP with REFACTOR-003 in the message.
    # Post-REFACTOR-003 the section invokes diff-check.ps1 and switches on
    # the exit code (0 = PASS no drift, 1 = FAIL drift, other = setup error).
    grep -qF 'diff-check.ps1' "$PS1_SCRIPT"
    grep -qF 'REFACTOR-003' "$PS1_SCRIPT"
    # The invocation must be inside the sec 12 block.
    sed -n "/Write-Section '11\/13' 'Repo - Deploy-Dir Drift'/,/^# ==/p" "$PS1_SCRIPT" | grep -qF 'diff-check.ps1'
}

# --- Cross-section content asserts ---

@test "healthcheck.ps1 verifies yarn version match against versions.conf" {
    grep -qF "YARN_VERSION" "$PS1_SCRIPT"
    grep -qF "Test-Command 'yarn'" "$PS1_SCRIPT"
}

@test "healthcheck.ps1 sources versions.conf" {
    grep -q 'versions\.conf' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 references env-mapping.conf for secrets" {
    grep -q 'env-mapping\.conf' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 references obsidian-linter data.json" {
    grep -q 'obsidian-linter' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 references BUG-012 marketplace junction" {
    grep -qF 'BUG-012' "$PS1_SCRIPT"
}

# --- BUG-014: install-state assertion for claude-mem ---
# The BUG-012 junction check above validates a proxy artifact (filesystem
# junction) but cannot detect the case where the marketplace dir is present
# but the plugin is never installed (installed_plugins.json omits claude-mem).
# BUG-014 adds a canonical assertion against installed_plugins.json so a
# false-positive PASS becomes a true FAIL when the plugin is actually missing.

@test "healthcheck.ps1 checks installed_plugins.json for claude-mem@thedotmack (BUG-014)" {
    grep -qF 'installed_plugins.json' "$PS1_SCRIPT"
    grep -qF 'claude-mem@thedotmack' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 references BUG-014 install-state check" {
    grep -qF 'BUG-014' "$PS1_SCRIPT"
}

# --- Exit code policy ---

@test "healthcheck.ps1 exits 0 on success" {
    grep -q '^exit 0' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 exits 1 on failure" {
    grep -q '^    exit 1' "$PS1_SCRIPT"
}

@test "healthcheck.ps1 gates exit 1 on \$script:Failed counter" {
    grep -qF 'if ($script:Failed -gt 0)' "$PS1_SCRIPT"
}

# --- PSScriptAnalyzer ---

@test "healthcheck.ps1 passes PSScriptAnalyzer (if pwsh available)" {
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

@test "healthcheck.ps1 parses as valid PowerShell (if pwsh available)" {
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
