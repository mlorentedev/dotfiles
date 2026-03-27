<#
.SYNOPSIS
    Thin wrapper around Obsidian CLI for Windows.

.DESCRIPTION
    Handles default vault name and GUI connectivity check.
    Passes all arguments through to the Obsidian CLI.

    Exit codes:
      0 = success
      1 = error
      2 = GUI not running

.PARAMETER Command
    The Obsidian CLI command to execute (e.g., backlinks, orphans, tags).

.PARAMETER Args
    Additional arguments passed through to the Obsidian CLI.

.EXAMPLE
    obs-cli.ps1 backlinks file="path/to/note.md"
    obs-cli.ps1 orphans
    obs-cli.ps1 tags
    obs-cli.ps1 tags:rename old=wip new=in-progress
    obs-cli.ps1 search query="pattern"
    obs-cli.ps1 eval "app.vault.getFiles().length"

.NOTES
    Requires Obsidian desktop app running (CLI communicates via IPC).
    Set $env:OBS_VAULT to override the default vault name.
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$OBS_VAULT = if ($env:OBS_VAULT) { $env:OBS_VAULT } else { 'knowledge' }

function Write-Info { param([string]$Message) Write-Host "[INFO] $Message" -ForegroundColor Cyan }
function Write-Err  { param([string]$Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }

if ($args.Count -eq 0 -or $args[0] -eq '--help' -or $args[0] -eq '-h') {
    Write-Host @"
Usage: obs-cli.ps1 <command> [args...]

Wrapper for Obsidian CLI with default vault.

Environment:
  OBS_VAULT   Vault name (default: knowledge)

Examples:
  obs-cli.ps1 backlinks file="path/to/note.md"
  obs-cli.ps1 orphans
  obs-cli.ps1 tags
  obs-cli.ps1 tags:rename old=wip new=in-progress
  obs-cli.ps1 search query="pattern"
  obs-cli.ps1 eval "app.vault.getFiles().length"
"@
    exit 0
}

# Check obsidian is in PATH
$obsCmd = Get-Command obsidian -ErrorAction SilentlyContinue
if (-not $obsCmd) {
    Write-Err 'Obsidian not found in PATH'
    exit 1
}

# Connectivity check: verify GUI is running
try {
    $null = & obsidian vault --vault $OBS_VAULT 2>&1
    if ($LASTEXITCODE -ne 0) { throw 'not running' }
} catch {
    Write-Err 'Cannot reach Obsidian GUI. Is Obsidian running?'
    Write-Info 'Start Obsidian desktop app first.'
    exit 2
}

& obsidian @args --vault $OBS_VAULT
exit $LASTEXITCODE
