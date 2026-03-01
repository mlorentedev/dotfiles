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

# ============================================================================
# HELPERS
# ============================================================================

function Write-Info    { param([string]$m) Write-Host "[INFO]    $m" -ForegroundColor Blue }
function Write-Success { param([string]$m) Write-Host "[SUCCESS] $m" -ForegroundColor Green }
function Write-Warn    { param([string]$m) Write-Host "[WARNING] $m" -ForegroundColor Yellow }

# Encode a path the same way Claude Code does on Windows:
# Replace all '\' with '-' and strip the ':' after drive letters
# e.g. C:\Users\manu\Projects\dotfiles -> C-Users-manu-Projects-dotfiles
function Get-EncodedPath {
    param([string]$Path)
    return $Path.Replace('\', '-').Replace(':', '')
}

# Decode a Claude Code encoded path back to a real filesystem path.
# Two-stage: drive-letter heuristic first, then filesystem scan.
function Get-DecodedPath {
    param([string]$Encoded)

    # Stage 1: Drive-letter heuristic
    # Pattern: single uppercase letter + '-Users-' or '-' -> C:\...
    if ($Encoded -match '^([A-Za-z])-(.+)$') {
        $drive    = $Matches[1].ToUpper()
        $rest     = $Matches[2].Replace('-', '\')
        $candidate = "${drive}:\${rest}"
        if (Test-Path $candidate -PathType Container) {
            return $candidate
        }
    }

    # Stage 2: Filesystem scan under USERPROFILE (handles dashes in dir names)
    $found = Get-ChildItem -Path $env:USERPROFILE -Recurse -Directory -Depth 4 `
        -ErrorAction SilentlyContinue |
        Where-Object { (Get-EncodedPath $_.FullName) -eq $Encoded } |
        Select-Object -First 1

    if ($found) { return $found.FullName }
    return $null
}

# Find the MEMORY.md path for a given project directory
function Get-MemoryFilePath {
    param([string]$ProjectPath)
    $encoded = Get-EncodedPath $ProjectPath
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
        Add-Content -Path $FilePath -Value "`n# currentDate`nToday's date is $Today." -Encoding UTF8
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
        Add-Content -Path $FilePath -Value "`n## Last Crystallized: $Today" -Encoding UTF8
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
    Write-Host "  [ ] Check ~/Projects/knowledge/00_meta/patterns/ for new patterns"
    Write-Host "  [ ] Update 11-tasks.md backlog progress bar"
    Write-Host ""
    Write-Host "Automated (done by this script):"
    Write-Host "  [x] currentDate updated"
    Write-Host "  [x] Last Crystallized stamped"
    Write-Host "  [x] Duplicate date entries removed"
    Write-Host ""
}

# Core logic: process one project's MEMORY.md
function Invoke-ProjectCrystallize {
    param([string]$MemoryFile, [string]$Today)

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
                $processed++
            } catch {
                Write-Warn "Failed to process $projectPath`: $_"
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
        $encoded = Get-EncodedPath $ProjectDir
        Write-Warn "No MEMORY.md found for $ProjectDir"
        Write-Warn "Expected: $env:USERPROFILE\.claude\projects\$encoded\memory\MEMORY.md"
        Write-Warn "Run Claude Code in this project first to initialize the memory directory."
        exit 0
    }

    Write-Info "MEMORY.md: $memoryFile"
    Invoke-ProjectCrystallize -MemoryFile $memoryFile -Today $Today
    Write-Checklist
}
