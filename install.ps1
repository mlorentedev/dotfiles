<#
.SYNOPSIS
    The one entrypoint on Windows: install `dotf`, then hand the machine to
    `dotf converge`.

.DESCRIPTION
    The ADR-020 bootstrap step for Windows (WIN-006): download the pinned `dotf`
    release zip from GitHub, verify its sha256 against the release checksums.txt,
    prove it runs, and install dotf.exe to ~/.local/bin - user-space, no admin.
    Everything after that is `dotf converge`, which clones the checkout when it
    is absent (PLAT-001b). The PowerShell twin of install.sh.

    Executed, every argument goes to `dotf converge`. DOTF_VERSION, DOTF_BIN_DIR
    and DOTF_RELEASE_BASE override the version, the install directory and the
    release location. The version defaults to the DOTF_VERSION line in a
    checkout's versions.conf, then to the latest published release.

    Dot-sourced (setup-windows.ps1), it only defines Install-Dotf, which bats and
    Pester drive against a file:// fixture with no network.

.EXAMPLE
    # A machine from zero - no clone, no admin:
    irm https://raw.githubusercontent.com/mlorentedev/dotfiles/main/install.ps1 | iex

.EXAMPLE
    # From a checkout, planning first:
    .\install.ps1 --plan
#>

# Map the host architecture to the goreleaser arch token, or $null when
# unsupported (the analogue of install.sh's `return 1`).
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
# DOTF_VERSION line in versions.conf (the SSOT). Mirrors install.sh.
function Get-DotfVersion {
    [CmdletBinding()]
    param([string]$Version)
    if ($Version) { return $Version }
    if ($env:DOTF_VERSION) { return $env:DOTF_VERSION }
    $versionsConf = if ($PSScriptRoot) {
        Join-Path $PSScriptRoot 'versions.conf'
    }
    if ($versionsConf -and (Test-Path $versionsConf)) {
        $match = Select-String -Path $versionsConf -Pattern '^DOTF_VERSION=(.+)$' | Select-Object -First 1
        if ($match) { return $match.Matches[0].Groups[1].Value.Trim() }
    }

    $releaseApi = if ($env:DOTF_RELEASE_API) {
        $env:DOTF_RELEASE_API
    } else {
        'https://api.github.com/repos/mlorentedev/dotfiles/releases/latest'
    }
    try {
        $release = Invoke-RestMethod -Uri $releaseApi -ErrorAction Stop
        if ($release.tag_name -match '^v?(\d+\.\d+\.\d+)$') {
            return $Matches[1]
        }
        throw "latest-release metadata has no semver tag"
    } catch {
        throw "latest-release lookup failed: $($_.Exception.Message)"
    }
}

# Place $Source at $Target, tolerating a *live* dotf. Windows locks a running
# image: it refuses to overwrite or delete dotf.exe while any dotf process is
# live, but it *does* allow renaming one. So stage the new binary beside the
# target, park the live one, then swap - the analogue of install.sh's
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
# download/verify error (no binary left in Dest). Never throws - setup wires it
# `if (-not (Install-Dotf)) { Write-Warn ... }`, the analogue of `|| log_warning`.
# On success $script:DotfBin names the binary it vetted: the one on PATH when it
# kept it, else the one it placed in Dest, which a fresh machine does not have
# on PATH yet.
function Install-Dotf {
    [CmdletBinding()]
    param(
        [string]$Version,
        [string]$Dest = $(if ($env:DOTF_BIN_DIR) { $env:DOTF_BIN_DIR } else { Join-Path $env:USERPROFILE '.local\bin' }),
        [string]$BaseUrl = $(if ($env:DOTF_RELEASE_BASE) { $env:DOTF_RELEASE_BASE } else { 'https://github.com/mlorentedev/dotfiles/releases/download' })
    )

    # Function-scoped, so dot-sourcing this script (setup-windows.ps1 does
    # `. install.ps1`) never leaks Stop/StrictMode into the caller's scope -
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
            # whatever dotf is already on PATH - including binaries built before
            # that fix, which answer on stderr. Merging both streams and regexing
            # the semver is correct for either. (StrictMode makes `@()[-1]` throw,
            # so never index blind.)
            $verRaw = (& dotf version 2>&1 | Out-String)
            # `dev` is what a source build reports. A source build on PATH is
            # deliberate (a dev box, or CI building the PR under test) and the
            # release installer must not replace it; remove it to converge.
            # Parity with install.sh.
            $current = if ($verRaw -match '(\d+\.\d+\.\d+|dev)') { $Matches[1] } else { '' }
            if ($current -eq 'dev') {
                Write-Host "dotf is a source build (dev); leaving it in place (remove it to converge to the $Version release)"
                $script:DotfBin = (Get-Command dotf).Source
                return $true
            }
            if ($current -eq $Version) {
                Write-Host "dotf $Version already installed; skipping"
                $script:DotfBin = (Get-Command dotf).Source
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
        # Exec probe before anything is placed: a binary that verifies but
        # cannot run here must not replace a working one.
        $null = & $exe version 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "the downloaded dotf $Version does not run on this machine"
        }
        New-Item -ItemType Directory -Force -Path $Dest | Out-Null
        $target = Join-Path $Dest 'dotf.exe'
        Set-DotfBinary -Source $exe -Target $target
        Write-Host "dotf $Version installed to $target"
        $script:DotfBin = $target
        return $true
    } catch {
        Write-Warning "install: $($_.Exception.Message)"
        return $false
    } finally {
        if ($work) { Remove-Item -Recurse -Force -Path $work -ErrorAction SilentlyContinue }
    }
}

# Standalone run-guard: install and converge when EXECUTED, not when
# dot-sourced. $MyInvocation.InvocationName is '.' under dot-sourcing
# (setup-windows.ps1 does `. install.ps1`, which must only define the
# functions).
if ($MyInvocation.InvocationName -ne '.') {
    if (-not (Install-Dotf)) {
        throw 'install.ps1: dotf could not be installed, so nothing was converged'
    }
    & $script:DotfBin converge @args
    # A file run reports converge's status. Under `irm | iex` there is no
    # script to leave, and `exit` would close the caller's shell.
    if ($PSCommandPath) { exit $LASTEXITCODE }
}
