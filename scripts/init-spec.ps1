# init-spec.ps1
# Purpose: Mechanical scaffold of a per-feature spec folder per pattern-spec-driven-development.
#          Vault-rooted: requires vault entry before scaffolding (SSOT discipline).
#
# Usage:
#   .\init-spec.ps1 <feature-id> [-Task <task-id>] [-ForceNoVault]
#
# Mechanical only. For Socratic proposal filling (Q1-Q6), use /spec fill in an agent.
# See: $env:VAULT_PATH\00_meta\skills\spec\SKILL.md for the full workflow.

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$FeatureId,

    [string]$Task,

    [switch]$ForceNoVault,

    [switch]$Help
)

$ErrorActionPreference = 'Stop'

if ($Help) {
    @"
Usage: init-spec.ps1 <feature-id> [-Task <task-id>] [-ForceNoVault]

  <feature-id>      e.g. AI-001-ollama-public or 2026-05-13-foo
  -Task <task-id>   override which task ID to look up in vault 11-tasks.md
  -ForceNoVault     skip vault context check (NOT RECOMMENDED — violates SSOT)
"@ | Write-Host
    exit 0
}

# --- Validate id ---
$Pattern = '^([A-Z]+-[0-9]+(-[a-z0-9-]+)?|[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9-]+)$'
if ($FeatureId -notmatch $Pattern) {
    Write-Error "Invalid feature-id: $FeatureId. Expected: TICKET-NNN[-slug] or YYYY-MM-DD-slug."
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

$RepoName = Split-Path $RepoRoot -Leaf
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

# --- Vault context check ---
$VaultLine = $null
$TasksFile = Join-Path $VaultPath "10_projects\$RepoName\11-tasks.md"

if (-not $ForceNoVault) {
    $SearchId = if ($Task) { $Task } else { $FeatureId }
    if (Test-Path $TasksFile) {
        $Escaped = [regex]::Escape($SearchId)
        $VaultLine = Select-String -Path $TasksFile -Pattern "^- \[[ x-]\] \*\*$Escaped\*\*" |
            Select-Object -First 1
    }
    if (-not $VaultLine) {
        @"
[ERROR] No vault entry found for "$SearchId" in $TasksFile.

Spec must be downstream of a vault entry (backlog/ADR/roadmap) per SSOT discipline.

Options:
  (a) Cancel. Add an entry to 11-tasks.md, then re-run.
  (b) Re-run with -ForceNoVault (NOT RECOMMENDED).
"@ | Write-Error
        exit 3
    }
    Write-Host "[INFO] Vault context found in $TasksFile"
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

# --- Inject vault context comment in proposal Why ---
if ($VaultLine) {
    $proposal = Join-Path $SpecDir 'proposal.md'
    $contextText = $VaultLine.Line -replace '^- \[[^]]+\] \*\*[^*]+\*\*:? *', ''
    $contextLine = "<!-- from 11-tasks.md: $contextText -->"
    $content = Get-Content $proposal -Raw
    $content = $content -replace '(?m)^(## Why)\s*$', "`$1`n`n$contextLine"
    Set-Content -Path $proposal -Value $content -NoNewline
}

# --- Output ---
Write-Host ''
Write-Host "[OK] Created: $SpecDir"
Write-Host '     proposal.md, tasks.md, verification.md'
if ($VaultLine) {
    Write-Host "     Vault context linked from $TasksFile"
}
Write-Host ''
Write-Host "Next: fill proposal.md interactively (`"/spec fill $FeatureId`" in an agent)"
Write-Host '      or edit by hand. Do not skip the Why.'
