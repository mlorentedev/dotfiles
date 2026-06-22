#!/usr/bin/env bats
# Tests for scripts/profile-heal.ps1 (BUG-020 recovery script).
#
# Companion of BUG-021 (setup-windows.ps1 fail-fast preflight). When the
# preflight detects a corrupted profile (size > 1 MB or marker count > 2),
# it points the user at this script. The script reconstructs the profile
# from the SSOT (powershell/profile.ps1), wrapped in fresh dotfiles markers,
# after backing up the corrupted file to <profile>.corrupted-<timestamp>.bak.
#
# Cross-OS note: Linux (setup-linux.sh) deploys .bashrc / .zshrc via
# deploy_file (clean-copy under set -e), so no equivalent heal script is
# needed. Asymmetry is intentional and documented here.

load 'winpath'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export PS1_SCRIPT="$SCRIPTS_DIR/profile-heal.ps1"
    export SSOT_PROFILE="$DOTFILES_DIR/powershell/profile.ps1"
}

# --- File presence + doc block ---

@test "profile-heal.ps1 exists" {
    [[ -f "$PS1_SCRIPT" ]]
}

@test "profile-heal.ps1 has .SYNOPSIS block" {
    grep -q '\.SYNOPSIS' "$PS1_SCRIPT"
}

@test "profile-heal.ps1 has .DESCRIPTION block" {
    grep -q '\.DESCRIPTION' "$PS1_SCRIPT"
}

@test "profile-heal.ps1 has [CmdletBinding()]" {
    grep -q '\[CmdletBinding()\]' "$PS1_SCRIPT"
}

@test "profile-heal.ps1 uses Set-StrictMode -Version Latest" {
    grep -q 'Set-StrictMode -Version Latest' "$PS1_SCRIPT"
}

# --- Corruption signals (must mirror setup-windows.ps1 preflight) ---

@test "profile-heal.ps1 detects size corruption (> 1 MB)" {
    # Same threshold as BUG-021 preflight. Keeps the two scripts in sync.
    grep -qE '\.(Length|Size)\s+-gt\s+1MB|Size\s+-gt\s+1MB' "$PS1_SCRIPT"
}

@test "profile-heal.ps1 detects marker-count corruption (> 2 of either marker)" {
    grep -qE 'markerCount|MarkerCount|StartMatches|EndMatches|StartMarkers|EndMarkers' "$PS1_SCRIPT"
    grep -qE -- '-gt\s+2' "$PS1_SCRIPT"
}

@test "profile-heal.ps1 detects parser errors as a corruption signal" {
    # PowerShell can syntax-check a file via [System.Management.Automation.Language.Parser]::ParseFile.
    # If the deployed profile fails to parse, that is a corruption signal even
    # if size/marker counts look healthy (e.g., truncated mid-block).
    grep -qF 'ParseFile' "$PS1_SCRIPT"
}

# --- Recovery contract: backup + reconstruct ---

@test "profile-heal.ps1 backs up corrupted profile before write" {
    # A safety net: NEVER overwrite without backup. Tag with timestamp to
    # support multiple heals on the same machine without overwrite collision.
    grep -qE 'corrupted-.*\.bak|\.bak.*corrupted|Copy-Item.*-Destination' "$PS1_SCRIPT"
}

@test "profile-heal.ps1 reconstructs profile from powershell/profile.ps1 SSOT" {
    grep -qF 'powershell\profile.ps1' "$PS1_SCRIPT" || grep -qF 'powershell/profile.ps1' "$PS1_SCRIPT"
}

@test "profile-heal.ps1 writes the dotfiles section between START/END markers" {
    grep -qF '# >>> DOTFILES PROFILE >>>' "$PS1_SCRIPT"
    grep -qF '# <<< DOTFILES PROFILE <<<' "$PS1_SCRIPT"
}

@test "profile-heal.ps1 exits 0 when profile is healthy (idempotent no-op)" {
    # Same shape as claude-mem-heal.ps1: silent on healthy installs, prints
    # one line per heal action. Always exit 0 (never block session start).
    grep -qE 'exit 0|return$' "$PS1_SCRIPT"
}

@test "profile-heal.ps1 valid PowerShell syntax (if pwsh available)" {
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

# --- Setup deploy + doctor integration (cross-script contract) ---

@test "setup-windows.ps1 deploys profile-heal.ps1 to ScriptsDir" {
    grep -q 'profile-heal.ps1' "$DOTFILES_DIR/setup-windows.ps1"
}

# CLI-018: doctor.ps1 was retired and its -Fix auto-invocation of
# profile-heal.ps1 (BUG-020) is NOT yet ported to `dotf doctor --fix`
# (runHeals only invokes claude-mem-heal.sh). Recovery still works manually:
# profile-heal.ps1 stays deployed and the BUG-021 setup preflight points at it.
# The doctor auto-heal port is tracked in #531; once landed, assert it in the Go
# doctor fix tests (cli/internal/doctor), not here.

# --- Cross-OS asymmetry documented (Linux uses clean-copy, no heal needed) ---

@test "parity (documented asymmetry): Linux deploys profile via deploy_file (no heal needed)" {
    # setup-linux.sh uses deploy_file (utils.sh:454) which is a clean-copy
    # under set -euo pipefail -- the file is byte-equivalent to the SSOT each
    # run, so accumulation cannot occur. Asserting the deploy_file usage is
    # in place is the Linux side of this contract.
    grep -qE 'deploy_file\s+"\$DOTFILES_DIR/\.bashrc"\s+"\$HOME/\.bashrc"' "$DOTFILES_DIR/setup-linux.sh"
    grep -qE 'deploy_file\s+"\$DOTFILES_DIR/\.zshrc"\s+"\$HOME/\.zshrc"' "$DOTFILES_DIR/setup-linux.sh"
}
