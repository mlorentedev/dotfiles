#!/usr/bin/env pwsh
# session-handoff.ps1 - MEMORY-001 cross-agent session bridge (ADR-014), Windows.
#
# Mirror of session-handoff.sh. Wired to Claude Code's SessionEnd hook: reads the
# hook JSON on stdin, locates the project's MEMORY.md, and archives the /handoff
# '## Session Handoff' block into an append-only record:
#     $VAULT_PATH/00_meta/sessions/<date>-<project>-claude.md
# The agent authors (via /handoff); this hook only persists.
#
# Resilience contract: a session-end hook must NEVER crash a session. Every
# trivial / missing / malformed input is a clean no-op (exit 0).

$ErrorActionPreference = 'SilentlyContinue'

$vault = $env:VAULT_PATH
if ([string]::IsNullOrEmpty($vault)) { $vault = Join-Path $env:USERPROFILE 'Projects\knowledge' }

# 1. Read the payload; empty stdin -> no-op.
$payload = [Console]::In.ReadToEnd()
if ([string]::IsNullOrWhiteSpace($payload)) { exit 0 }

# 2. Extract cwd + session_id (graceful on malformed JSON).
try { $json = $payload | ConvertFrom-Json } catch { exit 0 }
$cwd = $json.cwd
$sid = $json.session_id
if ([string]::IsNullOrEmpty($cwd)) { exit 0 }
$project = Split-Path $cwd -Leaf

# 3. Locate the project's MEMORY.md; absent -> no-op.
$memory = Join-Path $vault "10_projects\$project\memory\MEMORY.md"
if (-not (Test-Path $memory)) { exit 0 }

# 4. Extract the '## Session Handoff' block (to the next '## ' heading or EOF).
$block = New-Object System.Collections.Generic.List[string]
$cap = $false
foreach ($line in (Get-Content $memory)) {
    if ($line -match '^## Session Handoff\s*$') { $cap = $true; continue }
    if ($cap -and ($line -match '^## ')) { break }
    if ($cap) { $block.Add($line) }
}
$blockText = ($block -join "`n")
if ([string]::IsNullOrWhiteSpace($blockText)) { exit 0 }

# 5. Append a durable, timestamped session record.
$date = (Get-Date).ToUniversalTime().ToString('yyyy-MM-dd')
if ([string]::IsNullOrEmpty($sid)) { $sid = 'unknown' }
$outDir = Join-Path $vault '00_meta\sessions'
New-Item -ItemType Directory -Force -Path $outDir | Out-Null
$out = Join-Path $outDir "$date-$project-claude.md"

$record = @(
    '---',
    "id: `"session-$date-$project-claude`"",
    'type: session',
    "session_id: $sid",
    'agent: claude',
    "project: $project",
    "date: `"$date`"",
    "tags: [session, handoff, $project]",
    '---',
    '',
    "# Session $date - $project (claude)",
    '',
    $blockText
)
$record -join "`n" | Set-Content -Path $out -Encoding UTF8
exit 0
