# archive-spec.ps1
# Purpose: Mechanical archive of a per-feature spec folder.
#          Tag-check pre-flight enforced unless -ForceWithDrafts.
#          No vault promotion — do that interactively via /spec archive in an agent.
#
# Usage:
#   .\archive-spec.ps1 <feature-id> [-Pr <url>] [-Abandoned] [-ForceWithDrafts]
#
# See: $env:VAULT_PATH\00_meta\skills\spec\SKILL.md for the full archive workflow.

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$FeatureId,

    [string]$Pr,

    [switch]$Abandoned,

    [switch]$ForceWithDrafts,

    [switch]$Help
)

$ErrorActionPreference = 'Stop'

if ($Help) {
    @"
Usage: archive-spec.ps1 <feature-id> [-Pr <url>] [-Abandoned] [-ForceWithDrafts]

  <feature-id>           e.g. AI-001-ollama-public
  -Pr <url>              record PR URL in proposal.md (informational)
  -Abandoned             route to specs\archive\_abandoned\<id>\, set status abandoned
  -ForceWithDrafts       allow archive even with unresolved [AGENT-DRAFT] tags
"@ | Write-Host
    exit 0
}

# --- Resolve paths ---
$RepoRoot = $null
try {
    $RepoRoot = (git rev-parse --show-toplevel 2>$null) | ForEach-Object { $_.Trim() }
} catch { }
if (-not $RepoRoot) {
    Write-Error 'Not in a git repo.'
    exit 1
}

$SpecDir = Join-Path $RepoRoot "specs\$FeatureId"
if (-not (Test-Path $SpecDir)) {
    Write-Error "Spec not found: $SpecDir"
    exit 1
}

# --- Pre-flight: tag check ---
if (-not $ForceWithDrafts) {
    $tagged = Select-String -Path "$SpecDir\*.md" -Pattern '\[AGENT-(DRAFT|SUGGESTION)\]'
    if ($tagged) {
        Write-Host '[ERROR] Unresolved tags found:' -ForegroundColor Red
        foreach ($t in $tagged) {
            Write-Host ("  {0}:{1}: {2}" -f $t.Path, $t.LineNumber, $t.Line.Trim())
        }
        Write-Error 'Resolve them (accept/edit/delete) before archiving, or use -ForceWithDrafts.'
        exit 4
    }
}

# --- Determine target ---
if ($Abandoned) {
    $TargetDir = Join-Path $RepoRoot "specs\archive\_abandoned\$FeatureId"
    $NewStatus = 'abandoned'
} else {
    $TargetDir = Join-Path $RepoRoot "specs\archive\$FeatureId"
    $NewStatus = 'archived'
}

if (Test-Path $TargetDir) {
    Write-Error "Already in archive: $TargetDir"
    exit 1
}

# --- Move ---
$TargetParent = Split-Path $TargetDir -Parent
if (-not (Test-Path $TargetParent)) {
    New-Item -ItemType Directory -Path $TargetParent -Force | Out-Null
}
Move-Item -Path $SpecDir -Destination $TargetDir

# --- Update status in proposal.md frontmatter ---
$Proposal = Join-Path $TargetDir 'proposal.md'
if (Test-Path $Proposal) {
    $content = Get-Content $Proposal -Raw
    $content = $content -replace '(?m)^(status:)\s+\S+', "`$1 $NewStatus"
    Set-Content -Path $Proposal -Value $content -NoNewline
}

# --- Record PR URL if provided ---
if ($Pr) {
    $today = Get-Date -Format 'yyyy-MM-dd'
    Add-Content -Path $Proposal -Value ''
    Add-Content -Path $Proposal -Value "<!-- archived $today — PR: $Pr -->"
}

# --- Output ---
Write-Host ''
Write-Host "[OK] Archived: $SpecDir -> $TargetDir"
Write-Host "     status: $NewStatus"
if ($Pr) {
    Write-Host "     PR: $Pr"
}
Write-Host ''
Write-Host 'Vault promotion (lessons/ADR/pattern) and 11-tasks.md tick must be done'
Write-Host 'separately (via /spec archive in an agent, or by hand).'
