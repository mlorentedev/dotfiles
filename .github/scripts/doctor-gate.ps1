# doctor-gate.ps1 -- the post-setup dotf doctor gate for the test-windows job
# (TEST-003, #1298).
#
# Runs `dotf doctor` the way a fresh terminal would and fails the job unless
# every [FAIL] line matches an entry in the known-failures list AND every entry
# matches at least one line. The second half is what keeps the list honest: an
# entry whose failure no longer occurs is reported as STALE and fails the gate,
# so the list can only shrink as tickets close, never rot into a blanket allow.
#
# Known failures are runner-only conditions with an owning ticket (see the
# list's comments). A real-box defect is never added there; it is fixed or
# ticketed, and the gate stays red until then.
#
# ASCII-only by repo convention (pattern-powershell-ascii-only).

# Test-DoctorGate: pure classification, unit-tested by tests/doctor-gate.Tests.ps1.
# Returns an object with Failures (the [FAIL] lines), Unexpected (failures no
# pattern matched) and Stale (patterns that matched no failure).
function Test-DoctorGate {
    param(
        # AllowEmptyString on top of AllowEmptyCollection: doctor output has blank
        # lines between sections, and a Mandatory [string[]] rejects an empty
        # ELEMENT ("Cannot bind argument ... because it is an empty string") --
        # the gate died on that with the real output while every fixture passed.
        [Parameter(Mandatory)][AllowEmptyCollection()][AllowEmptyString()][AllowNull()][string[]]$Lines,
        [Parameter(Mandatory)][AllowEmptyCollection()][AllowEmptyString()][AllowNull()][string[]]$Patterns
    )
    # A list with no entries reaches here as $null, which [string[]] binds as
    # one null element: it matched nothing and was reported STALE, failing the
    # gate the moment its last row was retired (WIN-013). Zero rows is the
    # steady state of a list that only shrinks, so drop empties up front.
    $Patterns = @($Patterns | Where-Object { $_ })
    $failures = @($Lines | Where-Object { $_ -match '^\s*\[FAIL\]' })
    $unexpected = @($failures | Where-Object {
        $line = $_
        -not ($Patterns | Where-Object { $line -match $_ })
    })
    $stale = @($Patterns | Where-Object {
        $pattern = $_
        -not ($failures | Where-Object { $_ -match $pattern })
    })
    [pscustomobject]@{ Failures = $failures; Unexpected = $unexpected; Stale = $stale }
}

# Read-KnownFailures: one regex per non-blank, non-comment line.
function Read-KnownFailures {
    param([Parameter(Mandatory)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) { return @() }
    # `return ,$entries` hands the caller a real (possibly empty) array; a bare
    # @(...) unrolls to nothing and arrives as $null (WIN-013 emptied the list).
    $entries = @(Get-Content -LiteralPath $Path | ForEach-Object { $_.Trim() } | Where-Object { $_ -and -not $_.StartsWith('#') })
    return ,$entries
}

if ($MyInvocation.InvocationName -ne '.') {
    $ErrorActionPreference = 'Stop'
    $known = Join-Path $PSScriptRoot 'doctor-gate-known-failures.txt'

    # A fresh terminal's PATH: setup appends the winget Links dir and npm's
    # global prefix to the User PATH, but a later workflow step inherits the
    # runner's original PATH plus GITHUB_PATH only. Without this, every tool
    # setup just installed reads "not in PATH" (measured on #1308's first run).
    $env:PATH = [Environment]::GetEnvironmentVariable('PATH', 'Machine') + ';' +
        [Environment]::GetEnvironmentVariable('PATH', 'User') + ';' + $env:PATH

    # Name the PATH doctor ran with: a FAIL that reads "git missing" is a
    # statement about this step's environment before it is one about the box.
    $entries = @($env:PATH -split ';' | Where-Object { $_ })
    Write-Host ("doctor gate: PATH has {0} entries, {1} chars" -f $entries.Count, $env:PATH.Length)
    $entries | ForEach-Object { Write-Host "doctor gate:   $_" }
    foreach ($probe in 'dotf', 'git', 'bash', 'node', 'bw') {
        $cmd = Get-Command $probe -ErrorAction SilentlyContinue
        Write-Host ("doctor gate: {0} -> {1}" -f $probe, ($(if ($cmd) { $cmd.Source } else { 'NOT FOUND' })))
    }

    # The runner has no Bitwarden identity and must never hold one that resolves
    # real secrets, so [Bitwarden reach] reports "unauthenticated" by design.
    # Declare that to doctor (TEST-005, #1313) rather than allow-listing the FAIL:
    # the known-failures list is for runner-only conditions with a fix pending,
    # and a runner that will never log in is not a pending fix. doctor reads the
    # flag in that one branch only; every other reach tier still runs.
    $env:DOTFILES_DOCTOR_NO_IDENTITY = '1'
    Write-Host "doctor gate: DOTFILES_DOCTOR_NO_IDENTITY=1 (no Bitwarden identity on this runner, declared)"

    $output = & dotf doctor 2>&1 | Out-String
    Write-Host $output

    $result = Test-DoctorGate -Lines ($output -split "`r?`n") -Patterns (Read-KnownFailures -Path $known)
    if ($result.Unexpected.Count -gt 0) {
        Write-Host "UNEXPECTED doctor failure(s) -- not in $known (fix, or ticket and list with the ticket):"
        $result.Unexpected | ForEach-Object { Write-Host "  $_" }
    }
    if ($result.Stale.Count -gt 0) {
        Write-Host "STALE known-failure entr(y/ies) -- matched no [FAIL] line; remove them so the list cannot rot:"
        $result.Stale | ForEach-Object { Write-Host "  $_" }
    }
    if ($result.Unexpected.Count -gt 0 -or $result.Stale.Count -gt 0) { exit 1 }
    Write-Host ("doctor gate: {0} known runner-only FAIL(s), 0 unexpected, 0 stale" -f $result.Failures.Count)
    exit 0
}
