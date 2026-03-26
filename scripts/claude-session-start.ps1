<#
.SYNOPSIS
    Claude Code SessionStart hook for Windows

.DESCRIPTION
    Runs automatically at the start of every Claude Code session.
    Detects if CWD is inside an Obsidian vault and provides vault
    health context to Claude.

    Hook input: JSON on stdin with { cwd, session_id, ... }
    Hook output: JSON on stdout with additionalContext

.NOTES
    Deployed via dotfiles to ~/scripts/
    Registered in ~/.claude/settings.json under hooks.SessionStart
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Read hook input from stdin
$Input = [Console]::In.ReadToEnd()
$CWD = ''

try {
    $hookData = $Input | ConvertFrom-Json
    $CWD = $hookData.cwd
} catch {
    # Ignore parse errors
}

if (-not $CWD) {
    $CWD = (Get-Location).Path
}

$KnowledgeVault = Join-Path $env:USERPROFILE 'Projects\knowledge'
$ContextLines = ''

# --- Hive: detect project and suggest vault queries ---
function Find-HiveProject {
    param([string]$Path)

    $repoName = Split-Path -Leaf $Path
    $vaultProjectDir = Join-Path $KnowledgeVault "10_projects\$repoName"

    if (Test-Path $vaultProjectDir) {
        $script:ContextLines += @"

[hive] Project '$repoName' found in vault. Use hive-vault MCP tools for on-demand context:
  - vault_query(project="$repoName", section="context") - project overview
  - vault_query(project="$repoName", section="tasks") - active backlog
  - vault_search(query="...") - search across vault
  - vault_query(project="_meta", path="patterns/...") - cross-project patterns
"@
    }
}

# Check if CWD is a git repo
if (Test-Path (Join-Path $CWD '.git')) {
    Find-HiveProject -Path $CWD
}

# --- Walk up to find Obsidian vault ---
function Find-VaultRoot {
    param([string]$Path)

    $dir = $Path
    while ($dir -and $dir -ne [System.IO.Path]::GetPathRoot($dir)) {
        if (Test-Path (Join-Path $dir '.obsidian')) {
            return $dir
        }
        $dir = Split-Path -Parent $dir
    }
    return $null
}

$VaultRoot = Find-VaultRoot -Path $CWD

if (-not $VaultRoot -and -not $ContextLines) {
    # Not inside a vault and no project detected - exit cleanly
    exit 0
}

if ($VaultRoot) {
    $VaultName = Split-Path -Leaf $VaultRoot
    $ContextLines = "Obsidian vault detected: $VaultName ($VaultRoot)" + $ContextLines
}

# --- Knowledge maintenance health check ---
function Get-EncodedProjectPath {
    param([string]$Path)
    return $Path.Replace('\', '-').Replace(':', '').Replace('/', '-')
}

# --- Auto-create memory junction if vault has memory/ for this project ---
# Runs before health check so the junction exists when health check reads it.
function Ensure-MemoryJunction {
    $encoded = Get-EncodedProjectPath -Path $CWD
    $targetDir = Join-Path $env:USERPROFILE ".claude\projects\$encoded\memory"

    # Already linked? Skip.
    if (Test-Path $targetDir) {
        $item = Get-Item $targetDir -Force
        if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { return }
        if ((Get-ChildItem $targetDir -ErrorAction SilentlyContinue).Count -gt 0) { return }
    }

    # Try 10_projects/<name>/memory/ (personal projects convention)
    $projectName = Split-Path -Leaf $CWD
    $vaultMemory = Join-Path $KnowledgeVault "10_projects\$projectName\memory"
    if (-not (Test-Path $vaultMemory)) {
        # Try CWD/memory/ (work projects where CWD is inside the vault)
        if ($CWD.StartsWith($KnowledgeVault, [System.StringComparison]::OrdinalIgnoreCase)) {
            $vaultMemory = Join-Path $CWD "memory"
        }
    }

    if (-not (Test-Path $vaultMemory)) { return }

    # Create parent dir and junction
    $parentDir = Split-Path $targetDir -Parent
    if (-not (Test-Path $parentDir)) { New-Item $parentDir -ItemType Directory -Force | Out-Null }

    # Remove empty target dir if it exists (no files, not a junction)
    if ((Test-Path $targetDir) -and (Get-ChildItem $targetDir -ErrorAction SilentlyContinue).Count -eq 0) {
        Remove-Item $targetDir -Force -ErrorAction SilentlyContinue
    }

    try {
        New-Item -ItemType Junction -Path $targetDir -Target $vaultMemory -Force | Out-Null
        $script:ContextLines += "`n[auto-memory] Created junction for $projectName"
    } catch {
        # Non-fatal — session continues without junction
    }
}

Ensure-MemoryJunction

function Test-KnowledgeHealth {
    $encoded = Get-EncodedProjectPath -Path $CWD
    $memoryFile = Join-Path $env:USERPROFILE ".claude\projects\$encoded\memory\MEMORY.md"

    if (-not (Test-Path $memoryFile)) { return }

    $lineCount = (Get-Content $memoryFile).Count
    $today = Get-Date -Format 'yyyy-MM-dd'

    if ($lineCount -gt 150) {
        $script:ContextLines += "`nMEMORY.md has $lineCount lines (limit: 150) - run /crystallize to trim"
    }

    $lastDateLine = Select-String -Path $memoryFile -Pattern '^## Last Crystallized:' -ErrorAction SilentlyContinue | Select-Object -Last 1
    if (-not $lastDateLine) {
        $script:ContextLines += "`nKnowledge crystallization never run - run: ./scripts/knowledge-crystallize.ps1"
        return
    }

    $lastDate = ($lastDateLine.Line -replace '## Last Crystallized: ', '').Trim()
    try {
        $todayDate = [datetime]::ParseExact($today, 'yyyy-MM-dd', $null)
        $lastParsed = [datetime]::ParseExact($lastDate, 'yyyy-MM-dd', $null)
        $daysSince = ($todayDate - $lastParsed).Days
        if ($daysSince -gt 14) {
            $script:ContextLines += "`nKnowledge crystallization $daysSince days ago - consider running /crystallize"
        }
    } catch {
        # Date parse failed, skip
    }
}

Test-KnowledgeHealth

# Return context to Claude via hook output format
$output = @{
    hookSpecificOutput = @{
        hookEventName = 'SessionStart'
        additionalContext = $ContextLines
    }
} | ConvertTo-Json -Depth 3

Write-Output $output
exit 0
