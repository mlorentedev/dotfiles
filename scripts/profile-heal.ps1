<#
.SYNOPSIS
    Idempotently repair a corrupted PowerShell profile by reconstructing the
    dotfiles section from the SSOT (powershell/profile.ps1), after backing
    up the corrupted file. Companion to BUG-021 (setup-windows.ps1 fail-fast
    preflight). The rewrite keeps ONLY the dotfiles section: anything that
    lived outside the START/END markers survives in the backup, not in the
    healed profile. Heals this host's $PROFILE, or the file given as
    -ProfilePath (what dotf doctor --fix passes, so it heals what it measured).

.DESCRIPTION
    BUG-020 root cause: setup-windows.ps1 used to swallow -replace OOM errors
    silently and print [SUCCESS] regardless. Across many runs the deployed
    profile compounded into a 26 MB / 689K-line file with thousands of
    duplicated dotfiles sections and plugin-injected lines bleeding outside
    the START/END markers, producing parser errors on every session start.

    This script detects three corruption signals:

      1. Size > 1 MB (healthy profile is ~7 KB; 1 MB is ~140x oversized).
      2. More than 1 occurrence of the START or END marker. The SSOT contains
         neither, so a healthy deployed profile has exactly one pair; setup's
         splice replaces the first pair only, so a second one is the start of
         the accumulation. `dotf doctor` flags the same two signals (#531) and
         --fix runs this script, so the thresholds must agree or doctor would
         flag a profile this script declines to heal.
      3. PowerShell parser errors when [Parser]::ParseFile is run against it.

    When any signal trips, the existing profile is copied to
    <profile>.corrupted-yyyyMMdd-HHmmss.bak and replaced with a freshly-
    rendered dotfiles section (START marker + SSOT content + END marker).

    Cross-OS asymmetry note: Linux deploys .bashrc / .zshrc via
    deploy_file (utils.sh:454) which is a clean-copy under set -euo
    pipefail. No accumulation path exists on Linux, so no equivalent heal
    script ships there.

.NOTES
    Behaviour: silent on healthy profiles (exit 0, no output). Prints one
    line per heal action so doctor.ps1 -Fix can surface it. Always exits 0
    -- never blocks a session/setup on transient failures.

.PARAMETER VerboseOutput
    When passed, logs every check (not just heals).

.EXAMPLE
    pwsh -NoProfile -File profile-heal.ps1
    pwsh -NoProfile -File profile-heal.ps1 -VerboseOutput
#>

[CmdletBinding()]
param(
    [switch]$VerboseOutput,
    # The profile to heal. Empty means this host's $PROFILE, which is what a
    # hand invocation wants; dotf doctor --fix passes the file it measured so
    # detect and heal cannot split on a box whose Documents folder is
    # redirected (CLI-066, #1364).
    [string]$ProfilePath = ''
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'

$script:VerboseOutput = $VerboseOutput.IsPresent

function Write-HealLog {
    param([string]$Message)
    Write-Output "[profile-heal] $Message"
}

function Write-HealVerbose {
    param([string]$Message)
    if ($script:VerboseOutput) { Write-HealLog $Message }
}

# Resolve the SSOT next to this script (scripts/profile-heal.ps1 ->
# ../powershell/profile.ps1 in the dotfiles repo; on a deployed machine
# the script lives at $env:USERPROFILE\scripts\ and the SSOT comes from
# $env:DOTFILES_REPO_DIR or the canonical dotfiles repo path).
function Resolve-SsotProfile {
    $candidates = @()
    if ($env:DOTFILES_REPO_DIR) {
        $candidates += (Join-Path $env:DOTFILES_REPO_DIR 'powershell\profile.ps1')
    }
    $candidates += (Join-Path $env:USERPROFILE 'Projects\dotfiles\powershell\profile.ps1')
    $candidates += (Join-Path $PSScriptRoot '..\powershell\profile.ps1')

    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path -LiteralPath $candidate)) {
            return (Resolve-Path -LiteralPath $candidate).Path
        }
    }
    return $null
}

# Detect corruption signals. Returns a hashtable describing whether the
# profile is corrupt and which signal(s) tripped. A non-corrupt result
# leaves $reasons empty.
function Get-ProfileCorruptionState {
    param([string]$ProfilePath)

    $state = [ordered]@{
        Exists       = $false
        Size         = 0
        StartMarkers = 0
        EndMarkers   = 0
        ParseErrors  = 0
        Reasons      = @()
    }

    if (-not (Test-Path -LiteralPath $ProfilePath)) {
        return $state
    }
    $state.Exists = $true

    $info = Get-Item -LiteralPath $ProfilePath -ErrorAction Stop
    $state.Size = $info.Length

    if ($state.Size -gt 1MB) {
        $state.Reasons += "size $($state.Size) bytes > 1MB"
    }

    if ($state.Size -gt 0) {
        try {
            $raw = Get-Content -LiteralPath $ProfilePath -Raw -ErrorAction Stop
            $startMarker = '# >>> DOTFILES PROFILE >>>'
            $endMarker = '# <<< DOTFILES PROFILE <<<'
            $state.StartMarkers = [regex]::Matches($raw, [regex]::Escape($startMarker)).Count
            $state.EndMarkers = [regex]::Matches($raw, [regex]::Escape($endMarker)).Count
            if ($state.StartMarkers -gt 1 -or $state.EndMarkers -gt 1) {
                $state.Reasons += "marker counts (start=$($state.StartMarkers), end=$($state.EndMarkers)) exceed 1"
            }
        } catch {
            $state.Reasons += "unreadable: $($_.Exception.Message)"
        }

        try {
            $parseErrors = $null
            [System.Management.Automation.Language.Parser]::ParseFile(
                $ProfilePath, [ref]$null, [ref]$parseErrors
            ) | Out-Null
            if ($parseErrors -and $parseErrors.Count -gt 0) {
                $state.ParseErrors = $parseErrors.Count
                $state.Reasons += "parser errors: $($parseErrors.Count)"
            }
        } catch {
            $state.Reasons += "parser threw: $($_.Exception.Message)"
        }
    }

    return $state
}

function Repair-Profile {
    param(
        [string]$ProfilePath,
        [string]$SsotPath
    )

    $startMarker = '# >>> DOTFILES PROFILE >>>'
    $endMarker = '# <<< DOTFILES PROFILE <<<'

    # Backup BEFORE overwriting. Timestamped suffix lets repeated heals
    # accumulate without collision.
    $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
    $backup = "$ProfilePath.corrupted-$stamp.bak"
    try {
        Copy-Item -LiteralPath $ProfilePath -Destination $backup -Force -ErrorAction Stop
        Write-HealLog "backed up corrupted profile to $backup"
    } catch {
        Write-HealLog "ERROR: backup failed -- aborting heal: $($_.Exception.Message)"
        return $false
    }

    $sourceContent = Get-Content -LiteralPath $SsotPath -Raw -ErrorAction Stop
    $newContent = "$startMarker`r`n$sourceContent`r`n$endMarker"

    try {
        Set-Content -LiteralPath $ProfilePath -Value $newContent -Encoding UTF8 -ErrorAction Stop
        Write-HealLog "rewrote profile from SSOT: $SsotPath"
        return $true
    } catch {
        Write-HealLog "ERROR: write failed: $($_.Exception.Message)"
        return $false
    }
}

# --- Main ---

$profilePath = if ($ProfilePath) { $ProfilePath } else { $PROFILE }
$ssotPath = Resolve-SsotProfile

if (-not $ssotPath) {
    Write-HealLog "ERROR: cannot find powershell/profile.ps1 SSOT; checked DOTFILES_REPO_DIR and default paths"
    exit 0
}
Write-HealVerbose "SSOT: $ssotPath"
Write-HealVerbose "Target: $profilePath"

$state = Get-ProfileCorruptionState -ProfilePath $profilePath

if (-not $state.Exists) {
    Write-HealVerbose "profile does not exist; nothing to heal"
    exit 0
}

if ($state.Reasons.Count -eq 0) {
    Write-HealVerbose "profile healthy (size=$($state.Size), markers=$($state.StartMarkers)/$($state.EndMarkers), parseErrors=$($state.ParseErrors))"
    exit 0
}

Write-HealLog ("corruption detected: " + ($state.Reasons -join '; '))
$ok = Repair-Profile -ProfilePath $profilePath -SsotPath $ssotPath
if ($ok) {
    Write-HealLog "heal complete -- restart PowerShell or dot-source the profile"
}

exit 0
