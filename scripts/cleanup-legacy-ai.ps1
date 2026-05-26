# cleanup-legacy-ai.ps1 - one-shot script to remove legacy AI tooling artifacts
# from the user's Windows machine. NOT auto-run by setup-windows.ps1.
#
# Mirror of scripts/cleanup-legacy-ai.sh (cross-OS parity per SDD-007).
# ASCII-only by repo convention (pattern-powershell-ascii-only).
#
# Removes (after per-category confirmation):
#   - legacy @google/gemini-cli npm install (replaced by agy)
#   - aider-chat pipx / pip install (sunset by SDD-007)
#   - $env:USERPROFILE\.config\aider, .aider* cache files
#
# Does NOT touch:
#   - $env:USERPROFILE\.gemini   (that is agy's home dir)
#   - $env:USERPROFILE\.dotfiles (managed by setup-windows.ps1)
#   - ai\agy\               (agy stays as the replacement for legacy gemini-cli)
#
# Usage:
#   .\scripts\cleanup-legacy-ai.ps1              # dry-run + prompts per category
#   .\scripts\cleanup-legacy-ai.ps1 -Yes         # auto-confirm everything
#   .\scripts\cleanup-legacy-ai.ps1 -DryRun      # only list, never touch

[CmdletBinding()]
param(
    [switch]$Yes,
    [switch]$DryRun,
    [switch]$Help
)

Set-StrictMode -Version Latest

function Write-Info    { param([string]$m) Write-Host "[INFO] $m"    -ForegroundColor Blue }
function Write-Success { param([string]$m) Write-Host "[SUCCESS] $m" -ForegroundColor Green }
function Write-Warn    { param([string]$m) Write-Host "[WARNING] $m" -ForegroundColor Yellow }

if ($Help) {
    @"
cleanup-legacy-ai.ps1 - remove legacy AI tooling artifacts (Windows mirror)

Usage:
    .\scripts\cleanup-legacy-ai.ps1              dry-run + prompts per category
    .\scripts\cleanup-legacy-ai.ps1 -Yes         auto-confirm everything
    .\scripts\cleanup-legacy-ai.ps1 -DryRun      list only, no changes

Removes:
    1. legacy @google/gemini-cli (npm global)
    2. aider-chat (pipx / pip)
    3. stale config and cache dirs (.aider, .config\aider)

Preserves:
    %USERPROFILE%\.gemini  (agy home dir - agy stays as the gemini-cli replacement)
"@
    return
}

function Confirm-Action {
    if ($Yes)    { return $true }
    if ($DryRun) { return $false }
    $reply = Read-Host "  Proceed? [y/N]"
    return ($reply -match '^(y|Y|yes|YES)$')
}

function Invoke-OrDry {
    param([string]$Description, [scriptblock]$Action)
    if ($DryRun) {
        Write-Info "  [dry-run] would: $Description"
        return
    }
    Write-Info "  running: $Description"
    try { & $Action } catch { Write-Warn "  command exited non-zero (continuing): $_" }
}

# ---------------------------------------------------------------------------
# Category 1: legacy gemini-cli npm binary
# ---------------------------------------------------------------------------
Write-Info "[1/3] Legacy @google/gemini-cli (replaced by agy)"
$npmCmd = Get-Command npm -ErrorAction SilentlyContinue
if ($npmCmd) {
    $npmGlobal = & npm ls -g --depth=0 2>$null | Out-String
    if ($npmGlobal -match '@google/gemini-cli') {
        Write-Warn "  Found global npm install: @google/gemini-cli"
        if (Confirm-Action) {
            Invoke-OrDry "npm uninstall -g @google/gemini-cli" {
                & npm uninstall -g '@google/gemini-cli'
            }
        } else {
            Write-Info "  Skipped."
        }
    } else {
        Write-Success "  No global @google/gemini-cli install found."
    }
} else {
    Write-Info "  npm not on PATH; skipping."
}

# ---------------------------------------------------------------------------
# Category 2: aider Python package
# ---------------------------------------------------------------------------
Write-Info "[2/3] Aider (sunset by SDD-007, OpenCode covers same use case)"
$foundAider = $false
if (Get-Command pipx -ErrorAction SilentlyContinue) {
    $pipxList = & pipx list 2>$null | Out-String
    if ($pipxList -match 'aider-chat|^\s*package aider') {
        $foundAider = $true
        Write-Warn "  Found pipx install: aider-chat"
        if (Confirm-Action) {
            Invoke-OrDry "pipx uninstall aider-chat" { & pipx uninstall aider-chat }
        } else {
            Write-Info "  Skipped pipx uninstall."
        }
    }
}
if (Get-Command pip -ErrorAction SilentlyContinue) {
    $pipShow = & pip show aider-chat 2>$null
    if ($LASTEXITCODE -eq 0) {
        $foundAider = $true
        Write-Warn "  Found pip install: aider-chat (user-level)"
        if (Confirm-Action) {
            Invoke-OrDry "pip uninstall -y aider-chat" { & pip uninstall -y aider-chat }
        } else {
            Write-Info "  Skipped pip uninstall."
        }
    }
}
if (-not $foundAider) { Write-Success "  No aider install found." }

# ---------------------------------------------------------------------------
# Category 3: stale config + cache directories
# ---------------------------------------------------------------------------
Write-Info "[3/3] Stale config / cache directories"
$candidates = @(
    (Join-Path $env:USERPROFILE '.config\aider'),
    (Join-Path $env:USERPROFILE '.aider'),
    (Join-Path $env:USERPROFILE '.aider.input.history'),
    (Join-Path $env:USERPROFILE '.aider.chat.history.md')
)
$cacheFiles = Get-ChildItem -LiteralPath $env:USERPROFILE -Filter '.aider.tags.cache.v*' -ErrorAction SilentlyContinue
foreach ($cf in $cacheFiles) { $candidates += $cf.FullName }

$toRemove = @($candidates | Where-Object { Test-Path -LiteralPath $_ })

if ($toRemove.Count -eq 0) {
    Write-Success "  No stale aider directories or cache files found."
} else {
    foreach ($p in $toRemove) { Write-Warn "  Found: $p" }
    if (Confirm-Action) {
        foreach ($p in $toRemove) {
            Invoke-OrDry "Remove-Item -Recurse -Force $p" {
                Remove-Item -LiteralPath $p -Recurse -Force -ErrorAction SilentlyContinue
            }
        }
    } else {
        Write-Info "  Skipped."
    }
}

Write-Success "Cleanup pass complete."
if ($DryRun) { Write-Info "Dry-run only - nothing was changed. Re-run without -DryRun to apply." }
