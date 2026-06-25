<#
.SYNOPSIS
    Bidirectional sync between dotfiles repo and installation

.DESCRIPTION
    Syncs secrets (*.secret.age, .secrets-audit.log) bidirectionally
    (newest wins), then pushes repo and pulls to local. The registry.yaml
    mapping is git-tracked, so it travels with the repo push/pull.

.PARAMETER SecretsOnly
    Only sync secrets, skip git push/pull.

.EXAMPLE
    .\dotfiles-sync.ps1

.EXAMPLE
    .\dotfiles-sync.ps1 -SecretsOnly

.NOTES
    Requires: git, age (for secrets)
    Environment variables:
      DOTFILES_DIR     - local dotfiles path (default: ~\.dotfiles)
      DOTFILES_REPO_DIR - repo dotfiles path (default: ~\Projects\dotfiles)
#>

param(
    [switch]$SecretsOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Configuration
$DotfilesLocal = if ($env:DOTFILES_DIR) { $env:DOTFILES_DIR } else { "$env:USERPROFILE\.dotfiles" }
$DotfilesRepo = if ($env:DOTFILES_REPO_DIR) { $env:DOTFILES_REPO_DIR } else { "$env:USERPROFILE\Projects\dotfiles" }
$SensitiveDir = 'sensitive'

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

# Sync a single file between two directories (newest wins)
function Sync-File {
    param(
        [string]$FileName,
        [string]$LocalDir,
        [string]$RepoDir
    )

    $localFile = Join-Path $LocalDir $FileName
    $repoFile = Join-Path $RepoDir $FileName

    $localExists = Test-Path $localFile
    $repoExists = Test-Path $repoFile

    if ($localExists -and $repoExists) {
        $localTime = (Get-Item $localFile).LastWriteTime
        $repoTime = (Get-Item $repoFile).LastWriteTime
        if ($localTime -gt $repoTime) {
            Copy-Item $localFile $repoFile -Force
            Write-Info "  $FileName (-> repo)"
            return $true
        } elseif ($repoTime -gt $localTime) {
            Copy-Item $repoFile $localFile -Force
            Write-Info "  $FileName (repo -> local)"
            return $true
        }
    } elseif ($localExists) {
        Copy-Item $localFile $repoFile -Force
        Write-Info "  $FileName (-> repo) [new]"
        return $true
    } elseif ($repoExists) {
        Copy-Item $repoFile $localFile -Force
        Write-Info "  $FileName (repo -> local) [new]"
        return $true
    }
    return $false
}

# Sync secrets bidirectionally
function Sync-Secrets {
    $localDir = Join-Path $DotfilesLocal $SensitiveDir
    $repoDir = Join-Path $DotfilesRepo $SensitiveDir

    if (-not (Test-Path $localDir)) { Write-Warn "Local sensitive dir not found"; return }
    if (-not (Test-Path $repoDir)) { Write-Warn "Repo sensitive dir not found"; return }

    Write-Info "Syncing secrets..."

    $synced = 0
    $filesToSync = @('.secrets-audit.log')

    # Collect all .secret.age files from both dirs
    $ageFiles = @()
    $ageFiles += Get-ChildItem -Path $localDir -Filter '*.secret.age' -ErrorAction SilentlyContinue
    $ageFiles += Get-ChildItem -Path $repoDir -Filter '*.secret.age' -ErrorAction SilentlyContinue
    $ageNames = $ageFiles | ForEach-Object { $_.Name } | Select-Object -Unique
    $filesToSync += $ageNames

    foreach ($file in ($filesToSync | Select-Object -Unique)) {
        if (Sync-File -FileName $file -LocalDir $localDir -RepoDir $repoDir) {
            $synced++
        }
    }

    if ($synced -eq 0) {
        Write-Info "  All secrets already in sync"
    }
}

# Sync git repos
function Sync-Git {
    Write-Info "Pushing from repo..."
    $diffResult = & git -C $DotfilesRepo diff --quiet 2>&1
    $cachedResult = & git -C $DotfilesRepo diff --cached --quiet 2>&1
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

Sync-Secrets
Write-Host ""

if ($SecretsOnly) {
    Write-Ok "Secrets sync complete"
    exit 0
}

Sync-Git
Write-Host ""
Write-Ok "Sync complete. Restart PowerShell to reload profile."
