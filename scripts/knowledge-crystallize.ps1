<#
.SYNOPSIS
    AI knowledge maintenance tool (Windows/PowerShell equivalent of knowledge-crystallize.sh)

.DESCRIPTION
    Locates the MEMORY.md for a project (defaults to CWD), then:
      1. Removes duplicate # currentDate entries (keeps latest)
      2. Updates # currentDate to today
      3. Stamps ## Last Crystallized: YYYY-MM-DD
      4. Warns if MEMORY.md exceeds 150 lines
      5. Prints a checklist of manual AI workflow actions

    Use --All to auto-discover and process every project from
    $env:USERPROFILE\.claude\projects\ in one run.

.PARAMETER ProjectDir
    Path to the project directory. Defaults to current working directory.

.PARAMETER All
    Auto-discover and process ALL projects from $env:USERPROFILE\.claude\projects\

.EXAMPLE
    .\knowledge-crystallize.ps1
    .\knowledge-crystallize.ps1 -ProjectDir C:\Users\manu\Projects\kubelab
    .\knowledge-crystallize.ps1 -All

.NOTES
    Idempotent: safe to run multiple times.
    PowerShell equivalent of scripts/knowledge-crystallize.sh
#>

[CmdletBinding(DefaultParameterSetName = 'Single')]
param(
    [Parameter(ParameterSetName = 'Single', Position = 0)]
    [string]$ProjectDir = (Get-Location).Path,

    [Parameter(ParameterSetName = 'All')]
    [switch]$All
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Shared PowerShell helpers. Get-ClaudeProjectKey / Get-ClaudeProjectKeyEncoded
# are the single source of the Claude auto-memory key encoding (they defer to the
# Go layer via `dotf mem project-key`), so this script cannot re-drift from Claude
# Code the way it did in #689 (deleting ':' -> the wrong single-dash key).
. (Join-Path $PSScriptRoot 'utils.ps1')

# ============================================================================
# HELPERS
# ============================================================================

function Write-Info    { param([string]$m) Write-Host "[INFO]    $m" -ForegroundColor Blue }
function Write-Success { param([string]$m) Write-Host "[SUCCESS] $m" -ForegroundColor Green }
function Write-Warn    { param([string]$m) Write-Host "[WARNING] $m" -ForegroundColor Yellow }

# Decode a Claude Code encoded key back to a real filesystem path.
# Two-stage: drive-letter heuristic first, then filesystem scan. The key format is
# the double-dash drive encoding (C:\... -> C--...), matching Get-ClaudeProjectKey.
function Get-DecodedPath {
    param([string]$Encoded)

    # Stage 1: Drive-letter heuristic.
    # A key from a Windows path starts with '<drive>--' because the drive ':' and
    # its trailing '\' both map to '-' (C:\Users -> C--Users). Capture the drive,
    # then turn the remaining '-' back into path separators.
    if ($Encoded -match '^([A-Za-z])--(.+)$') {
        $drive    = $Matches[1].ToUpper()
        $rest     = $Matches[2].Replace('-', '\')
        $candidate = "${drive}:\${rest}"
        if (Test-Path $candidate -PathType Container) {
            return $candidate
        }
    }

    # Stage 2: Filesystem scan under USERPROFILE (handles dashes in dir names).
    # Uses the pure encoder (no per-directory subprocess) for the hot comparison.
    $found = Get-ChildItem -Path $env:USERPROFILE -Recurse -Directory -Depth 4 `
        -ErrorAction SilentlyContinue |
        Where-Object { (Get-ClaudeProjectKeyEncoded $_.FullName) -eq $Encoded } |
        Select-Object -First 1

    if ($found) { return $found.FullName }
    return $null
}

# Find the MEMORY.md path for a given project directory.
function Get-MemoryFilePath {
    param([string]$ProjectPath)
    $encoded = Get-ClaudeProjectKey $ProjectPath
    return Join-Path $env:USERPROFILE ".claude\projects\$encoded\memory\MEMORY.md"
}

# Remove duplicate # currentDate sections (keep last)
function Remove-DuplicateDate {
    param([string]$FilePath)
    $content = Get-Content $FilePath -Raw
    $pattern = '(# currentDate\r?\nToday''s date is [^\n]+\r?\n?)'
    $dateMatches = [regex]::Matches($content, $pattern)

    if ($dateMatches.Count -le 1) { return }

    Write-Info "Removing $($dateMatches.Count - 1) duplicate # currentDate entries..."

    # Keep only the last occurrence: remove all, then re-append the last
    $last = $dateMatches[$dateMatches.Count - 1].Value
    $cleaned = [regex]::Replace($content, $pattern, '')
    $cleaned = $cleaned.TrimEnd() + "`n`n" + $last.TrimEnd() + "`n"
    Set-Content -Path $FilePath -Value $cleaned -Encoding UTF8 -NoNewline
}

# Append a new section WITHOUT displacing the "## Session Handoff" block.
#
# HARNESS-029 requires that block to be the LAST section of MEMORY.md: it is
# rewritten every session, so keeping it out of the auto-loaded KV-cache prefix
# stops it from busting the provider prompt cache on every new session. A bare
# Add-Content appends after it and silently breaks that invariant - which is
# what happened on the first crystallize of a HARNESS-029-compliant file.
#
# So: insert before the handoff block when present, append when it is not.
function Add-SectionBeforeHandoff {
    param([string]$FilePath, [string]$Block)
    $content = Get-Content $FilePath -Raw

    if ($content -match '(?m)^## Session Handoff') {
        $newContent = [regex]::Replace(
            $content,
            '(?m)^## Session Handoff',
            ($Block.TrimEnd() + "`n`n## Session Handoff"),
            1
        )
        Set-Content -Path $FilePath -Value $newContent -Encoding UTF8 -NoNewline
    } else {
        Add-Content -Path $FilePath -Value $Block -Encoding UTF8
    }
}

# Update the # currentDate section to today
function Update-CurrentDate {
    param([string]$FilePath, [string]$Today)
    $content = Get-Content $FilePath -Raw

    if ($content -match '# currentDate') {
        $newContent = [regex]::Replace(
            $content,
            '(# currentDate\r?\n)Today''s date is [^\n]+',
            "`$1Today's date is $Today."
        )
        Set-Content -Path $FilePath -Value $newContent -Encoding UTF8 -NoNewline
        Write-Info "Updated currentDate to $Today"
    } else {
        Add-SectionBeforeHandoff -FilePath $FilePath -Block "`n# currentDate`nToday's date is $Today."
        Write-Info "Added currentDate section ($Today)"
    }
}

# Stamp or update ## Last Crystallized date
function Set-LastCrystallized {
    param([string]$FilePath, [string]$Today)
    $content = Get-Content $FilePath -Raw

    if ($content -match '## Last Crystallized:') {
        $newContent = [regex]::Replace(
            $content,
            '## Last Crystallized: \S+',
            "## Last Crystallized: $Today"
        )
        Set-Content -Path $FilePath -Value $newContent -Encoding UTF8 -NoNewline
        Write-Info "Updated Last Crystallized to $Today"
    } elseif ($content -match '# currentDate') {
        $newContent = [regex]::Replace(
            $content,
            '(# currentDate)',
            "## Last Crystallized: $Today`n`n`$1"
        )
        Set-Content -Path $FilePath -Value $newContent -Encoding UTF8 -NoNewline
        Write-Info "Added Last Crystallized stamp ($Today)"
    } else {
        Add-SectionBeforeHandoff -FilePath $FilePath -Block "`n## Last Crystallized: $Today"
        Write-Info "Added Last Crystallized stamp ($Today)"
    }
}

# Warn if MEMORY.md exceeds line limit
function Test-MemoryFileSize {
    param([string]$FilePath, [int]$Limit = 150)
    $lines = (Get-Content $FilePath | Measure-Object -Line).Lines
    if ($lines -gt $Limit) {
        Write-Warn "MEMORY.md has $lines lines (limit: $Limit) - run /crystallize to trim"
        return $false
    }
    Write-Success "MEMORY.md line count: $lines / $Limit"
    return $true
}

# Print the manual action checklist
function Write-Checklist {
    Write-Host ""
    Write-Host "=== Knowledge Crystallization Checklist ===" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Manual steps (AI-assisted, run in Claude Code):"
    Write-Host "  [ ] /insights  - audit observation backlog"
    Write-Host "  [ ] /crystallize - promote observations to vault lessons"
    Write-Host "  [ ] Check $env:VAULT_PATH\00_meta\patterns\ for new patterns"
    Write-Host "  [ ] Update bitácora GitHub Project board / issue"
    Write-Host ""
    Write-Host "Automated (done by this script):"
    Write-Host "  [x] currentDate updated"
    Write-Host "  [x] Last Crystallized stamped"
    Write-Host "  [x] Duplicate date entries removed"
    Write-Host ""
}

# True when MEMORY.md stores its body inside a YAML block scalar ("content: |"),
# the shape Claude Code's auto-memory uses for some projects.
#
# Structural detection, not an indent probe: the indent width is NOT fixed -
# pollex indents six spaces, hive four - so nothing here may key on a literal
# width. A wrapper is a "---" first line plus a "<key>: |" block-scalar opener.
function Test-YamlBlockScalar {
    param([string]$FilePath)

    $lines = @(Get-Content -Path $FilePath -ErrorAction SilentlyContinue)
    if ($lines.Count -eq 0) { return $false }
    if ($lines[0] -notmatch '^---\s*$') { return $false }
    foreach ($line in $lines) {
        if ($line -match '^[A-Za-z_][A-Za-z0-9_-]*:\s*\|[-+0-9]*\s*$') { return $true }
    }
    return $false
}

# Core logic: process one project's MEMORY.md
function Invoke-ProjectCrystallize {
    param([string]$MemoryFile, [string]$Today)

    # BUG-062 (#857): every marker below is anchored at column 0, so on a
    # block-scalar file none of them match, control reaches the bare append, and
    # the text lands OUTSIDE the block - breaking the handoff invariant,
    # duplicating the date stamp, and making the file stop parsing as YAML, all
    # while printing [SUCCESS]. Refusing is strictly better than corrupting.
    # The YAML-aware implementation belongs to the Go port (#490).
    if (Test-YamlBlockScalar -FilePath $MemoryFile) {
        throw "Refusing to stamp $MemoryFile - its body sits inside a YAML block scalar, which this script cannot edit without corrupting the file (#857). Stamp it by hand until the YAML-aware 'dotf vault crystallize' lands (#490)."
    }

    Remove-DuplicateDate  -FilePath $MemoryFile
    Update-CurrentDate    -FilePath $MemoryFile -Today $Today
    Set-LastCrystallized  -FilePath $MemoryFile -Today $Today
    Test-MemoryFileSize   -FilePath $MemoryFile | Out-Null
    Write-Success "Updated: $MemoryFile"
}

# ============================================================================
# MAIN
# ============================================================================

$Today = Get-Date -Format 'yyyy-MM-dd'

if ($All) {
    # --All mode: auto-discover projects
    $projectsDir = Join-Path $env:USERPROFILE '.claude\projects'
    Write-Info "Date: $Today"
    Write-Info "Discovering all projects in $projectsDir..."
    Write-Host ""

    $found = 0; $processed = 0; $skipped = 0

    Get-ChildItem -Path $projectsDir -Directory -ErrorAction SilentlyContinue | ForEach-Object {
        $memoryFile = Join-Path $_.FullName 'memory\MEMORY.md'
        if (-not (Test-Path $memoryFile)) { return }

        $found++
        $encodedName = $_.Name
        $projectPath = Get-DecodedPath -Encoded $encodedName

        if ($projectPath) {
            Write-Info "[$encodedName] -> $projectPath"
            try {
                Invoke-ProjectCrystallize -MemoryFile $memoryFile -Today $Today
            } catch {
                # A refusal counts as skipped, not processed: reporting "5 / 5"
                # while having declined one is the same "prints success while
                # doing nothing" failure this guard exists to stop.
                Write-Warn "Failed to process $projectPath`: $_"
                $skipped++
            }
        } else {
            Write-Warn "[$encodedName] -> not found on disk (different machine or deleted - skipping)"
            $skipped++
        }
        Write-Host ""
    }

    if ($found -eq 0) {
        Write-Warn "No MEMORY.md files found in $projectsDir"
        Write-Warn "Open any project in Claude Code first to initialize its memory directory."
        exit 0
    }

    $processed = $found - $skipped
    Write-Success "Processed $processed / $found projects ($skipped skipped)"
    Write-Checklist

} else {
    # Single project mode
    $ProjectDir = (Resolve-Path $ProjectDir).Path
    Write-Info "Project: $ProjectDir"
    Write-Info "Date: $Today"

    $memoryFile = Get-MemoryFilePath -ProjectPath $ProjectDir
    if (-not (Test-Path $memoryFile)) {
        $encoded = Get-ClaudeProjectKey $ProjectDir
        Write-Warn "No MEMORY.md found for $ProjectDir"
        Write-Warn "Expected: $env:USERPROFILE\.claude\projects\$encoded\memory\MEMORY.md"
        Write-Warn "Run Claude Code in this project first to initialize the memory directory."
        exit 0
    }

    Write-Info "MEMORY.md: $memoryFile"
    try {
        Invoke-ProjectCrystallize -MemoryFile $memoryFile -Today $Today
    } catch {
        Write-Warn "$_"
        exit 1
    }
    Write-Checklist
}
