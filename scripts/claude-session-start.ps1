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

# VAULT_PATH is the cross-machine seam (ADR-025), set by the sourced paths.ps1;
# fall back to the legacy default only when it is unset.
$KnowledgeVault = if ($env:VAULT_PATH) { $env:VAULT_PATH } else { Join-Path $env:USERPROFILE 'Projects\knowledge' }
# SDD discipline reminder -- unconditional, first in additionalContext (SDD-001).
# Surfaces the Discipline Gate to every Claude Code session regardless of CWD,
# repo, or vault state. Cross-OS parity with claude-session-start.sh. All
# subsequent diagnostic blocks (claude-mem, doctor, hive, specs, vault, memory)
# APPEND to $ContextLines defensively so they cannot wipe this reminder.
$ContextLines = '[sdd] Before your first tool call, read `AGENTS.md` at the repo root (or `~/Projects/dotfiles/AGENTS.md` as fallback) and apply its "Spec-Driven Development" (including the Discipline Gate) and "Standing Orders" sections. SDD applies by default for PR-sized changes (~50-300 LOC, public contract, new dep, multi-PR sequence). Skip ONLY for: typos, comment-only edits, mechanical refactors, bug fixes <20 lines with obvious cause, documentation-only changes. When in doubt, ASK the user.'

# --- Self-heal claude-mem plugin if marketplace shipped broken artifacts ---
# Patches .mcp.json (${_R%/} regression, upstream #2385) and installs the
# missing zod runtime dep. Silent on healthy installs. Mirrors the bash side.
$ClaudeMemHeal = Join-Path $PSScriptRoot 'claude-mem-heal.ps1'
if (Test-Path $ClaudeMemHeal) {
    try {
        $healOutput = & pwsh -NoProfile -File $ClaudeMemHeal 2>&1 | Out-String
        $healOutput = $healOutput.Trim()
        if ($healOutput) {
            $healBlock = "[claude-mem] self-healed plugin install:`n$healOutput"
            $ContextLines = "$ContextLines`n`n$healBlock"
        }
    } catch {
        # Non-fatal -- session continues even if heal fails
    }
}

# --- Silent doctor: surface env-contract drift only when detected ---
# Runs check-only; keeps only [warn]/[fail] lines.
$DoctorScript = Join-Path $PSScriptRoot 'doctor.ps1'
if (Test-Path $DoctorScript) {
    try {
        $doctorOut = & pwsh -NoProfile -File $DoctorScript 2>&1 | Out-String
        $drift = ($doctorOut -split "`n" | Where-Object { $_ -match '^\s+\[(warn|fail)\]' }) -join "`n"
        if ($drift) {
            $driftBlock = "[doctor] env-contract drift detected (run scripts/doctor.ps1 -Fix):`n$drift"
            if ($ContextLines) {
                $ContextLines = "$ContextLines`n`n$driftBlock"
            } else {
                $ContextLines = $driftBlock
            }
        }
    } catch {
        # Non-fatal
    }
}

# --- Hive: detect project and suggest vault queries ---
# Checks: 1) personal projects (10_projects/), 2) work SDK projects (50_work/45-development/).
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
        return
    }

    # Fallback: check work SDK projects under 50_work/45-development/
    Find-WorkSdkProject -Path $Path
}

# Detect work SDK nested repos by matching slugified path segments against vault family/component dirs.
function Find-WorkSdkProject {
    param([string]$Path)

    $devDir = Join-Path $KnowledgeVault '50_work\45-development'
    if (-not (Test-Path $devDir)) { return }

    $cwdSlug = $Path.ToLower() -replace '[^a-z0-9/\\]', ''

    foreach ($familyDir in (Get-ChildItem $devDir -Directory -ErrorAction SilentlyContinue)) {
        $familySlug = $familyDir.Name.ToLower() -replace '[^a-z0-9]', ''

        if ($cwdSlug -match $familySlug) {
            foreach ($compDir in (Get-ChildItem $familyDir.FullName -Directory -ErrorAction SilentlyContinue)) {
                $compSlug = $compDir.Name.ToLower() -replace '[^a-z0-9]', ''

                if ($cwdSlug -match $compSlug) {
                    $script:ContextLines += @"

[hive] Work SDK '$($compDir.Name)' (family: $($familyDir.Name)) found in vault. Use Hive MCP:
  - vault_query(project="_meta", path="50_work/45-development/$($familyDir.Name)/$($compDir.Name)/00-context.md") - project context
  - vault_query(project="_meta", path="50_work/45-development/$($familyDir.Name)/$($compDir.Name)/memory/MEMORY.md") - session memory
  - vault_query(project="_meta", path="patterns/workflow-protocol") - session protocol
"@
                    return
                }
            }
            # Family matched but no component
            $script:ContextLines += @"

[hive] Work SDK family '$($familyDir.Name)' found in vault. Use Hive MCP:
  - vault_query(project="_meta", path="50_work/45-development/$($familyDir.Name)/00-context.md") - all repos in family
"@
            return
        }
    }
}

# --- Spec-Driven Development: detect repo specs/ and surface in-flight work ---
# Counts active vs archived specs under $RepoPath/specs/ and flags any spec
# containing unresolved [AGENT-DRAFT] or [AGENT-SUGGESTION] tags.
# Discovery without proactive nag - silent if no specs/.
function Test-RepoSpecs {
    param([string]$RepoPath)

    $specsDir = Join-Path $RepoPath 'specs'
    if (-not (Test-Path $specsDir -PathType Container)) { return }

    $activeCount = 0
    $draftsSpecs = New-Object System.Collections.Generic.List[string]

    foreach ($d in (Get-ChildItem $specsDir -Directory -ErrorAction SilentlyContinue)) {
        if ($d.Name -eq 'archive') { continue }
        $activeCount++
        $hasTags = $false
        $files = Get-ChildItem $d.FullName -File -Recurse -ErrorAction SilentlyContinue
        foreach ($f in $files) {
            $content = Get-Content $f.FullName -Raw -ErrorAction SilentlyContinue
            if ($content -match '\[AGENT-DRAFT\]|\[AGENT-SUGGESTION\]') {
                $hasTags = $true
                break
            }
        }
        if ($hasTags) { $draftsSpecs.Add($d.Name) }
    }

    $archiveCount = 0
    $archiveDir = Join-Path $specsDir 'archive'
    if (Test-Path $archiveDir -PathType Container) {
        $archiveCount = @(Get-ChildItem $archiveDir -Directory -ErrorAction SilentlyContinue).Count
    }

    if ($activeCount -eq 0 -and $archiveCount -eq 0) { return }

    $msg = "[specs] $activeCount active, $archiveCount archived"
    if ($draftsSpecs.Count -gt 0) {
        $msg += " - $($draftsSpecs.Count) with unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags:"
        foreach ($name in $draftsSpecs) {
            $msg += "`n  - $name"
        }
    }

    $script:ContextLines += "`n$msg"
}

# Check if CWD is a git repo
if (Test-Path (Join-Path $CWD '.git')) {
    Find-HiveProject -Path $CWD
    Test-RepoSpecs -RepoPath $CWD
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

# Note: previous versions exited silently here when both $VaultRoot and
# $ContextLines were empty. After SDD-001 the [sdd] reminder is always present,
# so $ContextLines is never empty -- the early-exit branch is now dead code and
# was removed. The hook always emits the additionalContext envelope.

if ($VaultRoot) {
    $VaultName = Split-Path -Leaf $VaultRoot
    $ContextLines = "Obsidian vault detected: $VaultName ($VaultRoot)`n`n" + $ContextLines
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
        # Try CWD/memory/ (knowledge sessions where CWD is inside the vault)
        if ($CWD.StartsWith($KnowledgeVault, [System.StringComparison]::OrdinalIgnoreCase)) {
            $vaultMemory = Join-Path $CWD "memory"
        }
    }

    # Try 50_work/45-development/<family>/<component>/memory/ for nested work SDK repos
    if (-not (Test-Path $vaultMemory)) {
        $devDir = Join-Path $KnowledgeVault '50_work\45-development'
        if (Test-Path $devDir) {
            $cwdSlug = $CWD.ToLower() -replace '[^a-z0-9/\\]', ''
            :sdkSearch foreach ($familyDir in (Get-ChildItem $devDir -Directory -ErrorAction SilentlyContinue)) {
                $familySlug = $familyDir.Name.ToLower() -replace '[^a-z0-9]', ''
                if ($cwdSlug -match $familySlug) {
                    foreach ($compDir in (Get-ChildItem $familyDir.FullName -Directory -ErrorAction SilentlyContinue)) {
                        $compSlug = $compDir.Name.ToLower() -replace '[^a-z0-9]', ''
                        if ($cwdSlug -match $compSlug) {
                            $vaultMemory = Join-Path $compDir.FullName "memory"
                            break sdkSearch
                        }
                    }
                }
            }
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

function Test-VaultIntegrity {
    param([string]$VaultPath)

    # GUI-independent check: detect files deleted from working tree but still in HEAD.
    # Triggered by 2026-05-13 incident - SKILL.md vanished from disk, git status showed D.
    # Pattern '^.D ' matches Y-column=D (unstaged delete), NOT X-column=D (intentional git rm).
    if (-not (Test-Path (Join-Path $VaultPath '.git'))) { return }

    try {
        $deletedLines = @(& git -C $VaultPath status --short 2>$null | Where-Object { $_ -match '^.D ' })
    } catch {
        return
    }

    if ($deletedLines.Count -gt 0) {
        $count = $deletedLines.Count
        $list = ($deletedLines | ForEach-Object { "        $_" }) -join "`n"
        $script:ContextLines += @"

Vault integrity: $count file(s) deleted from working tree but still in HEAD - likely sync/watcher race
$list
        Recovery: cd $VaultPath; git checkout -- <file>
"@
    }
}

if ($VaultRoot) {
    Test-VaultIntegrity -VaultPath $VaultRoot
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

# --- Vault baseline integrity (SDD-017b) ---
# SDD-017 catches working-tree D entries (unstaged deletes). This catches
# already-auto-committed deletes: when Obsidian-git plugin commits a deletion
# before next session-start, `git status` is clean but a canonical file is gone.
# Checks: (a) Every 00_meta/skills/<name>/ dir must have a non-empty SKILL.md
#         (b) Static list of always-required canonical files must exist
function Test-VaultBaseline {
    if (-not $VaultRoot) { return }
    $metaDir = Join-Path $VaultRoot '00_meta'
    if (-not (Test-Path $metaDir)) { return }

    $issues = @()

    $skillsDir = Join-Path $metaDir 'skills'
    if (Test-Path $skillsDir) {
        Get-ChildItem -Path $skillsDir -Directory -ErrorAction SilentlyContinue | ForEach-Object {
            $skillMd = Join-Path $_.FullName 'SKILL.md'
            if (-not (Test-Path $skillMd)) {
                $issues += "        MISSING: 00_meta/skills/$($_.Name)/SKILL.md (skill dir exists but SKILL.md gone)"
            } elseif ((Get-Item $skillMd).Length -eq 0) {
                $issues += "        EMPTY: 00_meta/skills/$($_.Name)/SKILL.md (0 bytes - likely truncated)"
            }
        }
    }

    $criticalFiles = @(
        '00_meta/patterns/_index.md',
        '00_meta/skills/README.md',
        'README.md',
        'AGENTS.md'
    )
    foreach ($cf in $criticalFiles) {
        if (-not (Test-Path (Join-Path $VaultRoot $cf))) {
            $issues += "        MISSING: $cf (canonical file)"
        }
    }

    if ($issues.Count -gt 0) {
        $list = $issues -join "`n"
        $script:ContextLines += @"

Vault baseline FAIL - canonical artifacts missing (likely auto-committed delete):
$list
        Recovery: cd $VaultRoot; git log --diff-filter=D --name-only --pretty=format: -5 | Sort-Object -Unique
"@
    }
}

Test-VaultBaseline

# --- .claude.json size monitor (SDD-021) ---
# Defensive layer for the `claude plugin install` truncation bug (dotfiles#33 trigger
# fix, anthropics/claude-code#59870 upstream). If `~/.claude/.claude.json` drops below
# 10 KB, flag it - healthy is ~75 KB, post-truncation is ~1.5 KB. Catches recurrence
# even if a different `claude` CLI subcommand introduces the same strip behavior.
function Test-ClaudeJsonSize {
    $claudeJson = Join-Path $env:USERPROFILE '.claude\.claude.json'
    $threshold = 10240  # 10 KB

    if (-not (Test-Path $claudeJson)) { return }

    try {
        $size = (Get-Item $claudeJson).Length
    } catch {
        return
    }

    if ($size -gt 0 -and $size -lt $threshold) {
        $script:ContextLines += "`n[claude.json] WARNING: ~/.claude/.claude.json is $size bytes (threshold $threshold). Healthy state is ~75 KB; truncation bug (anthropics/claude-code#59870) reduces it to ~1.5 KB and silently drops subscription state. Recovery: pick newest from ~/.claude/backups/.claude.json.backup.* and copy back."
    }
}

Test-ClaudeJsonSize

# Return context to Claude via hook output format
$output = @{
    hookSpecificOutput = @{
        hookEventName = 'SessionStart'
        additionalContext = $ContextLines
    }
} | ConvertTo-Json -Depth 3

Write-Output $output
exit 0
