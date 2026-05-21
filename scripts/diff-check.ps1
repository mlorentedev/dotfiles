<#
.SYNOPSIS
    Detect drift between repo and deploy-dir (~/.dotfiles).

.DESCRIPTION
    The repo ($DOTFILES_REPO_DIR or default $env:USERPROFILE\Projects\dotfiles)
    is the source of truth. setup-windows.ps1 COPIES files into $DOTFILES_DIR
    (deploy-dir, default $env:USERPROFILE\.dotfiles). If you edit a file in
    the repo without re-running setup, the repo and deploy-dir diverge silently
    -- every shell still reads the stale deploy-dir copy. This script reports
    those divergences.

    PowerShell port of scripts/diff-check.sh (REFACTOR-003). Same allowlist,
    same exit semantics, same comparison logic (Get-FileHash SHA256 is the
    PowerShell equivalent of `cmp -s` byte-equality).

.PARAMETER VerboseOutput
    When set, prints a per-file unified diff for every drifted file.

.PARAMETER Help
    Print usage and exit.

.EXAMPLE
    pwsh -NoProfile -File diff-check.ps1
    pwsh -NoProfile -File diff-check.ps1 -VerboseOutput

.NOTES
    Exit codes:
      0  no drift
      1  drift detected
      2  setup error (missing dirs, not a git repo, etc.)
#>

[CmdletBinding()]
param(
    [switch]$VerboseOutput,
    [switch]$Help
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'

if ($Help) {
    @"
Usage: diff-check.ps1 [-VerboseOutput] [-Help]

Reports files that differ between the repo and the deploy-dir
(\$env:USERPROFILE\.dotfiles). Both must exist; only files tracked by git
in the repo are checked.

Env:
  DOTFILES_REPO_DIR   Repo root (default: parent of script's dir)
  DOTFILES_DIR        Deploy-dir (default: \$env:USERPROFILE\.dotfiles)

Exit codes:
  0  no drift
  1  drift detected
  2  setup error (missing dirs, not a git repo, etc.)
"@ | Write-Host
    exit 0
}

# Resolve paths from env vars with sensible defaults. Match the bash script's
# resolution order so cross-OS behaviour stays consistent.
$ScriptDir = $PSScriptRoot
$RepoDir   = if ($env:DOTFILES_REPO_DIR) { $env:DOTFILES_REPO_DIR } else { Split-Path $ScriptDir -Parent }
$DeployDir = if ($env:DOTFILES_DIR)      { $env:DOTFILES_DIR }      else { Join-Path $env:USERPROFILE '.dotfiles' }

if (-not (Test-Path -LiteralPath $RepoDir -PathType Container)) {
    Write-Host "Error: repo not found: $RepoDir" -ForegroundColor Red
    exit 2
}
if (-not (Test-Path -LiteralPath $DeployDir -PathType Container)) {
    Write-Host "Error: deploy-dir not found: $DeployDir" -ForegroundColor Red
    exit 2
}
if (-not (Test-Path -LiteralPath (Join-Path $RepoDir '.git'))) {
    Write-Host "Error: not a git repo: $RepoDir" -ForegroundColor Red
    exit 2
}

# Top-level files and dirs that setup-{linux,windows} copy into the deploy-dir.
# MUST mirror diff-check.sh's _is_managed allowlist; Linux-only items
# (.zshrc, .bashrc, tmux.conf) get implicitly skipped on Windows via the
# Test-Path guards below since they are not deployed there.
function Test-Managed {
    param([string]$RelPath)
    # Exact-match top-level files
    $files = @('versions.conf', '.zshrc', '.bashrc', '.profile', '.gitconfig', 'tmux.conf')
    if ($files -contains $RelPath) { return $true }
    # Prefix-match top-level dirs (forward slash on both Linux git output AND Windows)
    foreach ($prefix in @('.zsh/', 'ssh/', 'scripts/', 'sensitive/')) {
        if ($RelPath.StartsWith($prefix)) { return $true }
    }
    return $false
}

Write-Host "Checking drift: $RepoDir -> $DeployDir" -ForegroundColor Cyan

# git ls-files outputs forward-slash paths even on Windows.
Push-Location $RepoDir
try {
    $trackedFiles = (& git ls-files) -split "`n" | Where-Object { $_ }
} finally {
    Pop-Location
}

$driftCount     = 0
$checkedCount   = 0
$unmanagedCount = 0
$driftedPaths   = @()

foreach ($relPath in $trackedFiles) {
    if (-not (Test-Managed $relPath)) {
        $unmanagedCount++
        continue
    }

    $repoFile   = Join-Path $RepoDir   ($relPath -replace '/', '\')
    $deployFile = Join-Path $DeployDir ($relPath -replace '/', '\')

    if (-not (Test-Path -LiteralPath $deployFile -PathType Leaf)) { continue }
    if (-not (Test-Path -LiteralPath $repoFile   -PathType Leaf)) { continue }

    $checkedCount++

    $repoHash   = (Get-FileHash -LiteralPath $repoFile   -Algorithm SHA256).Hash
    $deployHash = (Get-FileHash -LiteralPath $deployFile -Algorithm SHA256).Hash

    if ($repoHash -ne $deployHash) {
        $driftCount++
        $driftedPaths += $relPath
        Write-Host "  DRIFT: $relPath" -ForegroundColor Red
        if ($VerboseOutput) {
            $repoLines   = Get-Content -LiteralPath $repoFile
            $deployLines = Get-Content -LiteralPath $deployFile
            $diff = Compare-Object -ReferenceObject $deployLines -DifferenceObject $repoLines
            $diff | ForEach-Object {
                $marker = if ($_.SideIndicator -eq '=>') { '+' } else { '-' }
                Write-Host "    $marker $($_.InputObject)"
            }
        }
    }
}

Write-Host ""
Write-Host "Checked: $checkedCount managed files (ignored $unmanagedCount tracked but not deployed)" -ForegroundColor Cyan

if ($driftCount -eq 0) {
    Write-Host "No drift detected -- repo and deploy-dir agree" -ForegroundColor Green
    exit 0
}

Write-Host "Drift detected in $driftCount file(s)" -ForegroundColor Yellow
Write-Host "  Run '.\setup-windows.ps1' to refresh the deploy-dir from the repo"
Write-Host "  (or use -VerboseOutput to see the diff)"
exit 1
