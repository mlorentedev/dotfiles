<#
.SYNOPSIS
    Bootstrap AGENTS.md in a repo with Spec-Driven Development snippet from vault.

.DESCRIPTION
    Idempotent - re-running is safe (skips if section already present).
    See: $VaultPath/00_meta/templates/agents-spec-section.md for the snippet source.

.PARAMETER Repo
    Target repo root (default: current git repo)

.PARAMETER Force
    Overwrite existing SDD section in AGENTS.md

.NOTES
    VAULT_PATH (env)  vault root (default: $env:USERPROFILE\Projects\knowledge)
#>
param(
    [string]$Repo,
    [switch]$Force
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# --- Resolve target repo ---
if (-not $Repo) {
    try {
        $Repo = (& git rev-parse --show-toplevel 2>$null) -replace "`r`n|`r|`n", ''
    } catch {
        $Repo = ''
    }
    if (-not $Repo) {
        Write-Error 'Not in a git repo. Use -Repo <path> or cd into a repo.'
        exit 1
    }
}

if (-not (Test-Path $Repo -PathType Container)) {
    Write-Error "Repo not found: $Repo"
    exit 1
}

# --- Resolve vault snippet ---
$VaultPath = if ($env:VAULT_PATH) { $env:VAULT_PATH } else { Join-Path $env:USERPROFILE 'Projects\knowledge' }
$SnippetSrc = Join-Path $VaultPath '00_meta\templates\agents-spec-section.md'

if (-not (Test-Path $SnippetSrc -PathType Leaf)) {
    Write-Error "Snippet template not found: $SnippetSrc. Set VAULT_PATH env var or clone vault."
    exit 2
}

# --- Extract snippet content (between markers, exclusive) ---
$lines = Get-Content $SnippetSrc
$inside = $false
$snippet = New-Object System.Collections.Generic.List[string]
foreach ($line in $lines) {
    if ($line -eq '## --- BEGIN SNIPPET ---') { $inside = $true; continue }
    if ($line -eq '## --- END SNIPPET ---')   { $inside = $false; continue }
    if ($inside) { $snippet.Add($line) }
}

if ($snippet.Count -eq 0) {
    Write-Error "BEGIN/END SNIPPET markers not found in $SnippetSrc"
    exit 2
}

$snippetText = $snippet -join "`n"

# --- Target AGENTS.md ---
$AgentsFile = Join-Path $Repo 'AGENTS.md'

# --- Idempotency / force ---
if (Test-Path $AgentsFile -PathType Leaf) {
    $existing = Get-Content $AgentsFile
    $hasSection = @($existing | Where-Object { $_ -eq '## Spec-Driven Development' }).Count -gt 0

    if ($hasSection -and -not $Force) {
        Write-Host "[OK] Spec-Driven Development section already present in $AgentsFile"
        Write-Host '     Re-run with -Force to overwrite.'
        exit 0
    }

    if ($hasSection -and $Force) {
        $output = New-Object System.Collections.Generic.List[string]
        $skip = $false
        foreach ($line in $existing) {
            if ($line -eq '## Spec-Driven Development') {
                $skip = $true
                $output.Add($snippetText)
                continue
            }
            if ($skip -and $line -match '^## ' -and $line -ne '## Spec-Driven Development') {
                $skip = $false
            }
            if (-not $skip) { $output.Add($line) }
        }
        Set-Content -Path $AgentsFile -Value ($output -join "`n") -Encoding UTF8
        Write-Host "[OK] Replaced Spec-Driven Development section in $AgentsFile"
        exit 0
    }

    # Append with leading blank line separation
    Add-Content -Path $AgentsFile -Value "`n$snippetText" -Encoding UTF8
    Write-Host "[OK] Appended Spec-Driven Development section to $AgentsFile"
} else {
    # Create new with minimal header
    $content = @"
# AGENTS.md

> Instructions for AI coding agents (Claude Code, Copilot, Cursor, Codex) operating in this repo.

$snippetText
"@
    Set-Content -Path $AgentsFile -Value $content -Encoding UTF8
    Write-Host "[OK] Created $AgentsFile with Spec-Driven Development section"
}
