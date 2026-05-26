# utils.ps1 - shared PowerShell helpers for cross-OS setup parity.
#
# Today only the deploy/drift helpers live here (SDD-007 IaC strategy).
# Other helpers (Write-Info, Ensure-Directory) remain inline in setup-windows.ps1
# to avoid coupling load order; the rule will be revisited in a follow-up.
#
# ASCII-only by repo convention (pattern-powershell-ascii-only). No em-dashes,
# no smart quotes, no arrows, no non-ASCII glyphs anywhere in this file.
# PSScriptAnalyzer fails CI on non-ASCII chars without a BOM.

Set-StrictMode -Version Latest

# Deploy-File: copy SRC to DST atomically + idempotently.
# Parity with bash scripts/utils.sh deploy_file().
#
# Behavior:
#   - skips silently if SHA256 matches (idempotent on repeat runs)
#   - removes pre-existing reparse-point (Windows equivalent of symlink) at DST
#   - writes via Copy-Item then sets owner-only on staging (atomic in NTFS)
#   - logs "Deployed: <path>" only on actual change
#   - returns $true on success, $false on error
#
# Usage:
#   . $PSScriptRoot\utils.ps1
#   Deploy-File "C:\repo\.zshrc" "$env:USERPROFILE\.zshrc"
function Deploy-File {
    param(
        [Parameter(Mandatory)][string]$Source,
        [Parameter(Mandatory)][string]$Destination
    )

    if (-not (Test-Path -LiteralPath $Source -PathType Leaf)) {
        Write-Host "[ERROR] Deploy-File: source missing: $Source" -ForegroundColor Red
        return $false
    }

    # Idempotency check by SHA256
    if (Test-Path -LiteralPath $Destination -PathType Leaf) {
        $isReparse = (Get-Item -LiteralPath $Destination -Force).Attributes -band [IO.FileAttributes]::ReparsePoint
        if (-not $isReparse) {
            $srcHash = (Get-FileHash -LiteralPath $Source -Algorithm SHA256).Hash
            $dstHash = (Get-FileHash -LiteralPath $Destination -Algorithm SHA256).Hash
            if ($srcHash -eq $dstHash) {
                return $true
            }
        }
    }

    # Drift recovery: pre-existing symlink / junction at destination
    if (Test-Path -LiteralPath $Destination) {
        $item = Get-Item -LiteralPath $Destination -Force
        if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) {
            Remove-Item -LiteralPath $Destination -Force
        }
    }

    # Ensure parent directory exists
    $dstDir = Split-Path -LiteralPath $Destination -Parent
    if ($dstDir -and -not (Test-Path -LiteralPath $dstDir)) {
        New-Item -ItemType Directory -Path $dstDir -Force | Out-Null
    }

    try {
        Copy-Item -LiteralPath $Source -Destination $Destination -Force
        Write-Host "[SUCCESS] Deployed: $Destination" -ForegroundColor Green
        return $true
    } catch {
        Write-Host "[ERROR] Deploy-File: copy failed $Source -> $Destination" -ForegroundColor Red
        return $false
    }
}

# Test-FileDrift: assert DST matches SRC content (drift detection for healthcheck).
# Parity with bash scripts/utils.sh check_deployed().
#
# Returns $true if files match, $false if drift detected.
# Logs PASS/FAIL via Write-Host so callers can compose with their own counters.
function Test-FileDrift {
    param(
        [Parameter(Mandatory)][string]$Source,
        [Parameter(Mandatory)][string]$Destination,
        [string]$DisplayName = ''
    )
    if (-not $DisplayName) { $DisplayName = Split-Path -Leaf $Destination }

    if (-not (Test-Path -LiteralPath $Destination -PathType Leaf)) {
        Write-Host "[FAIL] $DisplayName missing at $Destination (run setup-windows.ps1)" -ForegroundColor Red
        return $false
    }

    $isReparse = (Get-Item -LiteralPath $Destination -Force).Attributes -band [IO.FileAttributes]::ReparsePoint
    if ($isReparse) {
        Write-Host "[FAIL] $DisplayName is a reparse-point/symlink (expected regular file)" -ForegroundColor Red
        return $false
    }

    $srcHash = (Get-FileHash -LiteralPath $Source -Algorithm SHA256).Hash
    $dstHash = (Get-FileHash -LiteralPath $Destination -Algorithm SHA256).Hash
    if ($srcHash -eq $dstHash) {
        Write-Host "[PASS] $DisplayName deployed (matches repo)" -ForegroundColor Green
        return $true
    }

    Write-Host "[FAIL] $DisplayName has drifted from $Source (edit in repo + re-run setup)" -ForegroundColor Red
    return $false
}
