<#
.SYNOPSIS
    Windows setup script for dotfiles (PowerShell, no admin required)

.DESCRIPTION
    Sets up AI configuration (Claude & Gemini), PowerShell profile,
    Git config, and scripts directory for Windows environments.

    This script does NOT require administrator privileges and uses
    file copies instead of symlinks for compatibility.

.EXAMPLE
    # One-time bypass (no permanent changes)
    powershell -ExecutionPolicy Bypass -File .\setup-windows.ps1

.EXAMPLE
    # Or set policy for current user first (recommended, persistent)
    Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
    .\setup-windows.ps1

.NOTES
    - No admin rights required
    - No symlinks (copies files instead)
    - Modifies User-level PATH only
    - Creates/updates PowerShell profile
#>

[CmdletBinding()]
param()

# ============================================================================
# BUG-005: AUTO-REEXEC UNDER PWSH IF RUNNING ON WINDOWS POWERSHELL 5.1
# ============================================================================
# SDD-002 (PR #51) added Merge-ClaudeSettings which uses
# `ConvertFrom-Json -AsHashtable` -- a parameter added in PowerShell 7.0
# (https://learn.microsoft.com/powershell/scripting/whats-new/what-s-new-in-powershell-70)
# that does NOT exist in Windows PowerShell 5.1. The natural Windows command
# `PowerShell -ExecutionPolicy Bypass -File .\setup-windows.ps1` resolves
# `PowerShell` to 5.1, the Merge function's wide try/catch swallows the
# ParameterBindingException as if it were a JSON parse error, and the
# settings.json merge is silently skipped.
#
# Defense: detect the host version up front; if pwsh (7+) is on PATH,
# re-exec under it; otherwise fail loud with an install hint. The current
# script has an empty param() block, so forwarding @args is sufficient; if
# named parameters are added later, forward $PSBoundParameters explicitly.

if ($PSVersionTable.PSVersion.Major -lt 7) {
    $pwshCmd = Get-Command pwsh -ErrorAction SilentlyContinue
    if ($pwshCmd) {
        Write-Host "[INFO] Windows PowerShell $($PSVersionTable.PSVersion) detected; re-executing under pwsh ($($pwshCmd.Source)) for full feature compatibility (BUG-005)" -ForegroundColor Yellow
        & $pwshCmd.Source -NoProfile -ExecutionPolicy Bypass -File $PSCommandPath @args
        exit $LASTEXITCODE
    } else {
        Write-Host "[ERROR] Windows PowerShell $($PSVersionTable.PSVersion) detected and pwsh (PowerShell 7+) is not installed." -ForegroundColor Red
        Write-Host "        This script requires PowerShell 7+ for ConvertFrom-Json -AsHashtable" -ForegroundColor Red
        Write-Host "        support in Merge-ClaudeSettings (introduced by SDD-002 / PR #51)." -ForegroundColor Red
        Write-Host "        Install via: winget install Microsoft.PowerShell" -ForegroundColor Red
        Write-Host "        Then re-run this script." -ForegroundColor Red
        exit 1
    }
}

# ============================================================================
# CONFIGURATION
# ============================================================================

$DotfilesDir = $PSScriptRoot
$DotfilesDest = "$env:USERPROFILE\.dotfiles"
$ClaudeHome = "$env:USERPROFILE\.claude"
$GeminiHome = "$env:USERPROFILE\.gemini"
$ScriptsDir = "$env:USERPROFILE\scripts"

# ============================================================================
# HELPER FUNCTIONS
# ============================================================================

function Write-Info { param([string]$Message) Write-Host "[INFO] $Message" -ForegroundColor Blue }
function Write-Success { param([string]$Message) Write-Host "[SUCCESS] $Message" -ForegroundColor Green }
function Write-Warn { param([string]$Message) Write-Host "[WARNING] $Message" -ForegroundColor Yellow }
function Write-Err { param([string]$Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }

function Ensure-Directory {
    param([string]$Path)
    if (-not (Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

# BUG-004: defense-in-depth wrapper around `claude plugin install`. Snapshots
# ~/.claude/.claude.json before the action; if the post-action size drops below
# 50% of the snapshot (and the snapshot was >= 10 KB), restores the snapshot.
# Defends against upstream anthropics/claude-code#59870: the CLI's deserialize-
# modify-serialize cycle drops fields outside its internal struct (organizationType,
# organizationRateLimitTier, projects map, onboarding flags), shrinking the file
# from ~75 KB to ~1.5 KB and forcing re-authentication in every project. The
# existing `installedPlugins -match` idempotence guard against `claude plugin list`
# yields a false negative for claude-mem@thedotmack (not present in that listing),
# so every setup run triggers a real install of claude-mem and hits #59870 --
# this wrapper is the second layer that catches the false-negative case.
# Complementary to SDD-021 session-start canary in claude-session-start.ps1
# (same 10240-byte threshold, same upstream issue, different detection moment).
# See dotfiles#33 for the original incomplete trigger fix.
function Backup-AndRestoreClaudeJson {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][scriptblock]$Action
    )
    $claudeJson = Join-Path $env:USERPROFILE '.claude\.claude.json'
    $backup = $null
    $snapshotSize = 0
    if (Test-Path $claudeJson) {
        $snapshotSize = (Get-Item $claudeJson).Length
        $backup = [System.IO.Path]::GetTempFileName()
        Copy-Item $claudeJson $backup -Force
    }
    try {
        & $Action
    } finally {
        if ($backup -and (Test-Path $backup)) {
            if ((Test-Path $claudeJson) -and $snapshotSize -ge 10240) {
                $newSize = (Get-Item $claudeJson).Length
                if ($newSize -lt ($snapshotSize / 2)) {
                    Copy-Item $backup $claudeJson -Force
                    Write-Warn ".claude.json shrunk from $snapshotSize to $newSize bytes after install (upstream #59870); restored from backup"
                }
            }
            Remove-Item $backup -Force -ErrorAction SilentlyContinue
        }
    }
}

# Merge `ai/claude/settings.json` template into the deployed `~/.claude/settings.json`
# per the per-key policy in specs/SDD-002-settings-portability/proposal.md. Bootstrap
# when target missing. Preserves user customizations (Read paths,
# additionalDirectories, third-party hooks like claude-mem / GitGuardian) by only
# touching the keys declared as "ours" in the template. The template's
# __HOOK_COMMAND__ placeholder is replaced with the OS-specific hook command
# before any merge / write.
function Merge-ClaudeSettings {
    [CmdletBinding()]
    # "Settings" is the canonical Claude Code config-file name (settings.json);
    # the function operates on the whole file, not one setting -- using
    # `Setting` (singular) would be misleading. Plural noun warning suppressed.
    [Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSUseSingularNouns', '')]
    param(
        [Parameter(Mandatory)][string]$TemplatePath,
        [Parameter(Mandatory)][string]$TargetPath,
        [Parameter(Mandatory)][string]$HookCommand
    )

    if (-not (Test-Path $TemplatePath)) {
        Write-Warn "Claude settings template not found at $TemplatePath, skipping merge"
        return
    }

    # Read template, JSON-escape the hook command, substitute __HOOK_COMMAND__
    $escapedCommand = ($HookCommand -replace '\\', '\\') -replace '"', '\"'
    $templateRaw = (Get-Content $TemplatePath -Raw) -replace '__HOOK_COMMAND__', $escapedCommand
    try {
        $template = $templateRaw | ConvertFrom-Json -AsHashtable
    } catch {
        Write-Warn "Claude settings template is not valid JSON after placeholder substitution: $_"
        return
    }

    # Bootstrap if target missing
    if (-not (Test-Path $TargetPath)) {
        Write-Info "Bootstrapping ~/.claude/settings.json from template (file did not exist)"
        $template | ConvertTo-Json -Depth 10 | Set-Content $TargetPath -Encoding UTF8
        Write-Success "Claude settings.json bootstrapped from template"
        return
    }

    # Read existing target
    try {
        $existing = Get-Content $TargetPath -Raw | ConvertFrom-Json -AsHashtable
    } catch {
        Write-Warn "Claude settings.json at $TargetPath is not valid JSON, skipping merge: $_"
        return
    }
    if ($null -eq $existing) { $existing = @{} }

    # Per-key merge policy (table in proposal.md)
    if ($template.ContainsKey('model')) { $existing['model'] = $template['model'] }
    if ($template.ContainsKey('effortLevel')) { $existing['effortLevel'] = $template['effortLevel'] }

    # permissions.allow: UNION (template + existing, deduped)
    if ($template.ContainsKey('permissions') -and $template['permissions'].ContainsKey('allow')) {
        if (-not $existing.ContainsKey('permissions')) { $existing['permissions'] = @{} }
        if (-not $existing['permissions'].ContainsKey('allow')) { $existing['permissions']['allow'] = @() }
        $merged = @(@($existing['permissions']['allow']) + @($template['permissions']['allow']) | Select-Object -Unique)
        $existing['permissions']['allow'] = $merged
    }

    # hooks.SessionStart: TEMPLATE wins (replace entire array). Other hook
    # surfaces (PreToolUse, PostToolUse, Stop) are untouched -- third parties
    # register there.
    if ($template.ContainsKey('hooks') -and $template['hooks'].ContainsKey('SessionStart')) {
        if (-not $existing.ContainsKey('hooks')) { $existing['hooks'] = @{} }
        $existing['hooks']['SessionStart'] = $template['hooks']['SessionStart']
    }

    # enabledPlugins: object merge (template wins on conflict). User-added
    # plugins beyond the 14 universal ones survive.
    if ($template.ContainsKey('enabledPlugins')) {
        if (-not $existing.ContainsKey('enabledPlugins')) { $existing['enabledPlugins'] = @{} }
        foreach ($plugin in $template['enabledPlugins'].Keys) {
            $existing['enabledPlugins'][$plugin] = $template['enabledPlugins'][$plugin]
        }
    }

    $existing | ConvertTo-Json -Depth 10 | Set-Content $TargetPath -Encoding UTF8
    Write-Success "Claude settings.json merged from template (user customizations preserved)"
}

# ============================================================================
# BANNER
# ============================================================================

Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  Dotfiles Setup - Windows (PowerShell)    " -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# ============================================================================
# 1. CREATE DIRECTORIES
# ============================================================================

Write-Info "Creating directories..."

Ensure-Directory $ClaudeHome
Ensure-Directory "$ClaudeHome\skills"
Ensure-Directory $ScriptsDir

Write-Success "Directories created"

# ============================================================================
# 1b. DEPLOY VERSIONS.CONF
# ============================================================================

Write-Info "Deploying versions.conf..."

$versionsSource = "$DotfilesDir\versions.conf"
if (Test-Path $versionsSource) {
    Ensure-Directory $DotfilesDest
    Copy-Item $versionsSource "$DotfilesDest\" -Force
    Write-Success "versions.conf deployed to $DotfilesDest\"
} else {
    Write-Warn "versions.conf not found at $versionsSource"
}

# ============================================================================
# 1c. DEVELOPER TOOLS (via winget)
# ============================================================================

Write-Info "Installing developer tools..."

$wingetCmd = Get-Command winget -ErrorAction SilentlyContinue
if ($wingetCmd) {
    $tools = @(
        @{ Name = "age"; Cmd = "age"; Id = "FiloSottile.age" },
        @{ Name = "eza"; Cmd = "eza"; Id = "eza-community.eza" },
        @{ Name = "jq"; Cmd = "jq"; Id = "jqlang.jq" },
        @{ Name = "GitHub CLI"; Cmd = "gh"; Id = "GitHub.cli" },
        @{ Name = "zoxide"; Cmd = "zoxide"; Id = "ajeetdsouza.zoxide" },
        @{ Name = "GitHub Copilot CLI"; Cmd = "copilot"; Id = "GitHub.Copilot" }
    )
    foreach ($tool in $tools) {
        if (-not (Get-Command $tool.Cmd -ErrorAction SilentlyContinue)) {
            Write-Info "Installing $($tool.Name)..."
            try {
                & winget install $tool.Id --accept-package-agreements --accept-source-agreements 2>$null | Out-Null
                Write-Success "$($tool.Name) installed"
            } catch {
                Write-Warn "Failed to install $($tool.Name): $_"
            }
        } else {
            Write-Info "$($tool.Name) already installed"
        }
    }
    # Refresh PATH so freshly-installed winget tools are visible to subsequent
    # blocks of this same setup run (otherwise Get-Command misses them until
    # the next shell start; first introduced for BUG-003 so the Copilot config
    # deploy block sees the just-installed `copilot` binary).
    $env:PATH = [Environment]::GetEnvironmentVariable("PATH", "Machine") + ";" + [Environment]::GetEnvironmentVariable("PATH", "User")
} else {
    Write-Warn "winget not found, skipping developer tools installation"
}

# ============================================================================
# 2. DEPLOY CLAUDE CONFIGURATION
# ============================================================================

Write-Info "Deploying Claude configuration..."

# Bulk copy all Claude config files EXCEPT settings.json (SDD-002: handled by
# Merge-ClaudeSettings below, which substitutes __HOOK_COMMAND__ and applies the
# per-key merge policy preserving user customizations).
$claudeSource = "$DotfilesDir\ai\claude"
if (Test-Path $claudeSource) {
    Copy-Item "$claudeSource\*" "$ClaudeHome\" -Recurse -Force -Exclude 'settings.json' -ErrorAction SilentlyContinue
}

# Force copy CLAUDE.md (Neural Hive Protocol)
$claudeMdSource = "$DotfilesDir\ai\claude\CLAUDE.md"
if (Test-Path $claudeMdSource) {
    Copy-Item $claudeMdSource "$ClaudeHome\" -Force
    if (Select-String -Path "$ClaudeHome\CLAUDE.md" -Pattern 'First, read `AGENTS.md`' -SimpleMatch -Quiet) {
        Write-Success "CLAUDE.md deployed successfully (verified pointer to AGENTS.md)"
    } else {
        Write-Err "CLAUDE.md deployment failed verification (expected pointer to AGENTS.md)"
    }
} else {
    Write-Warn "CLAUDE.md not found at $claudeMdSource"
}

# Sync skills: remove stale skill directories not in source, then copy current
$skillsSource = "$DotfilesDir\ai\skills"
if (Test-Path $skillsSource) {
    # Remove stale skills from target that no longer exist in source.
    # CRITICAL: Junction directories (vault-hosted skills, see vault-skills loop
    # later in this script) must be detected via the ReparsePoint attribute and
    # removed with .Delete() only. Remove-Item -Recurse -Force on a junction in
    # Windows PowerShell 5.1 historically follows the link and deletes the target
    # contents. The vault-skills loop below re-creates the junction if still valid.
    $existingSkills = Get-ChildItem "$ClaudeHome\skills" -Directory -ErrorAction SilentlyContinue
    foreach ($existing in $existingSkills) {
        if (-not (Test-Path "$skillsSource\$($existing.Name)")) {
            if ($existing.Attributes -band [IO.FileAttributes]::ReparsePoint) {
                $existing.Delete()
            } else {
                Remove-Item $existing.FullName -Recurse -Force
            }
        }
    }
    # Copy current skills
    $skillDirs = Get-ChildItem $skillsSource -Directory -ErrorAction SilentlyContinue
    foreach ($skillDir in $skillDirs) {
        $targetDir = "$ClaudeHome\skills\$($skillDir.Name)"
        Ensure-Directory $targetDir
        Copy-Item "$($skillDir.FullName)\*" $targetDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    Write-Success "Synced skills to $ClaudeHome\skills\"
}

# Register MCP servers (requires Claude Code CLI, Node.js)
# Idempotent: server list lives in mcp-servers.json (SSOT shared with Linux);
# `claude mcp get` is used to skip already-registered entries, and `add` errors
# are surfaced rather than swallowed. BUG-011: every `claude mcp {get,add}`
# invocation is wrapped with Backup-AndRestoreClaudeJson because both subcommands
# hit the same #59870 deserialize-modify-serialize truncation path as
# `plugin install`.
$mcpConfig = "$DotfilesDir\mcp-servers.json"
$claudeCmd = Get-Command claude -ErrorAction SilentlyContinue
$npxCmd = Get-Command npx -ErrorAction SilentlyContinue
if (-not ($claudeCmd -and $npxCmd)) {
    Write-Warn "Claude Code CLI or npx not found, skipping MCP server registration"
} elseif (-not (Test-Path $mcpConfig)) {
    Write-Warn "mcp-servers.json not found at $mcpConfig, skipping MCP registration"
} else {
    Write-Info "Registering Claude Code MCP servers from $mcpConfig..."
    $mcpAdded = 0
    $mcpSkipped = 0
    $mcpFailed = 0
    try {
        $servers = (Get-Content $mcpConfig -Raw | ConvertFrom-Json).servers
        foreach ($srv in $servers) {
            if ($srv.prerequisite_binary) {
                $prereqCmd = Get-Command $srv.prerequisite_binary -ErrorAction SilentlyContinue
                if (-not $prereqCmd) {
                    Write-Warn "MCP $($srv.name): prerequisite '$($srv.prerequisite_binary)' not found, skipping"
                    $mcpFailed++
                    continue
                }
                if ($srv.prerequisite_command) {
                    $prereqParts = $srv.prerequisite_command -split '\s+'
                    & $prereqParts[0] @($prereqParts[1..($prereqParts.Length - 1)]) 2>&1 | Out-Null
                }
            }
            # BUG-011: wrap the idempotence-check `claude mcp get` with the same
            # guard used for install -- the CLI rewrites .claude.json on any
            # invocation. $LASTEXITCODE is automatic and survives the scriptblock.
            Backup-AndRestoreClaudeJson -Action {
                $null = & claude mcp get $srv.name 2>&1
            }
            if ($LASTEXITCODE -eq 0) {
                Write-Info "MCP $($srv.name) already registered, skipping"
                $mcpSkipped++
                continue
            }
            $argParts = $srv.args -split '\s+'
            # BUG-011: wrap `claude mcp add` -- the unwrapped call here was the
            # residual #59870 trigger after BUG-004 (PR #57) closed only the
            # plugin-install path.
            $mcpErr = Backup-AndRestoreClaudeJson -Action {
                & claude mcp add --transport $srv.transport $srv.name --scope user -- @argParts 2>&1
            }
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Registered MCP $($srv.name)"
                $mcpAdded++
            } else {
                Write-Warn "Failed to register MCP $($srv.name): $mcpErr"
                $mcpFailed++
            }
        }
        Write-Success "MCP servers: $mcpAdded added, $mcpSkipped already present, $mcpFailed failed"
    } catch {
        Write-Warn "Failed to register MCP servers: $_"
    }
}

# Claude Code plugins (requires claude CLI).
# Idempotent: cache the installed-plugins list ONCE before the loop and skip
# entries already present. CRITICAL: every `claude plugin install` writes to
# %USERPROFILE%\.claude\.claude.json. The CLI does NOT preserve all fields
# on rewrite -- subscription metadata (organizationType, organizationRateLimitTier),
# the projects map, and onboarding flags get silently dropped. Re-running
# install for already-installed plugins triggers silent .claude.json truncation
# and forces re-authentication in every project. Same idempotence pattern as
# MCP registration above. BUG-011: the pre-loop `claude plugin list` is now
# also wrapped because it goes through the same #59870 deserialize-modify-
# serialize path.
if ($claudeCmd) {
    Write-Info "Installing Claude Code plugins..."
    $plugins = @(
        "claude-mem@thedotmack",
        "code-simplifier@claude-plugins-official",
        "gopls-lsp@claude-plugins-official",
        "security-guidance@claude-plugins-official",
        "claude-md-management@claude-plugins-official",
        "claude-code-setup@claude-plugins-official",
        "frontend-design@claude-plugins-official",
        "ralph-loop@claude-plugins-official",
        "code-review@claude-plugins-official",
        "commit-commands@claude-plugins-official",
        "pr-review-toolkit@claude-plugins-official"
    )
    # BUG-014: register the `thedotmack` marketplace BEFORE the install loop.
    # Without this, `claude plugin install claude-mem@thedotmack` cannot resolve
    # `@thedotmack` (only `claude-plugins-official` is registered by default)
    # and fails silently inside the loop's try/catch. The CLI is idempotent
    # ("Marketplace 'thedotmack' already on disk" + exit 0 on re-run), so we
    # call it unconditionally. Wrapped per BUG-011 -- every `claude` CLI
    # invocation in setup must be snapshot-guarded.
    Backup-AndRestoreClaudeJson -Action {
        try {
            & claude plugin marketplace add thedotmack/claude-mem 2>$null | Out-Null
        } catch {
            # Network failure or transient CLI issue -- let the install loop fail
            # loud-but-recoverable rather than blocking the rest of setup.
        }
    }

    # BUG-011: wrap the read-only `claude plugin list` pre-fetch with the
    # snapshot guard -- the CLI still rewrites .claude.json on any invocation.
    $installedPlugins = Backup-AndRestoreClaudeJson -Action {
        try { (& claude plugin list 2>$null) -join "`n" } catch { "" }
    }
    $pluginsAdded = 0
    $pluginsSkipped = 0
    foreach ($plugin in $plugins) {
        if ($installedPlugins -match [regex]::Escape($plugin)) {
            $pluginsSkipped++
            continue
        }
        # BUG-004: wrap the install with the snapshot/restore guard so the upstream
        # truncation bug (#59870) cannot drop subscription state. The existing
        # `installedPlugins -match` idempotence above does NOT catch claude-mem
        # (it does not appear in `claude plugin list` output).
        Backup-AndRestoreClaudeJson -Action {
            try {
                & claude plugin install $plugin 2>$null | Out-Null
            } catch {
                # Silently continue if a plugin fails
            }
        }
        $pluginsAdded++
    }
    Write-Success "Claude Code plugins ready ($pluginsAdded added, $pluginsSkipped already present)"
} else {
    Write-Warn "Claude Code CLI not found, skipping plugin installation"
}

# Deploy auto-memory junctions from vault (see ADR-007)
# Junctions are bidirectional (like Linux symlinks) and require no admin privileges.
# Scans both 10_projects/ and 50_work/ for memory directories.
$VaultRoot = Join-Path $env:USERPROFILE "Projects\knowledge"
$VaultProjects = Join-Path $VaultRoot "10_projects"
if (Test-Path $VaultRoot) {
    Write-Info "Deploying auto-memory junctions from vault..."

    # Collect all memory/ dirs: 10_projects/* and recursive scan of 50_work/
    $memoryDirs = @()
    if (Test-Path $VaultProjects) {
        foreach ($projDir in (Get-ChildItem -Path $VaultProjects -Directory)) {
            $mem = Join-Path $projDir.FullName "memory"
            if (Test-Path $mem) { $memoryDirs += @{ Source = $mem; ProjectDir = $projDir.FullName; Scope = '10_projects' } }
        }
    }
    $VaultWork = Join-Path $VaultRoot "50_work"
    if (Test-Path $VaultWork) {
        Get-ChildItem -Path $VaultWork -Filter "memory" -Directory -Recurse | ForEach-Object {
            $memoryDirs += @{ Source = $_.FullName; ProjectDir = $_.Parent.FullName; Scope = '50_work' }
        }
    }

    foreach ($entry in $memoryDirs) {
        $memorySource = $entry.Source
        $projectDir = $entry.ProjectDir
        $projectName = Split-Path -Leaf $projectDir

        # Determine CWD path based on vault scope
        if ($entry.Scope -eq '10_projects') {
            # Convention: repo lives at ~/Projects/<name>
            $cwdPath = Join-Path $env:USERPROFILE "Projects\$projectName"
        } else {
            # Work projects: CWD is the vault path itself
            $cwdPath = $projectDir
        }

        $encodedPath = $cwdPath.Replace('\', '-').Replace(':', '')
        $targetDir = Join-Path $env:USERPROFILE ".claude\projects\$encodedPath\memory"
        $parentDir = Split-Path $targetDir -Parent

        Ensure-Directory $parentDir

        # Handle existing target: junction=recreate, real dir with files=backup, empty=remove
        if (Test-Path $targetDir) {
            $item = Get-Item $targetDir -Force
            if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) {
                # cmd rmdir removes the junction point without deleting target contents
                cmd /c rmdir $targetDir 2>&1 | Out-Null
            } elseif ((Get-ChildItem $targetDir -ErrorAction SilentlyContinue).Count -gt 0) {
                Write-Warn "Backing up existing memory for $projectName"
                Rename-Item $targetDir "$($targetDir).bak.$(Get-Date -Format 'yyyyMMddHHmmss')"
            } else {
                Remove-Item $targetDir -Recurse -Force -ErrorAction SilentlyContinue
            }
        }

        New-Item -ItemType Junction -Path $targetDir -Target $memorySource -Force | Out-Null
        Write-Success "Linked auto-memory: $projectName"
    }

    # Migrate orphan memories: local Claude Code memories not yet in vault
    $claudeProjects = Join-Path $env:USERPROFILE ".claude\projects"
    if (Test-Path $claudeProjects) {
        $claudeDirs = Get-ChildItem -Path $claudeProjects -Directory
        foreach ($cpDir in $claudeDirs) {
            $memDir = Join-Path $cpDir.FullName "memory"
            if (-not (Test-Path $memDir)) { continue }
            if ((Get-Item $memDir).Attributes -band [IO.FileAttributes]::ReparsePoint) { continue }
            $files = Get-ChildItem -Path $memDir -ErrorAction SilentlyContinue
            if (-not $files) { continue }

            $encodedName = $cpDir.Name
            if ($encodedName -match 'Projects-(.+)$') {
                $projectName = $Matches[1]
                $vaultMemory = Join-Path $VaultProjects "$projectName\memory"
                $vaultProject = Join-Path $VaultProjects $projectName
                if ((Test-Path $vaultProject) -and -not (Test-Path $vaultMemory)) {
                    Write-Info "Migrating orphan memory: $projectName -> vault"
                    Copy-Item $memDir $vaultMemory -Recurse -Force
                    Remove-Item $memDir -Recurse -Force
                    New-Item -ItemType Junction -Path $memDir -Target $vaultMemory -Force | Out-Null
                    Write-Success "Migrated and linked: $projectName"
                }
            }
        }
    }
}

# Deploy vault-hosted skill junctions (vault -> Claude Code)
# Vault skills live in $VaultRoot\00_meta\skills\<name>\SKILL.md per pattern-spec-driven-development.
# Junction each into $env:USERPROFILE\.claude\skills\ for Claude Code discovery. Idempotent.
$VaultSkillsDir = Join-Path $VaultRoot "00_meta\skills"
if (Test-Path $VaultSkillsDir) {
    Write-Info "Linking vault-hosted skills to Claude Code..."
    Get-ChildItem -Path $VaultSkillsDir -Directory -ErrorAction SilentlyContinue | ForEach-Object {
        $skillSource = $_.FullName
        $skillName = $_.Name
        $target = Join-Path $ClaudeHome "skills\$skillName"

        if (Test-Path $target) {
            $item = Get-Item $target -Force
            if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) {
                cmd /c rmdir $target 2>&1 | Out-Null
            } elseif ((Get-ChildItem $target -ErrorAction SilentlyContinue).Count -gt 0) {
                $backup = "$target.bak.$(Get-Date -Format 'yyyyMMddHHmmss')"
                Write-Warn "$skillName exists at $target - backing up to $backup"
                Rename-Item $target $backup
            } else {
                Remove-Item $target -Recurse -Force -ErrorAction SilentlyContinue
            }
        }

        New-Item -ItemType Junction -Path $target -Target $skillSource -Force | Out-Null
        Write-Success "Linked vault skill: $skillName"
    }
}

# ============================================================================
# 2b. PYTHON TOOLING (uv + poetry)
# ============================================================================
# uv is required by the hive MCP server (uvx hive-vault). poetry is general
# Python tooling. Aider was removed in chore/aider-sunset-full (2026-05-16);
# OpenCode integration on Windows is a separate follow-up (admin-conditional,
# not automated here).

# Install uv (Python package manager -- provides uvx)
$uvCmd = Get-Command uv -ErrorAction SilentlyContinue
if (-not $uvCmd) {
    Write-Info "Installing uv..."
    try {
        $uvInstaller = Join-Path $env:TEMP "uv-install.ps1"
        Invoke-RestMethod https://astral.sh/uv/install.ps1 -OutFile $uvInstaller
        & $uvInstaller 2>$null
        Remove-Item -Path $uvInstaller -Force -ErrorAction SilentlyContinue
        # Refresh PATH for current session
        $env:PATH = "$env:USERPROFILE\.local\bin;$env:PATH"
        $uvCmd = Get-Command uv -ErrorAction SilentlyContinue
        if ($uvCmd) {
            Write-Success "uv installed"
        } else {
            Write-Warn "uv installation failed"
        }
    } catch {
        Write-Warn "Failed to install uv: $_"
    }
} else {
    Write-Info "uv already installed"
}

# Install poetry via uv
$poetryCmd = Get-Command poetry -ErrorAction SilentlyContinue
if (-not $poetryCmd) {
    $uvCmd = Get-Command uv -ErrorAction SilentlyContinue
    if ($uvCmd) {
        Write-Info "Installing poetry via uv..."
        try {
            & uv tool install poetry 2>$null
            Write-Success "Poetry installed"
        } catch {
            Write-Warn "Failed to install poetry: $_"
        }
    } else {
        Write-Warn "uv not available, skipping poetry installation"
    }
} else {
    Write-Info "poetry already installed"
}

# ============================================================================
# 2c. OBSIDIAN CLI (BUG-013)
# ============================================================================
# @vorillaz/obsidian-cli provides the `obsidian` binary used by obs-cli.ps1 and
# the vault-health workflow. npm global install writes to %APPDATA%\npm
# (user-writable, no admin). Idempotent: skip if `obsidian` already on PATH.
# Guarded on `npm` availability so machines without Node.js gracefully skip.

$obsidianCmd = Get-Command obsidian -ErrorAction SilentlyContinue
if (-not $obsidianCmd) {
    $npmCmd = Get-Command npm -ErrorAction SilentlyContinue
    if ($npmCmd) {
        Write-Info "Installing Obsidian CLI (@vorillaz/obsidian-cli) via npm..."
        try {
            & npm install -g '@vorillaz/obsidian-cli' 2>$null | Out-Null
            # Refresh PATH so the freshly-installed binary is visible in this
            # session (same trick as the winget block in section 1c).
            $env:PATH = [Environment]::GetEnvironmentVariable("PATH", "Machine") + ";" + [Environment]::GetEnvironmentVariable("PATH", "User")
            if (Get-Command obsidian -ErrorAction SilentlyContinue) {
                Write-Success "Obsidian CLI installed"
            } else {
                Write-Warn "Obsidian CLI install completed but binary not on PATH (restart shell)"
            }
        } catch {
            Write-Warn "Failed to install Obsidian CLI: $_"
        }
    } else {
        Write-Warn "npm not available, skipping Obsidian CLI install (install Node.js then re-run)"
    }
} else {
    Write-Info "Obsidian CLI already installed at $($obsidianCmd.Source)"
}

# ============================================================================
# 3. DEPLOY GEMINI CONFIGURATION (config deploy is always safe)
# ============================================================================

Write-Info "Deploying Gemini configuration..."

Ensure-Directory $GeminiHome
Ensure-Directory "$GeminiHome\prompts"

# Bulk copy all Gemini config files
$geminiSource = "$DotfilesDir\ai\gemini"
if (Test-Path $geminiSource) {
    Copy-Item "$geminiSource\*" "$GeminiHome\" -Recurse -Force -ErrorAction SilentlyContinue
}

# Force copy GEMINI.md (Neural Hive Protocol)
$geminiMdSource = "$DotfilesDir\ai\gemini\GEMINI.md"
if (Test-Path $geminiMdSource) {
    Copy-Item $geminiMdSource "$GeminiHome\" -Force
    if (Select-String -Path "$GeminiHome\GEMINI.md" -Pattern 'First, read `AGENTS.md`' -SimpleMatch -Quiet) {
        Write-Success "GEMINI.md deployed successfully (verified pointer to AGENTS.md)"
    } else {
        Write-Err "GEMINI.md deployment failed verification (expected pointer to AGENTS.md)"
    }
} else {
    Write-Warn "GEMINI.md not found at $geminiMdSource"
}

# Sync Gemini prompts: remove stale, then extract from current skills
if (Test-Path $skillsSource) {
    # Remove stale skill-derived prompts not in source
    $existingPrompts = Get-ChildItem "$GeminiHome\prompts\*.md" -ErrorAction SilentlyContinue
    foreach ($prompt in $existingPrompts) {
        if (-not (Test-Path "$skillsSource\$($prompt.BaseName)")) {
            Remove-Item $prompt.FullName -Force
        }
    }
    # Extract SKILL.md content as flat prompts (strip YAML frontmatter)
    $skillDirs = Get-ChildItem $skillsSource -Directory -ErrorAction SilentlyContinue
    foreach ($skillDir in $skillDirs) {
        $skillMd = "$($skillDir.FullName)\SKILL.md"
        if (Test-Path $skillMd) {
            $content = Get-Content $skillMd -Raw

            # Strip YAML frontmatter (content between --- markers)
            if ($content -match '^---\r?\n[\s\S]*?\r?\n---\r?\n(.*)$') {
                $strippedContent = $Matches[1]
            } else {
                $strippedContent = $content
            }

            $targetFile = "$GeminiHome\prompts\$($skillDir.Name).md"
            Set-Content -Path $targetFile -Value $strippedContent.Trim() -Encoding UTF8
        }
    }
    Write-Success "Synced Gemini prompts to $GeminiHome\prompts\"
}

# ============================================================================
# 4. DEPLOY POWERSHELL PROFILE
# ============================================================================

Write-Info "Setting up PowerShell profile..."

$profileSource = "$DotfilesDir\powershell\profile.ps1"
$profileTarget = $PROFILE

if (Test-Path $profileSource) {
    # Ensure profile directory exists
    $profileDir = Split-Path -Parent $profileTarget
    Ensure-Directory $profileDir

    # Read source profile
    $sourceContent = Get-Content $profileSource -Raw

    # Marker for dotfiles section
    $startMarker = "# >>> DOTFILES PROFILE >>>"
    $endMarker = "# <<< DOTFILES PROFILE <<<"

    if (Test-Path $profileTarget) {
        $existingContent = Get-Content $profileTarget -Raw

        # Check if our section already exists
        if ($existingContent -match [regex]::Escape($startMarker)) {
            # Replace existing section
            $pattern = [regex]::Escape($startMarker) + "[\s\S]*?" + [regex]::Escape($endMarker)
            $newSection = "$startMarker`r`n$sourceContent`r`n$endMarker"
            $newContent = $existingContent -replace $pattern, $newSection
            Set-Content -Path $profileTarget -Value $newContent -Encoding UTF8
            Write-Success "Updated dotfiles section in PowerShell profile"
        } else {
            # Append our section
            $appendContent = "`r`n`r`n$startMarker`r`n$sourceContent`r`n$endMarker"
            Add-Content -Path $profileTarget -Value $appendContent -Encoding UTF8
            Write-Success "Appended dotfiles section to PowerShell profile"
        }
    } else {
        # Create new profile with our section
        $newContent = "$startMarker`r`n$sourceContent`r`n$endMarker"
        Set-Content -Path $profileTarget -Value $newContent -Encoding UTF8
        Write-Success "Created PowerShell profile at $profileTarget"
    }
} else {
    Write-Warn "Profile template not found at $profileSource"
}

# ============================================================================
# 5. DEPLOY GIT CONFIGURATION
# ============================================================================

Write-Info "Setting up Git configuration..."

$gitconfigSource = "$DotfilesDir\.gitconfig"
$gitconfigTarget = "$env:USERPROFILE\.gitconfig"

if (Test-Path $gitconfigSource) {
    # Check if target already exists
    if (Test-Path $gitconfigTarget) {
        Write-Warn ".gitconfig already exists at $gitconfigTarget"
        Write-Info "Skipping to avoid overwriting existing configuration"
        Write-Info "To update manually: Copy-Item '$gitconfigSource' '$gitconfigTarget' -Force"
    } else {
        Copy-Item $gitconfigSource $gitconfigTarget -Force
        Write-Success "Deployed .gitconfig"
    }
} else {
    Write-Warn ".gitconfig not found at $gitconfigSource"
}

# tmux: intentionally skipped on Windows (Linux-only -- see tmux.conf in repo root)

# ============================================================================
# 5b. SSH CONFIG
# ============================================================================

Write-Info "Setting up SSH config..."

$sshDir = "$env:USERPROFILE\.ssh"
Ensure-Directory $sshDir

$sshConfigSource = "$DotfilesDir\ssh\config"
if (Test-Path $sshConfigSource) {
    Copy-Item $sshConfigSource "$sshDir\config" -Force
    Write-Success "Deployed SSH config"
} else {
    Write-Warn "SSH config not found at $sshConfigSource"
}

$sshPubKeySource = "$DotfilesDir\ssh\id_ed25519.pub"
if (Test-Path $sshPubKeySource) {
    if (-not (Test-Path "$sshDir\id_ed25519.pub")) {
        Copy-Item $sshPubKeySource "$sshDir\id_ed25519.pub"
        Write-Success "Deployed SSH public key"
    } else {
        Write-Info "SSH public key already exists, skipping"
    }
} else {
    Write-Warn "SSH public key not found at $sshPubKeySource"
}

# ============================================================================
# 6. ADD SCRIPTS TO PATH
# ============================================================================

Write-Info "Configuring PATH..."

$currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")

if ($currentPath -notlike "*$ScriptsDir*") {
    $newPath = "$ScriptsDir;$currentPath"
    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    Write-Success "Added $ScriptsDir to User PATH"
    Write-Info "Restart PowerShell for PATH changes to take effect"
} else {
    Write-Info "Scripts directory already in PATH"
}

# ============================================================================
# 7. COPY SCRIPTS TO USER SCRIPTS FOLDER
# ============================================================================

Write-Info "Deploying scripts..."

$initProjectSource = "$DotfilesDir\scripts\init-project.ps1"
if (Test-Path $initProjectSource) {
    Copy-Item $initProjectSource "$ScriptsDir\" -Force
    Copy-Item $initProjectSource "$ClaudeHome\" -Force
    Write-Success "Deployed init-project.ps1 to $ScriptsDir\ and $ClaudeHome\"
} else {
    Write-Warn "init-project.ps1 not found at $initProjectSource"
}

$crystallizeSource = "$DotfilesDir\scripts\knowledge-crystallize.ps1"
if (Test-Path $crystallizeSource) {
    Copy-Item $crystallizeSource "$ScriptsDir\" -Force
    Write-Success "Deployed knowledge-crystallize.ps1 to $ScriptsDir\"
} else {
    Write-Warn "knowledge-crystallize.ps1 not found at $crystallizeSource"
}

$sessionStartSource = "$DotfilesDir\scripts\claude-session-start.ps1"
if (Test-Path $sessionStartSource) {
    Copy-Item $sessionStartSource "$ScriptsDir\" -Force
    Write-Success "Deployed claude-session-start.ps1 to $ScriptsDir\"
} else {
    Write-Warn "claude-session-start.ps1 not found at $sessionStartSource"
}

$memHealSource = "$DotfilesDir\scripts\claude-mem-heal.ps1"
if (Test-Path $memHealSource) {
    Copy-Item $memHealSource "$ScriptsDir\" -Force
    Write-Success "Deployed claude-mem-heal.ps1 to $ScriptsDir\"
} else {
    Write-Warn "claude-mem-heal.ps1 not found at $memHealSource"
}

$doctorSource = "$DotfilesDir\scripts\doctor.ps1"
if (Test-Path $doctorSource) {
    Copy-Item $doctorSource "$ScriptsDir\" -Force
    Write-Success "Deployed doctor.ps1 to $ScriptsDir\"
} else {
    Write-Warn "doctor.ps1 not found at $doctorSource"
}

# WIN-001 follow-up: PR #71 added the post-setup invocation in section 8d but
# omitted the deploy. Without this Copy-Item, $ScriptsDir\healthcheck.ps1 never
# exists and section 8d silently warns "not deployed, skipping" on every run.
$healthcheckSource = "$DotfilesDir\scripts\healthcheck.ps1"
if (Test-Path $healthcheckSource) {
    Copy-Item $healthcheckSource "$ScriptsDir\" -Force
    Write-Success "Deployed healthcheck.ps1 to $ScriptsDir\"
} else {
    Write-Warn "healthcheck.ps1 not found at $healthcheckSource"
}

$contractSource = "$DotfilesDir\env-contract.json"
if (Test-Path $contractSource) {
    Copy-Item $contractSource "$DotfilesDest\" -Force
    Write-Success "Deployed env-contract.json to $DotfilesDest\"
} else {
    Write-Warn "env-contract.json not found at $contractSource"
}

$syncSource = "$DotfilesDir\scripts\dotfiles-sync.ps1"
if (Test-Path $syncSource) {
    Copy-Item $syncSource "$ScriptsDir\" -Force
    Write-Success "Deployed dotfiles-sync.ps1 to $ScriptsDir\"
} else {
    Write-Warn "dotfiles-sync.ps1 not found at $syncSource"
}

$obsCliSource = "$DotfilesDir\scripts\obs-cli.ps1"
if (Test-Path $obsCliSource) {
    Copy-Item $obsCliSource "$ScriptsDir\" -Force
    Write-Success "Deployed obs-cli.ps1 to $ScriptsDir\"
} else {
    Write-Warn "obs-cli.ps1 not found at $obsCliSource"
}

$loadSecretsSource = "$DotfilesDir\scripts\load-secrets.ps1"
if (Test-Path $loadSecretsSource) {
    Ensure-Directory "$DotfilesDest\scripts"
    Copy-Item $loadSecretsSource "$DotfilesDest\scripts\" -Force
    Write-Success "Deployed load-secrets.ps1 to $DotfilesDest\scripts\"
    Write-Info "To load secrets at startup, add to your PowerShell profile:"
    Write-Info "  . `"$DotfilesDest\scripts\load-secrets.ps1`""
} else {
    Write-Warn "load-secrets.ps1 not found at $loadSecretsSource"
}

# ============================================================================
# 7b. DEPLOY SECRETS SYSTEM
# ============================================================================

Write-Info "Setting up secrets system..."

$sensitiveSource = "$DotfilesDir\sensitive"
$sensitiveDest = "$DotfilesDest\sensitive"

if (Test-Path $sensitiveSource) {
    Ensure-Directory $sensitiveDest

    # Copy env-mapping.conf
    $mappingSource = "$sensitiveSource\env-mapping.conf"
    if (Test-Path $mappingSource) {
        Copy-Item $mappingSource "$sensitiveDest\" -Force
        Write-Success "Deployed env-mapping.conf"
    }

    # Copy all .secret.age files
    $ageFiles = Get-ChildItem -Path $sensitiveSource -Filter '*.secret.age' -ErrorAction SilentlyContinue
    if ($ageFiles) {
        foreach ($ageFile in $ageFiles) {
            Copy-Item $ageFile.FullName "$sensitiveDest\" -Force
        }
        Write-Success "Deployed $($ageFiles.Count) encrypted secret files"
    }
} else {
    Write-Warn "Sensitive directory not found at $sensitiveSource"
}

# ============================================================================
# 7c. REGISTER SESSIONSTART HOOK
# ============================================================================

Write-Info "Applying Claude settings.json template + registering SessionStart hook..."

# SDD-002 (PR #51): single source of truth for the "dotfiles-owned" subset of
# settings.json lives at ai/claude/settings.json. The previous inline hashtable
# for the hook entry is gone -- Merge-ClaudeSettings reads the template,
# substitutes __HOOK_COMMAND__, and applies the per-key policy. Bootstraps a
# fresh settings.json if missing (closes the v1 doble-paso friction).
$ClaudeSettings = "$ClaudeHome\settings.json"
$ClaudeSettingsTemplate = "$DotfilesDir\ai\claude\settings.json"
$sessionStartCmd = "$ScriptsDir\claude-session-start.ps1"
$expectedHookCommand = "pwsh -NoProfile -File `"$sessionStartCmd`""

Merge-ClaudeSettings -TemplatePath $ClaudeSettingsTemplate -TargetPath $ClaudeSettings -HookCommand $expectedHookCommand

# ============================================================================
# 8. GITHUB COPILOT CLI
# ============================================================================

Write-Info "Setting up GitHub Copilot CLI..."

# BUG-003: detect the new standalone `copilot` CLI (winget GitHub.Copilot,
# agentic interface, closer to Claude Code than to the legacy gh-copilot
# extension's suggest/explain wrappers). The dev tools winget block above
# auto-installs it; this block deploys config when the binary is on PATH.
# Note: AWS Copilot CLI (Amazon.CopilotCLI) also exposes itself as `copilot`.
# If both are installed, Get-Command resolves to the first on PATH. Out-of-
# scope to disambiguate here; <1% population.
$copilotCmd = Get-Command copilot -ErrorAction SilentlyContinue
if ($copilotCmd) {
    Write-Info "GitHub Copilot CLI detected at $($copilotCmd.Source), deploying configuration..."
    $CopilotHome = "$env:USERPROFILE\.copilot"
    Ensure-Directory $CopilotHome

    $copilotSource = "$DotfilesDir\ai\copilot"
    if (Test-Path $copilotSource) {
        Copy-Item "$copilotSource\*" "$CopilotHome\" -Recurse -Force -ErrorAction SilentlyContinue
        if ((Test-Path "$CopilotHome\copilot-instructions.md") -and
            (Select-String -Path "$CopilotHome\copilot-instructions.md" -Pattern 'First, read `AGENTS.md`' -SimpleMatch -Quiet)) {
            Write-Success "copilot-instructions.md deployed successfully (verified pointer to AGENTS.md)"
        } else {
            Write-Warn "copilot-instructions.md deployment failed verification (expected pointer to AGENTS.md)"
        }
    }

    Write-Success "GitHub Copilot CLI configured (aliases cop/cops in profile.ps1)"
} else {
    Write-Info "GitHub Copilot CLI not installed; the dev tools block above attempts auto-install via winget GitHub.Copilot. Re-run setup or open a new shell if the binary was just installed and PATH needs refresh."
}

# Weekly vault maintenance scheduled task (Sundays 10:07 AM)
# Self-healing: compare existing action arguments against expected and rewrite
# when they diverge -- guards against stale tasks pointing at moved/renamed scripts.
Write-Info "Setting up weekly vault maintenance task..."
$taskName = "DotfilesVaultMaintenance"
$expectedTaskScript = "$DotfilesDir\scripts\vault-maintenance-weekly.ps1"
$expectedTaskArgument = "-NoProfile -ExecutionPolicy Bypass -File `"$expectedTaskScript`""
$existingTask = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
$existingTaskArgument = $null
if ($existingTask -and $existingTask.Actions -and $existingTask.Actions[0]) {
    $existingTaskArgument = $existingTask.Actions[0].Arguments
}
if ($existingTask -and ($existingTaskArgument -eq $expectedTaskArgument)) {
    Write-Info "Weekly vault maintenance task already correctly configured, skipping"
} else {
    if ($existingTask) {
        Write-Info "Weekly vault maintenance task arguments drifted ('$existingTaskArgument' != expected); re-registering"
    }
    try {
        $action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument $expectedTaskArgument
        $trigger = New-ScheduledTaskTrigger -Weekly -DaysOfWeek Sunday -At "10:07AM"
        Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger `
            -Description "Weekly vault maintenance - knowledge crystallization and health checks" -Force | Out-Null
        Write-Success "Installed weekly vault maintenance task (Sundays 10:07)"
        if (-not (Test-Path $expectedTaskScript)) {
            Write-Warn "Task registered but target script does not exist: $expectedTaskScript"
        }
    } catch {
        Write-Warn "Could not register scheduled task (may need admin): $_"
    }
}

# ============================================================================
# 8c. POST-SETUP DOCTOR
# ============================================================================
# Final assertion against env-contract.json -- catches drift between what
# setup just deployed and what's actually in place / on PATH / in env vars.
#
# Pre-export the REFACTOR-002 path vars so doctor sees what the deployed
# profile.ps1 WILL set on the next pwsh session. Without this, every fresh
# setup run reports 4 false warnings because `& pwsh -NoProfile` skips the
# deployed profile entirely. Values must match the corresponding lines in
# powershell/profile.ps1. PowerShell propagates parent $env: to child procs.

if (-not $env:SCRIPTS_DIR)   { $env:SCRIPTS_DIR   = "$env:DOTFILES_DIR\scripts" }
if (-not $env:GEMINI_HOME)   { $env:GEMINI_HOME   = "$env:USERPROFILE\.gemini" }
if (-not $env:COPILOT_HOME)  { $env:COPILOT_HOME  = "$env:USERPROFILE\.copilot" }
if (-not $env:OPENCODE_HOME) { $env:OPENCODE_HOME = "$env:USERPROFILE\.config\opencode" }

$doctorScript = "$ScriptsDir\doctor.ps1"
if (Test-Path $doctorScript) {
    Write-Info "Running post-setup doctor check..."
    Write-Host ""
    & pwsh -NoProfile -File $doctorScript
    $doctorExit = $LASTEXITCODE
    Write-Host ""
    if ($doctorExit -ne 0) {
        Write-Warn "doctor reported one or more required items missing (exit $doctorExit) -- review output above"
    }
} else {
    Write-Warn "doctor.ps1 not deployed at $doctorScript, skipping post-setup check"
}

# ============================================================================
# 8d. POST-SETUP HEALTHCHECK (WIN-001)
# ============================================================================
# Full structural health check after doctor. Non-fatal: surfaces deploy gaps
# but does NOT alter setup's $LASTEXITCODE. Linux setup-linux.sh does not
# auto-invoke healthcheck.sh today; that parity is tracked by WIN-001b.

$healthcheckScript = "$ScriptsDir\healthcheck.ps1"
if (Test-Path $healthcheckScript) {
    Write-Info "Running post-setup healthcheck..."
    Write-Host ""
    & pwsh -NoProfile -File $healthcheckScript
    $healthcheckExit = $LASTEXITCODE
    Write-Host ""
    if ($healthcheckExit -ne 0) {
        Write-Warn "healthcheck reported one or more FAIL items (exit $healthcheckExit) -- review output above; use 'hc' alias to re-run"
    }
} else {
    Write-Warn "healthcheck.ps1 not deployed at $healthcheckScript, skipping post-setup check"
}

# ============================================================================
# 9. SUMMARY
# ============================================================================

Write-Host ""
Write-Host "============================================" -ForegroundColor Green
Write-Host "  Setup Complete!                          " -ForegroundColor Green
Write-Host "============================================" -ForegroundColor Green
Write-Host ""
Write-Host "Deployed:" -ForegroundColor Cyan
Write-Host "  - Claude config:  $ClaudeHome\CLAUDE.md"
Write-Host "  - Claude skills:  $ClaudeHome\skills\"
Write-Host "  - Gemini config:  $GeminiHome\GEMINI.md"
Write-Host "  - Gemini prompts: $GeminiHome\prompts\"
Write-Host "  - Scripts:        $ScriptsDir\"
Write-Host "  - Secrets:        $DotfilesDest\sensitive\"
Write-Host "  - Copilot config: $env:USERPROFILE\.copilot\"
Write-Host "  - Profile:        $profileTarget"
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Restart PowerShell to load the new profile"
Write-Host "  2. Verify setup:"
Write-Host "       Test-Path `"$ClaudeHome\CLAUDE.md`""
Write-Host "       Test-Path `"$GeminiHome\GEMINI.md`""
Write-Host "       `$env:PATH -like `"*scripts*`""
Write-Host "  3. Initialize a project:"
Write-Host "       project-init test-project python"
Write-Host ""
