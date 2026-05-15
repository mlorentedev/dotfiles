<#
.SYNOPSIS
    Verify structural env vars, PATH entries and binaries against env-contract.json.

.DESCRIPTION
    Reads $DotfilesDir/env-contract.json (or repo-local fallback), checks env
    vars + required PATH entries + binaries against the contract, and reports
    drift. With -Fix, applies known-safe defaults to the current session and
    invokes claude-mem-heal.ps1.

    Mirrors scripts/doctor.sh (Linux).

.PARAMETER Fix
    Apply defaults to current session, run heal scripts. Without this, runs in
    check-only mode: reports drift and exits 1 if any required item missing.

.PARAMETER VerboseOutput
    Show every check, not just warnings/failures.

.EXAMPLE
    pwsh -NoProfile -File doctor.ps1
    pwsh -NoProfile -File doctor.ps1 -Fix -VerboseOutput
#>

[CmdletBinding()]
param(
    [switch]$Fix,
    [switch]$VerboseOutput
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'

$script:Mode = if ($Fix) { 'fix' } else { 'check' }
$script:Verbose = $VerboseOutput.IsPresent
$script:Issues = 0
$script:Applied = 0
$script:ExitCode = 0

# Per-section counters so the section summary line shows even when every
# item is silent ([ok] in non-verbose) -- mirrors doctor.sh.
$script:SecOk = 0; $script:SecWarn = 0; $script:SecFail = 0; $script:SecInfo = 0
function Reset-Section { $script:SecOk = 0; $script:SecWarn = 0; $script:SecFail = 0; $script:SecInfo = 0 }
function Write-SectionSummary {
    $total = $script:SecOk + $script:SecWarn + $script:SecFail + $script:SecInfo
    if ($total -eq 0) { return }
    if ($script:SecWarn -eq 0 -and $script:SecFail -eq 0) {
        if ($script:SecInfo -gt 0) {
            Write-Host "  ($total checks: $($script:SecOk) ok, $($script:SecInfo) info)"
        } else {
            Write-Host "  ($total checks, all ok)"
        }
    } else {
        Write-Host "  ($total total: $($script:SecOk) ok, $($script:SecWarn) warn, $($script:SecFail) fail)"
    }
}
function Write-Pass { param([string]$Msg) $script:SecOk++; if ($script:Verbose) { Write-Host "  [ok]   $Msg" } }
function Write-Warn { param([string]$Msg) $script:SecWarn++; Write-Host "  [warn] $Msg" -ForegroundColor Yellow; $script:Issues++ }
function Write-Fail { param([string]$Msg) $script:SecFail++; Write-Host "  [fail] $Msg" -ForegroundColor Red;    $script:Issues++; $script:ExitCode = 1 }
function Write-Fix  { param([string]$Msg) Write-Host "  [fix]  $Msg" -ForegroundColor Cyan;   $script:Applied++ }
function Write-Inf  { param([string]$Msg) $script:SecInfo++; Write-Host "  [info] $Msg" }

function Expand-ContractPath {
    param([string]$Value)
    # Expand $env:USERPROFILE and ${HOME} style; leave other refs alone
    $expanded = $Value -replace '\$env:USERPROFILE', $env:USERPROFILE
    $expanded = $expanded -replace '\$\{USERPROFILE\}', $env:USERPROFILE
    $expanded
}

# Locate contract
$dotfilesDir = if ($env:DOTFILES_DIR) { $env:DOTFILES_DIR } else { Join-Path $env:USERPROFILE '.dotfiles' }
$candidates = @(
    (Join-Path $dotfilesDir 'env-contract.json'),
    (Join-Path $PSScriptRoot '..\env-contract.json')
)
$contract = $null
foreach ($c in $candidates) {
    if (Test-Path $c) { $contract = (Resolve-Path $c).Path; break }
}
if (-not $contract) {
    Write-Host "doctor: env-contract.json not found (looked in DOTFILES_DIR and repo root)" -ForegroundColor Red
    exit 2
}

$data = Get-Content $contract -Raw | ConvertFrom-Json

Write-Host "doctor [$script:Mode] using $contract"

# 1. Env vars
Write-Host ""
Write-Host "Environment variables:"
Reset-Section
foreach ($v in $data.env_vars) {
    # Skip vars scoped to the other OS
    if ($v.PSObject.Properties.Match('required_on').Count -gt 0 -and $v.required_on -eq 'linux') {
        Write-Pass "$($v.name) (linux-scoped, skipped on Windows)"
        continue
    }
    $current = [Environment]::GetEnvironmentVariable($v.name)
    $defaultRaw = if ($v.PSObject.Properties.Match('default').Count -gt 0 -and $v.default -and $v.default.PSObject.Properties.Match('windows').Count -gt 0) { $v.default.windows } else { $null }
    $isRequired = ($v.PSObject.Properties.Match('required').Count -gt 0 -and $v.required) -or `
                  ($v.PSObject.Properties.Match('required_on').Count -gt 0 -and $v.required_on -eq 'windows')

    if ([string]::IsNullOrEmpty($current)) {
        if ($defaultRaw) {
            $expanded = Expand-ContractPath $defaultRaw
            if ($script:Mode -eq 'fix') {
                [Environment]::SetEnvironmentVariable($v.name, $expanded, 'Process')
                Write-Fix "$($v.name): applied default '$expanded' to current session (add to profile to persist)"
                $current = $expanded
            } else {
                if ($isRequired) {
                    Write-Warn "$($v.name) unset (required); default available: '$expanded' -- run with -Fix or set in profile"
                } else {
                    Write-Warn "$($v.name) unset; default '$expanded' would be applied with -Fix"
                }
                $current = $expanded
            }
        } else {
            if ($isRequired) {
                Write-Fail "$($v.name) unset and no default available (required)"
                continue
            } else {
                Write-Pass "$($v.name) unset (optional, no default)"
                continue
            }
        }
    }
    # Validation
    if ($v.PSObject.Properties.Match('validation').Count -gt 0 -and $v.validation -eq 'path_exists') {
        if (Test-Path $current -PathType Container) {
            Write-Pass "$($v.name)=$current (path exists)"
        } else {
            Write-Fail "$($v.name)=$current (path does not exist)"
        }
    } else {
        Write-Pass "$($v.name)=$current"
    }
}

Write-SectionSummary

# 2. PATH entries
Write-Host ""
Write-Host "PATH entries:"
Reset-Section
$pathEntries = $env:PATH -split ';'
foreach ($entry in $data.required_path_entries.windows) {
    $expanded = Expand-ContractPath $entry
    if ($pathEntries -contains $expanded) {
        Write-Pass "$expanded in PATH"
    } else {
        Write-Warn "$expanded not in PATH -- check shell profile"
    }
}

Write-SectionSummary

# 3. Required binaries (with optional version pinning)
Write-Host ""
Write-Host "Required binaries:"
Reset-Section
foreach ($b in $data.required_binaries) {
    if (-not (Get-Command $b.name -ErrorAction SilentlyContinue)) {
        Write-Fail "$($b.name) missing"
        continue
    }
    $minVersion = if ($b.PSObject.Properties.Match('min_version').Count -gt 0) { $b.min_version } else { '' }
    $pattern = if ($b.PSObject.Properties.Match('version_pattern').Count -gt 0) { $b.version_pattern } else { '' }
    if ([string]::IsNullOrEmpty($minVersion)) {
        Write-Pass "$($b.name) on PATH"
        continue
    }
    # Run `<name> --version` and apply pattern. Some tools write version to
    # stderr, hence the 2>&1 merge.
    $raw = (& $b.name --version 2>&1 | Select-Object -First 1) | Out-String
    $raw = $raw.Trim()
    if ($raw -match $pattern) {
        $actual = $Matches[1]
        try {
            $cmp = [Version]::Parse($actual).CompareTo([Version]::Parse($minVersion))
            if ($cmp -ge 0) {
                Write-Pass "$($b.name) $actual (>= $minVersion)"
            } else {
                Write-Fail "$($b.name) $actual is older than minimum $minVersion"
            }
        } catch {
            Write-Warn "$($b.name) version compare failed for '$actual' vs '$minVersion': $_"
        }
    } else {
        Write-Warn "$($b.name) on PATH but version unparseable: '$raw' (pattern: $pattern)"
    }
}

Write-SectionSummary

# 4. Optional binaries
Write-Host ""
Write-Host "Optional binaries:"
Reset-Section
foreach ($b in $data.optional_binaries) {
    if (Get-Command $b.name -ErrorAction SilentlyContinue) {
        Write-Pass "$($b.name) on PATH ($($b.purpose))"
    } else {
        Write-Inf "$($b.name) missing -- $($b.purpose)"
    }
}
Write-SectionSummary

# 5. Fix mode: invoke known heals
if ($script:Mode -eq 'fix') {
    Write-Host ""
    Write-Host "Heal scripts:"
    $heal = Join-Path $PSScriptRoot 'claude-mem-heal.ps1'
    if (Test-Path $heal) {
        $healOut = & pwsh -NoProfile -File $heal 2>&1 | Out-String
        $healOut = $healOut.Trim()
        if ($healOut) {
            Write-Host "  [fix]  claude-mem-heal:" -ForegroundColor Cyan
            $healOut -split "`n" | ForEach-Object { Write-Host "         $_" }
            $script:Applied++
        } else {
            Write-Pass "claude-mem-heal: nothing to heal"
        }
    } else {
        Write-Inf "claude-mem-heal.ps1 not deployed (skipping)"
    }
}

Write-Host ""
Write-Host "Summary: $script:Issues issue(s), $script:Applied action(s) applied"
exit $script:ExitCode
