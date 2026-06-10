# init-spec.ps1
# Purpose: Mechanical scaffold of a per-feature spec folder per pattern-spec-driven-development.
#          Work-gated: requires an OPEN GitHub issue (bitacora) before scaffolding (ADR-018).
#
# Usage:
#   .\init-spec.ps1 <feature-id> -Issue <number> [-ForceNoGate]
#
# Mechanical only. For Socratic proposal filling (Q1-Q6), use /spec fill in an agent.
# See: $env:VAULT_PATH\00_meta\skills\spec\SKILL.md for the full workflow.

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$FeatureId,

    [int]$Issue,

    [switch]$ForceNoGate,

    # Deprecated alias of -ForceNoGate (pre-ADR-018 name).
    [switch]$ForceNoVault,

    [switch]$Help
)

$ErrorActionPreference = 'Stop'

if ($Help) {
    @"
Usage: init-spec.ps1 <feature-id> -Issue <number> [-ForceNoGate]

  <feature-id>      e.g. AI-001-ollama-public or 2026-05-13-foo
  -Issue <number>   GitHub issue that gates this work (must exist and be OPEN)
  -ForceNoGate      skip the work-gate check (NOT RECOMMENDED -- gate is the SSOT)
  -ForceNoVault     deprecated alias of -ForceNoGate
"@ | Write-Host
    exit 0
}

if ($ForceNoVault) {
    Write-Warning '-ForceNoVault is deprecated; use -ForceNoGate (ADR-018).'
    $ForceNoGate = $true
}

# --- Validate id ---
# The optional single letter after the number admits the sub-id convention
# (SDD-012b, WIN-002a) that check-backlog-integrity treats as a distinct ticket.
$Pattern = '^([A-Z]+-[0-9]+[a-z]?(-[a-z0-9-]+)?|[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9-]+)$'
if ($FeatureId -notmatch $Pattern) {
    Write-Error "Invalid feature-id: $FeatureId. Expected: TICKET-NNN[letter][-slug] or YYYY-MM-DD-slug."
    exit 1
}

# --- Resolve paths ---
$VaultPath = if ($env:VAULT_PATH) { $env:VAULT_PATH } else { Join-Path $HOME 'Projects\knowledge' }
$TemplatesDir = Join-Path $VaultPath '00_meta\templates'

if (-not (Test-Path $TemplatesDir)) {
    Write-Error "Vault templates not found at: $TemplatesDir. Set `$env:VAULT_PATH or clone the vault."
    exit 2
}

$RepoRoot = $null
try {
    $RepoRoot = (git rev-parse --show-toplevel 2>$null) | ForEach-Object { $_.Trim() }
} catch { }
if (-not $RepoRoot) {
    Write-Error 'Not in a git repo. cd into a repo first.'
    exit 1
}

$SpecDir = Join-Path $RepoRoot "specs\$FeatureId"

# --- No clobber ---
if (Test-Path $SpecDir) {
    Write-Error "Already exists: $SpecDir"
    exit 1
}
$ArchivedPath = Join-Path $RepoRoot "specs\archive\$FeatureId"
if (Test-Path $ArchivedPath) {
    Write-Warning "$FeatureId exists in specs\archive\. Possibly reviving."
}

# --- Work-gate check (ADR-018: an OPEN GitHub issue, not a vault entry) ---
$IssueTitle = $null

if (-not $ForceNoGate) {
    if (-not $Issue) {
        @"
[ERROR] No work-gate given. Pass -Issue <number>.

Per ADR-018 every spec is downstream of an OPEN GitHub issue on the bitacora
Project -- that issue is the work-gate (the vault no longer holds task state).

Options:
  (a) Open (or find) the issue, then re-run with -Issue <number>.
  (b) Re-run with -ForceNoGate (NOT RECOMMENDED).
"@ | Write-Error
        exit 3
    }
    if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
        Write-Error "gh CLI not found -- cannot verify the work-gate issue #$Issue. Install gh, or re-run with -ForceNoGate (NOT RECOMMENDED)."
        exit 3
    }
    $GateInfo = $null
    try {
        $GateInfo = gh issue view "$Issue" --json state,title --jq '.state + "\t" + .title' 2>&1
        if ($LASTEXITCODE -ne 0) { $GateInfo = $null }
    } catch { $GateInfo = $null }
    if (-not $GateInfo) {
        Write-Error "Work-gate issue #$Issue not found (or gh failed)."
        exit 3
    }
    $Parts = ($GateInfo | Out-String).Trim() -split "`t", 2
    $IssueState = $Parts[0]
    $IssueTitle = if ($Parts.Count -gt 1) { $Parts[1] } else { '' }
    if ($IssueState -ne 'OPEN') {
        Write-Error "Work-gate issue #$Issue is not open (state: $IssueState). The work-gate is an OPEN issue."
        exit 3
    }
    Write-Host "[INFO] Work-gate OK: issue #$Issue is open -- $IssueTitle"
}

# --- Scaffold ---
New-Item -ItemType Directory -Path $SpecDir | Out-Null
$Today = (Get-Date).ToUniversalTime().ToString('yyyy-MM-dd')
$Title = $FeatureId

foreach ($tpl in @('proposal', 'tasks', 'verification')) {
    $src = Join-Path $TemplatesDir "spec-$tpl.md"
    $dst = Join-Path $SpecDir "$tpl.md"
    if (-not (Test-Path $src)) {
        Write-Error "Template missing: $src"
        exit 2
    }
    $content = (Get-Content $src -Raw) `
        -replace '<feature-id>', $FeatureId `
        -replace '\{TITLE\}', $Title `
        -replace '\{\{date:YYYY-MM-DD\}\}', $Today
    Set-Content -Path $dst -Value $content -NoNewline
}

# --- Inject issue context comment in proposal Why ---
if ($IssueTitle) {
    $proposal = Join-Path $SpecDir 'proposal.md'
    $contextLine = "<!-- from issue #${Issue}: $IssueTitle -->"
    $content = Get-Content $proposal -Raw
    $content = $content -replace '(?m)^(## Why)\s*$', "`$1`n`n$contextLine"
    Set-Content -Path $proposal -Value $content -NoNewline
}

# --- Output ---
Write-Host ''
Write-Host "[OK] Created: $SpecDir"
Write-Host '     proposal.md, tasks.md, verification.md'
if ($IssueTitle) {
    Write-Host "     Work-gate linked: issue #$Issue"
}
Write-Host ''
Write-Host "Next: fill proposal.md interactively (`"/spec fill $FeatureId`" in an agent)"
Write-Host '      or edit by hand. Do not skip the Why.'
