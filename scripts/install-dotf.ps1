<#
.SYNOPSIS
    Fetch, verify, and install the `dotf` CLI release binary on Windows.

.DESCRIPTION
    The ADR-020 bootstrap step for Windows (WIN-006): download the pinned `dotf`
    release zip from GitHub, verify its sha256 against the release checksums.txt,
    and install dotf.exe to ~/.local/bin — user-space, no admin. The PowerShell
    twin of scripts/install-dotf.sh.

    Dot-sourced by setup-windows.ps1 (which then calls Install-Dotf); also runnable
    standalone to (re)install or upgrade. DOTF_VERSION is the pinned version
    (versions.conf SSOT); the function takes version/dest/base as parameters so
    bats can drive it against a file:// fixture with no network.

.PARAMETER Version
    dotf version to install. Defaults to $env:DOTF_VERSION, then the DOTF_VERSION
    line in versions.conf.

.EXAMPLE
    # One-line bootstrap — no clone, no admin:
    irm https://raw.githubusercontent.com/mlorentedev/dotfiles/main/scripts/install-dotf.ps1 | iex

.EXAMPLE
    # From a checkout:
    . .\scripts\install-dotf.ps1 ; Install-Dotf
#>
[CmdletBinding()]
param(
    [string]$Version,
    [string]$Dest = (Join-Path $env:USERPROFILE '.local\bin'),
    [string]$BaseUrl = 'https://github.com/mlorentedev/dotfiles/releases/download'
)

# Map the host architecture to the goreleaser arch token, or $null when
# unsupported (the analogue of install-dotf.sh's `return 1`).
function Get-DotfArch {
    [CmdletBinding()]
    param([string]$Arch = $env:PROCESSOR_ARCHITECTURE)
    switch ($Arch) {
        'AMD64' { 'amd64'; break }
        'ARM64' { 'arm64'; break }
        default { $null }
    }
}

# Resolve the pinned version: explicit arg, then $env:DOTF_VERSION, then the
# DOTF_VERSION line in versions.conf (the SSOT). Mirrors install-dotf.sh.
function Get-DotfVersion {
    [CmdletBinding()]
    param([string]$Version)
    if ($Version) { return $Version }
    if ($env:DOTF_VERSION) { return $env:DOTF_VERSION }
    $versionsConf = Join-Path $PSScriptRoot '..\versions.conf'
    if (Test-Path $versionsConf) {
        $match = Select-String -Path $versionsConf -Pattern '^DOTF_VERSION=(.+)$' | Select-Object -First 1
        if ($match) { return $match.Matches[0].Groups[1].Value.Trim() }
    }
    return $null
}

# Place $Source at $Target, tolerating a *live* dotf. Windows locks a running
# image: it refuses to overwrite or delete dotf.exe while any dotf process is
# live, but it *does* allow renaming one. So stage the new binary beside the
# target, park the live one, then swap — the analogue of install-dotf.sh's
# atomic mv (BUG-037). Throws on failure, having restored the previous binary.
function Set-DotfBinary {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$Source,
        [Parameter(Mandatory)][string]$Target
    )

    # Self-contained: callers other than Install-Dotf (Pester) must get the same
    # terminating-error behaviour, or the rollback below would never fire.
    $ErrorActionPreference = 'Stop'

    $staged = "$Target.new"
    $parked = "$Target.old"

    # A park left behind by an earlier upgrade (its image was still locked then)
    # would otherwise block this one.
    Remove-Item -LiteralPath $parked -Force -ErrorAction SilentlyContinue
    Copy-Item -LiteralPath $Source -Destination $staged -Force

    if (Test-Path -LiteralPath $Target) {
        Move-Item -LiteralPath $Target -Destination $parked -Force
    }
    try {
        Move-Item -LiteralPath $staged -Destination $Target -Force
    } catch {
        # Never leave the user without a dotf: put the previous one back.
        if (Test-Path -LiteralPath $parked) {
            Move-Item -LiteralPath $parked -Destination $Target -Force
        }
        Remove-Item -LiteralPath $staged -Force -ErrorAction SilentlyContinue
        throw
    }

    # Best effort: a park still locked by the outgoing process is cleared by the
    # next upgrade, so a failure here must not fail the install.
    Remove-Item -LiteralPath $parked -Force -ErrorAction SilentlyContinue
}

# Idempotently install the pinned dotf release. No-op when the pinned version is
# already on PATH; converges on drift. Returns $true on success, $false on any
# download/verify error (no binary left in Dest). Never throws — setup wires it
# `if (-not (Install-Dotf)) { Write-Warn ... }`, the analogue of `|| log_warning`.
function Install-Dotf {
    [CmdletBinding()]
    param(
        [string]$Version,
        [string]$Dest = (Join-Path $env:USERPROFILE '.local\bin'),
        [string]$BaseUrl = 'https://github.com/mlorentedev/dotfiles/releases/download'
    )

    # Function-scoped, so dot-sourcing this script (setup-windows.ps1 does
    # `. install-dotf.ps1`) never leaks Stop/StrictMode into the caller's scope —
    # only this function's body runs strict. Also required for the try/catch below
    # to catch non-terminating errors regardless of the caller's preference.
    Set-StrictMode -Version Latest
    $ErrorActionPreference = 'Stop'

    $work = $null
    try {
        $Version = Get-DotfVersion -Version $Version
        if (-not $Version) {
            throw 'no version given (set DOTF_VERSION in versions.conf)'
        }
        $arch = Get-DotfArch
        if (-not $arch) {
            throw "unsupported arch: $env:PROCESSOR_ARCHITECTURE"
        }

        # Idempotence: skip when the pinned version is already on PATH.
        if (Get-Command dotf -ErrorAction SilentlyContinue) {
            # The stream merge (2>&1) is kept deliberately: BUG-070 (#915) fixed
            # `dotf version` to write to stdout, but this installer runs against
            # whatever dotf is already on PATH — including binaries built before
            # that fix, which answer on stderr. Merging both streams and regexing
            # the semver is correct for either. (StrictMode makes `@()[-1]` throw,
            # so never index blind.)
            $verRaw = (& dotf version 2>&1 | Out-String)
            # `dev` is what a source build reports; CI pins DOTF_VERSION=dev so the
            # binary built from the PR under test is kept (parity with install-dotf.sh).
            $current = if ($verRaw -match '(\d+\.\d+\.\d+|dev)') { $Matches[1] } else { '' }
            if ($current -eq $Version) {
                Write-Host "dotf $Version already installed; skipping"
                return $true
            }
            if ($current) { Write-Host "dotf $current drifted from pinned $Version; converging" }
        }

        $artifact = "dotf_${Version}_windows_${arch}.zip"
        $work = Join-Path ([System.IO.Path]::GetTempPath()) ('dotf-' + [System.IO.Path]::GetRandomFileName())
        New-Item -ItemType Directory -Force -Path $work | Out-Null

        Invoke-WebRequest -Uri "$BaseUrl/v$Version/$artifact" -OutFile (Join-Path $work $artifact) -UseBasicParsing
        Invoke-WebRequest -Uri "$BaseUrl/v$Version/checksums.txt" -OutFile (Join-Path $work 'checksums.txt') -UseBasicParsing

        $entry = Select-String -Path (Join-Path $work 'checksums.txt') -Pattern ([regex]::Escape($artifact)) | Select-Object -First 1
        if (-not $entry) {
            throw "$artifact not listed in checksums.txt"
        }
        $expected = (($entry.Line -split '\s+') | Where-Object { $_ })[0].ToLower()
        $actual = (Get-FileHash -Path (Join-Path $work $artifact) -Algorithm SHA256).Hash.ToLower()
        if ($expected -ne $actual) {
            throw "checksum mismatch for $artifact (want $expected, got $actual)"
        }

        Expand-Archive -Path (Join-Path $work $artifact) -DestinationPath $work -Force
        $exe = Join-Path $work 'dotf.exe'
        if (-not (Test-Path $exe)) {
            throw "dotf.exe not found in $artifact"
        }
        New-Item -ItemType Directory -Force -Path $Dest | Out-Null
        $target = Join-Path $Dest 'dotf.exe'
        Set-DotfBinary -Source $exe -Target $target
        Write-Host "dotf $Version installed to $target"
        return $true
    } catch {
        Write-Warning "install-dotf: $($_.Exception.Message)"
        return $false
    } finally {
        if ($work) { Remove-Item -Recurse -Force -Path $work -ErrorAction SilentlyContinue }
    }
}

# Standalone run-guard: install when EXECUTED, not when dot-sourced.
# $MyInvocation.InvocationName is '.' under dot-sourcing (setup-windows.ps1 does
# `. install-dotf.ps1`, which must only define the functions).
if ($MyInvocation.InvocationName -ne '.') {
    $null = Install-Dotf -Version $Version -Dest $Dest -BaseUrl $BaseUrl
}
