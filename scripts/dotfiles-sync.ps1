<#
.SYNOPSIS
    Push the dotfiles repo, then pull it into the installation

.DESCRIPTION
    Pushes the repo and pulls it into the local installation. sensitive/ is
    never touched, in either direction: secrets live in Bitwarden behind
    secrets/registry.yaml (ADR-028), and the one age blob left is git-tracked.
    A two-way copy of sensitive/ used to live here, and it copied retired
    blobs from the installation back into the repo (#1795).

.EXAMPLE
    .\dotfiles-sync.ps1

.NOTES
    Requires: git
    Environment variables:
      DOTFILES_DIR     - local dotfiles path (default: ~\.dotfiles)
      DOTFILES_REPO_DIR - repo dotfiles path (default: ~\Projects\dotfiles)
#>

# No parameters. CmdletBinding makes a stale -SecretsOnly fail loudly instead of
# landing in $args unnoticed: sensitive/ no longer syncs here (#1795).
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Configuration
$DotfilesLocal = if ($env:DOTFILES_DIR) { $env:DOTFILES_DIR } else { "$env:USERPROFILE\.dotfiles" }
$DotfilesRepo = if ($env:DOTFILES_REPO_DIR) { $env:DOTFILES_REPO_DIR } else { "$env:USERPROFILE\Projects\dotfiles" }

# Helpers
function Write-Info { param([string]$Message) Write-Host "-> $Message" -ForegroundColor Blue }
function Write-Ok { param([string]$Message) Write-Host "OK $Message" -ForegroundColor Green }
function Write-Warn { param([string]$Message) Write-Host "!  $Message" -ForegroundColor Yellow }
function Write-Err { param([string]$Message) Write-Host "x  $Message" -ForegroundColor Red }

# Validate directories
function Test-Dirs {
    if (-not (Test-Path $DotfilesLocal)) {
        Write-Err "Local dotfiles not found: $DotfilesLocal"
        return $false
    }
    if (-not (Test-Path $DotfilesRepo)) {
        Write-Err "Repo dotfiles not found: $DotfilesRepo"
        return $false
    }
    if ($DotfilesLocal -eq $DotfilesRepo) {
        Write-Warn "Local and repo are same directory"
        return $false
    }
    return $true
}

# Sync git repos
function Sync-Git {
    Write-Info "Pushing from repo..."
    $null = & git -C $DotfilesRepo diff --quiet 2>&1
    $null = & git -C $DotfilesRepo diff --cached --quiet 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Warn "  Uncommitted changes in repo - commit first"
        return
    }

    & git -C $DotfilesRepo push 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Ok "Push complete"
    } else {
        Write-Warn "  Nothing to push or push failed"
    }

    Write-Host ""
    Write-Info "Pulling to local..."
    & git -C $DotfilesLocal pull
    if ($LASTEXITCODE -eq 0) {
        Write-Ok "Pull complete"
    } else {
        Write-Err "Pull failed"
    }
}

# Main
Write-Host "Dotfiles Sync"
Write-Host "============="
Write-Host "Local: $DotfilesLocal"
Write-Host "Repo:  $DotfilesRepo"
Write-Host ""

if (-not (Test-Dirs)) { exit 1 }

Sync-Git
Write-Host ""
Write-Ok "Sync complete. Restart PowerShell to reload profile."
