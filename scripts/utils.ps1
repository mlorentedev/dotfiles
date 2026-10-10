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

# Console I/O is UTF-8 for this process (WIN-009/#1290). PowerShell decodes
# captured native output with [Console]::OutputEncoding, which defaults to the
# system OEM code page (437 on the work box), so a `dotf` message carrying an
# em dash arrived as three OEM glyphs -- and every block that PARSES captured
# output was reading corrupted bytes, not merely printing them. Process state,
# not logic: one of the few things that legitimately stays in shell. Guarded:
# a host without a console (a transcript-only CI step) throws on the setter.
try {
    $script:Utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [Console]::OutputEncoding = $script:Utf8NoBom
    [Console]::InputEncoding = $script:Utf8NoBom
    $global:OutputEncoding = $script:Utf8NoBom
} catch {
    Write-Verbose "console encoding not set: $_"
}

# Write-Utf8LfFile: write CONTENT to PATH as UTF-8 without BOM and with LF line
# endings, whatever the platform newline is. Set-Content joins with the platform
# newline (CRLF here) and Windows PowerShell 5.1 adds a BOM, so a deployed .md
# drifted from its LF repo source on every run and the doctor's drift check
# could never clear (WIN-008/#1289). .gitattributes declares *.md eol=lf; the
# deployed copy honours the same contract.
function Write-Utf8LfFile {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][AllowEmptyString()][string]$Content
    )
    $text = $Content -replace "`r`n", "`n"
    if (-not $text.EndsWith("`n")) { $text += "`n" }
    $dir = Split-Path -Path $Path -Parent
    if ($dir -and -not (Test-Path -LiteralPath $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
    [System.IO.File]::WriteAllText($Path, $text, (New-Object System.Text.UTF8Encoding($false)))
}

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
# Sync-SessionPath: rebuild this process's PATH as Machine + User + the
# current process PATH, first occurrence wins (case-insensitive, a trailing
# backslash does not make a different entry). Setup calls it after every block
# that installs into the registry PATH so the new tools resolve in this same
# run; registry first so a fresh winget install beats a stale process entry.
# The process PATH is kept because an entry that exists only here (a CI
# GITHUB_PATH addition, a build dir put on PATH for the run) was silently
# dropped by the registry-only rebuild this replaces (TEST-003/#1298), and the
# merge is deduplicated because the plain concatenation that first fixed that
# doubled PATH on every call -- 153 entries, 72 of them duplicates, measured on
# the CI runner (#1308).
function Sync-SessionPath {
    $seen = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    $merged = foreach ($entry in @(
            [Environment]::GetEnvironmentVariable('PATH', 'Machine'),
            [Environment]::GetEnvironmentVariable('PATH', 'User'),
            $env:PATH) -split ';') {
        $e = "$entry".Trim()
        if ($e -and $seen.Add($e.TrimEnd('\'))) { $e }
    }
    $env:PATH = @($merged) -join ';'
}

# Installs one winget package and reports what actually happened (#2157). A
# native command reports failure through its exit code, never an exception, so
# setup's old loop -- winget's output discarded, a catch that could not fire --
# printed "installed" after a failed install, and doctor found the tool missing
# later with nothing in the log to say why (2 of 12 test-windows runs on
# 2026-10-08). Success is judged by the command resolving after a PATH refresh,
# not by what winget says; one retry covers the transient failures measured on
# the runner. Returns the outcome; the caller words it.
function Install-WingetTool {
    param(
        [Parameter(Mandatory)][string]$Cmd,
        [Parameter(Mandatory)][string]$Id,
        [ValidateRange(1, 5)][int]$Attempts = 2
    )
    $output = @()
    $exitCode = $null
    for ($i = 1; $i -le $Attempts; $i++) {
        $output = @(& winget install $Id --accept-package-agreements --accept-source-agreements 2>&1)
        $exitCode = $LASTEXITCODE
        Sync-SessionPath
        if (Get-Command $Cmd -ErrorAction SilentlyContinue) {
            return [pscustomobject]@{ Installed = $true; Attempts = $i; ExitCode = $exitCode; Detail = '' }
        }
    }
    $detail = @($output | ForEach-Object { "$_".Trim() } | Where-Object { $_ } | Select-Object -Last 3) -join ' | '
    [pscustomobject]@{ Installed = $false; Attempts = $Attempts; ExitCode = $exitCode; Detail = $detail }
}

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
    $dstDir = Split-Path -Path $Destination -Parent
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

# Test-FileDrift was removed by OPS-043 (#1337). It was written as parity with
# bash check_deployed for healthcheck.ps1, and CLI-018 retired that script into
# `dotf doctor` without removing the helper -- leaving a fully implemented
# function with zero call sites anywhere in the repo. Its behaviour now lives in
# doctor's `Deploy-dir<->$HOME drift` section, which is cross-compiled and so
# needs no PowerShell twin at all.

# Get-ClaudeProjectKey and Get-ClaudeProjectKeyEncoded were removed with the
# setup's auto-memory junction sweep (#1843 B13), their last caller. The key is
# computed by memlink.ClaudeProjectKey in Go, and `dotf mem project-key` prints
# it for any script that needs one.
