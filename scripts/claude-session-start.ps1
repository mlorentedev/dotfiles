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
