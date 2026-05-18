<#
.SYNOPSIS
    Apply opinionated GitHub repo-level defaults to a target repo via gh CLI.

.DESCRIPTION
    Idempotent: detect current state, only PATCH when divergent.

    Settings applied:
        delete_branch_on_merge = true   GitHub auto-deletes head branches
                                        on PR merge. Eliminates the slow
                                        drift of stale merged-PR branches
                                        that this script was written to
                                        prevent (captured 2026-05-18 after
                                        a marathon session left 8 orphans).

    Documented as a cross-project pattern in the vault at
    00_meta/patterns/github-branch-hygiene.md.

.PARAMETER Repo
    Target repo as owner/name. Defaults to the current repo's origin remote.

.PARAMETER DryRun
    Show what would change without applying.

.NOTES
    Requires the gh CLI authenticated (gh auth status).
    ASCII-only per project policy (PSScriptAnalyzer PSUseBOMForUnicodeEncodedFile).
#>
param(
    [string]$Repo,
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ghCmd = Get-Command gh -ErrorAction SilentlyContinue
if (-not $ghCmd) {
    Write-Error 'gh CLI not found. Install: https://cli.github.com/'
    exit 3
}

# --- Resolve repo ---
if (-not $Repo) {
    try {
        $originUrl = (& git remote get-url origin 2>$null) -replace "`r`n|`r|`n", ''
    } catch {
        $originUrl = ''
    }
    if (-not $originUrl) {
        Write-Error 'Not in a git repo or no origin remote. Use -Repo <owner/name>.'
        exit 4
    }
    # Strip git@github.com: or https://github.com/ and trailing .git
    $Repo = $originUrl -replace '^git@github\.com:', '' `
                       -replace '^https?://github\.com/', '' `
                       -replace '\.git$', ''
    if ($Repo -notmatch '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$') {
        Write-Error "Could not derive owner/name from origin ($originUrl). Use -Repo."
        exit 5
    }
}

Write-Host "[INFO] Target repo: $Repo"

# --- Current state ---
try {
    $currentState = & gh api "/repos/$Repo" --jq '.delete_branch_on_merge' 2>$null
    $currentState = "$currentState".Trim()
} catch {
    $currentState = 'unknown'
}

Write-Host "[INFO] Current delete_branch_on_merge: $currentState"

if ($currentState -eq 'true') {
    Write-Host '[OK] Already enabled, nothing to do.'
    exit 0
}

# --- Apply ---
if ($DryRun) {
    Write-Host "[DRY-RUN] Would PATCH /repos/$Repo with delete_branch_on_merge=true"
    exit 0
}

try {
    & gh api -X PATCH "/repos/$Repo" -f 'delete_branch_on_merge=true' --jq '.delete_branch_on_merge' 2>&1 | Out-Null
    Write-Host "[OK] delete_branch_on_merge enabled on $Repo"
} catch {
    Write-Error "PATCH failed: $_"
    exit 6
}
