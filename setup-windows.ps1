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
param(
    # WIN-005: opt-in HKCU engineering defaults (scripts/windows-defaults.ps1).
    # OFF by default -- mass-setting user preferences without explicit consent
    # violates user autonomy (proposal R5).
    [switch]$WithDefaults
)

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
# re-exec under it; otherwise fail loud with an install hint. Named parameters
# bound by param() are NOT in @args, so each one must be forwarded explicitly
# (WIN-005 added the first one).

if ($PSVersionTable.PSVersion.Major -lt 7) {
    $pwshCmd = Get-Command pwsh -ErrorAction SilentlyContinue
    if ($pwshCmd) {
        Write-Host "[INFO] Windows PowerShell $($PSVersionTable.PSVersion) detected; re-executing under pwsh ($($pwshCmd.Source)) for full feature compatibility (BUG-005)" -ForegroundColor Yellow
        $forward = @()
        if ($WithDefaults) { $forward += '-WithDefaults' }
        & $pwshCmd.Source -NoProfile -ExecutionPolicy Bypass -File $PSCommandPath @forward @args
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

function Test-VersionAtLeast {
    # versions.conf pins are MINIMUMS (REFACTOR-013): an install gate must
    # upgrade when installed < pin and leave a NEWER install untouched (an
    # exact-match `-ne` reconcile would downgrade it). Returns $true when
    # $Installed >= $Minimum. Empty minimum -> $true (no pin); empty installed
    # -> $false (needs install). Non-semver strings fall back to string equality
    # so a tag that does not parse as [version] never throws.
    param([string]$Installed, [string]$Minimum)
    if ([string]::IsNullOrWhiteSpace($Minimum)) { return $true }
    if ([string]::IsNullOrWhiteSpace($Installed)) { return $false }
    try {
        return ([version]$Installed -ge [version]$Minimum)
    } catch {
        return ($Installed -eq $Minimum)
    }
}

function Ensure-Directory {
    param([string]$Path)
    if (-not (Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

# SDD-007: dot-source shared deploy helpers (Deploy-File / Test-FileDrift).
# Same SHA256-based idempotent copy pattern previously inlined throughout this
# script; centralizing in utils.ps1 keeps cross-OS parity with scripts/utils.sh.
$utilsPs1Path = Join-Path $PSScriptRoot 'scripts\utils.ps1'
if (Test-Path -LiteralPath $utilsPs1Path -PathType Leaf) {
    . $utilsPs1Path
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

# ADR-015 / dotfiles#230: single registration point for hive Scheduled Tasks so
# they ALWAYS run with the strongest achievable principal. A task registered
# without an explicit principal defaults to the Interactive logon type, which
# runs in the user's desktop session and pops a console window every tick for a
# powershell.exe action. An S4U principal ("run whether the user is logged on
# or not", no stored password) runs the task in session 0 -- the non-interactive
# session with no desktop -- so no window can appear. BUT registering an S4U
# task requires an elevated caller: on a non-admin box Register-ScheduledTask
# raises "Access is denied" as a NON-terminating error, which used to slip past
# every call site's try/catch and print a false SUCCESS while a stale task
# survived. So: compute the strongest achievable logon type once (elevated ->
# S4U windowless; non-admin -> Interactive, where -WindowStyle Hidden on the
# action softens the console flash), and register with -ErrorAction Stop so any
# residual failure is terminating and the call-site catch reports the truth.
$script:HiveTaskLogonType =
    if (([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { 'S4U' } else { 'Interactive' }
function Register-HiveScheduledTask {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$TaskName,
        [Parameter(Mandatory)]$Action,
        [Parameter(Mandatory)]$Trigger,
        [Parameter(Mandatory)]$Settings,
        [Parameter(Mandatory)][string]$Description
    )
    $principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType $script:HiveTaskLogonType -RunLevel Limited
    Register-ScheduledTask -TaskName $TaskName -Action $Action -Trigger $Trigger `
        -Settings $Settings -Principal $principal -Description $Description -Force -ErrorAction Stop | Out-Null
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
        [Parameter(Mandatory)][string]$HookCommand,
        [Parameter(Mandatory)][string]$SessionEndCommand
    )

    if (-not (Test-Path $TemplatePath)) {
        Write-Warn "Claude settings template not found at $TemplatePath, skipping merge"
        return
    }

    # Read template, JSON-escape the hook command, substitute __HOOK_COMMAND__
    $escapedCommand = ($HookCommand -replace '\\', '\\') -replace '"', '\"'
    $escapedEndCommand = ($SessionEndCommand -replace '\\', '\\') -replace '"', '\"'
    $templateRaw = (Get-Content $TemplatePath -Raw) -replace '__HOOK_COMMAND__', $escapedCommand -replace '__SESSION_END_COMMAND__', $escapedEndCommand
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
    if ($template.ContainsKey('hooks') -and $template['hooks'].ContainsKey('SessionEnd')) {
        if (-not $existing.ContainsKey('hooks')) { $existing['hooks'] = @{} }
        $existing['hooks']['SessionEnd'] = $template['hooks']['SessionEnd']
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

# Pin OpenCode to OPENCODE_VERSION from versions.conf (REFACTOR-011), mirroring
# the Linux installer. winget honors --version. If the manifest is missing or
# the key is absent, $opencodeVersion stays $null and OpenCode installs latest.
$opencodeVersion = $null
if (Test-Path -LiteralPath $versionsSource) {
    foreach ($line in Get-Content -LiteralPath $versionsSource) {
        if ($line -match '^\s*OPENCODE_VERSION\s*=\s*(.+?)\s*$') {
            $opencodeVersion = $Matches[1].Trim().Trim('"').Trim("'")
            break
        }
    }
}

$wingetCmd = Get-Command winget -ErrorAction SilentlyContinue
if ($wingetCmd) {
    $tools = @(
        @{ Name = "age"; Cmd = "age"; Id = "FiloSottile.age" },
        @{ Name = "eza"; Cmd = "eza"; Id = "eza-community.eza" },
        @{ Name = "jq"; Cmd = "jq"; Id = "jqlang.jq" },
        @{ Name = "GitHub CLI"; Cmd = "gh"; Id = "GitHub.cli" },
        @{ Name = "zoxide"; Cmd = "zoxide"; Id = "ajeetdsouza.zoxide" },
        @{ Name = "GitHub Copilot CLI"; Cmd = "copilot"; Id = "GitHub.Copilot" },
        # AI-014: OpenCode (secondary AI agent, ADR-009). User-scope winget
        # package, no admin. Cleaner than the Linux curl-bash equivalent.
        @{ Name = "OpenCode"; Cmd = "opencode"; Id = "SST.opencode"; Version = $opencodeVersion }
    )
    foreach ($tool in $tools) {
        if (-not (Get-Command $tool.Cmd -ErrorAction SilentlyContinue)) {
            Write-Info "Installing $($tool.Name)..."
            try {
                $wingetArgs = @($tool.Id, '--accept-package-agreements', '--accept-source-agreements')
                # ContainsKey guard: utils.ps1 sets Set-StrictMode Latest, so
                # $tool.Version on a hashtable WITHOUT that key throws instead
                # of returning $null -- latent until a box actually needs the
                # install branch (first caught by the WIN-004 CI runner).
                if ($tool.ContainsKey('Version') -and $tool.Version) { $wingetArgs += @('--version', $tool.Version) }
                & winget install @wingetArgs 2>$null | Out-Null
                Write-Success "$($tool.Name) installed"
            } catch {
                Write-Warn "Failed to install $($tool.Name): $_"
            }
        } elseif ($tool.ContainsKey('Version') -and $tool.Version) {
            # REFACTOR-011/013: presence is not convergence, and the pin is a
            # MINIMUM. A tool OLDER than the pin was skipped as "already
            # installed"; an exact-match `-ne` reconcile would conversely
            # DOWNGRADE a newer install. Upgrade only when installed < pin.
            $installedVer = ((& $tool.Cmd --version 2>&1 | Select-Object -First 1) -split '\s+')[-1]
            if (-not (Test-VersionAtLeast $installedVer $tool.Version)) {
                Write-Info "$($tool.Name) $installedVer below pinned minimum $($tool.Version), upgrading..."
                & winget install $tool.Id --version $tool.Version --accept-package-agreements --accept-source-agreements --force 2>$null | Out-Null
                # Re-query instead of trusting winget's exit code: when the
                # PATH-resolved binary comes from another package manager (npm
                # global, scoop), winget converges its own copy but the
                # shadowing install keeps winning -- that is not convergence.
                $postVer = ((& $tool.Cmd --version 2>&1 | Select-Object -First 1) -split '\s+')[-1]
                if (Test-VersionAtLeast $postVer $tool.Version) {
                    Write-Success "$($tool.Name) upgraded to $postVer (>= $($tool.Version))"
                } else {
                    Write-Warn "$($tool.Name) still $postVer after winget install $($tool.Version): another install shadows it in PATH (e.g. npm global, scoop); converge that install instead"
                }
            } else {
                Write-Info "$($tool.Name) already installed ($installedVer >= $($tool.Version))"
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

# Skills (claude, opencode, agy) are deployed from the committed vault skill
# records by Deploy-SkillRecord near the end of this script (SDD-008, option A),
# replacing the former ai\skills copy loops. Deploy is a regular copy that
# de-junctions any pre-existing vault junction first (BUG-100), honoring each
# skill's targets[].
Ensure-Directory "$ClaudeHome\skills"

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
        # HIVE-118: migrate a stale `uvx hive-vault` entry by REMOVING it here so
        # the skip-if-present loop below re-adds the current definition
        # (`hive client`) from mcp-servers.json. Remove-only -- the loop owns every
        # `claude mcp add` so this stays SSOT-driven (no hardcoded server add).
        $hiveEntry = Backup-AndRestoreClaudeJson -Action { & claude mcp get hive 2>&1 }
        if ($hiveEntry -match 'uvx|hive-vault') {
            Backup-AndRestoreClaudeJson -Action { & claude mcp remove hive --scope user 2>&1 } | Out-Null
            Write-Info "Migrating hive MCP entry: uvx hive-vault -> hive client (via mcp-servers.json)"
        }
        foreach ($srv in $servers) {
            # mcp-servers.json only declares prerequisite_binary/_command for
            # MCPs that need a runtime install (e.g. hive needs uv). Under
            # `Set-StrictMode -Version Latest` direct $srv.prerequisite_binary
            # access throws for servers that omit the field. Probe via
            # PSObject.Properties so missing fields are $null, not an error.
            # Linux side handles this via jq `(.prerequisite_binary // "")`.
            $prereqBin = if ($srv.PSObject.Properties['prerequisite_binary']) { $srv.prerequisite_binary } else { $null }
            $prereqCmdStr = if ($srv.PSObject.Properties['prerequisite_command']) { $srv.prerequisite_command } else { $null }
            if ($prereqBin) {
                $prereqCmd = Get-Command $prereqBin -ErrorAction SilentlyContinue
                if (-not $prereqCmd) {
                    Write-Warn "MCP $($srv.name): prerequisite '$prereqBin' not found, skipping"
                    $mcpFailed++
                    continue
                }
                if ($prereqCmdStr) {
                    $prereqParts = $prereqCmdStr -split '\s+'
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

# dotf CLI (ADR-020 / WIN-006): install the pinned release binary into ~/.local/bin
# (user-space, no admin) — the PowerShell twin of setup-linux.sh sourcing
# install-dotf.sh. goreleaser already publishes the Windows zip, so nothing is
# compiled. Makes the `dotf env path` / `dotf env generate` steps below resolve
# automatically; non-fatal, like the Linux side (`install_dotf || log_warning`).
$installDotfScript = Join-Path $PSScriptRoot 'scripts\install-dotf.ps1'
if (Test-Path $installDotfScript) {
    . $installDotfScript
    if (-not (Install-Dotf)) {
        Write-Warn "dotf installation failed (continuing; the env steps below auto-skip)"
    }
} else {
    Write-Warn "scripts\install-dotf.ps1 not found; skipping dotf install"
}

# Phase C daemon supervision (HIVE-118 / hive#176). Install the supervised
# `hive serve` Scheduled Task now that the MCP loop's prerequisite installed/upgraded
# the tool. Gated on hive-vault >= 1.32.0 via the package version (NOT by probing
# `hive service`, which an older hive routes to the blocking stdio server). The MCP
# entry itself is already `hive client` (re-added from mcp-servers.json above).
# Non-fatal: any failure leaves the in-process `hive client` fallback.
$hiveVerMatch = (& uv tool list 2>$null | Select-String -Pattern '^hive-vault\s+v?([\d.]+)')
$hiveVer = if ($hiveVerMatch) { $hiveVerMatch.Matches[0].Groups[1].Value } else { $null }
if ((Get-Command hive -ErrorAction SilentlyContinue) -and $hiveVer -and ([version]$hiveVer -ge [version]'1.32.0')) {
    & hive service install *> $null
    $hiveSvcRc = $LASTEXITCODE
    if ($hiveSvcRc -eq 0) { Write-Success "Installed hive daemon service (Scheduled Task, v$hiveVer)" }
    else { Write-Warn "hive service install failed (non-fatal; client works via fallback)" }

    # ADR-025 + HARNESS-024 (#446): the hive serve daemon runs as a Scheduled Task
    # and inherits the User-scope environment (not the shell paths.ps1). Persist
    # the resolved vault path at User scope so the daemon follows the cascade.
    # hive#246 hardens the daemon's own resolution.
    $hiveVaultResolved = $null
    if (Get-Command dotf -ErrorAction SilentlyContinue) {
        $hiveVaultResolved = (& dotf env path HIVE_VAULT_PATH 2>$null | Select-Object -First 1)
    }
    if (-not $hiveVaultResolved) {
        $hiveVaultResolved = if ($env:HIVE_VAULT_PATH) { $env:HIVE_VAULT_PATH } elseif ($env:VAULT_PATH) { $env:VAULT_PATH } else { Join-Path $env:USERPROFILE 'Projects\knowledge' }
    }
    [Environment]::SetEnvironmentVariable('HIVE_VAULT_PATH', $hiveVaultResolved, 'User')
    Write-Success "Provisioned hive daemon vault path at User scope ($hiveVaultResolved)"

    # hive#252: on a policy-locked / non-admin box, `hive service install` cannot
    # register the Scheduled Task ("Access is denied" -- even a trivial schtasks
    # /Create fails on a domain machine that blocks non-admin task creation). Fall
    # back to a Startup-folder launcher that runs the daemon supervisor hidden at
    # logon -- no Task Scheduler, no admin. The daemon reads HIVE_VAULT_PATH from
    # the User-scope env provisioned above; the in-process `hive client` fallback
    # still covers correctness if even this is unavailable.
    if ($hiveSvcRc -ne 0) {
        $hiveSupSrc = Join-Path $DotfilesDir "windows\hive-serve-supervisor.ps1"
        $hiveSupDst = Join-Path $ClaudeHome "scripts\hive-serve-supervisor.ps1"
        if (Test-Path $hiveSupSrc) {
            Ensure-Directory (Split-Path $hiveSupDst -Parent)
            Copy-Item $hiveSupSrc $hiveSupDst -Force
            $hiveStartupVbs = Join-Path ([Environment]::GetFolderPath('Startup')) "hive-serve.vbs"
            # .vbs runs the supervisor with a hidden window (0) -> no console flash
            $hiveVbsBody = 'CreateObject("WScript.Shell").Run "powershell.exe -NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File ""' + $hiveSupDst + '""", 0, False'
            Set-Content -Path $hiveStartupVbs -Value $hiveVbsBody -Encoding ASCII
            Write-Success "Provisioned hive daemon autostart via Startup folder (no-admin fallback)"
        } else {
            Write-Warn "hive-serve supervisor script not found at $hiveSupSrc"
        }
    }

    # AI-023 / hive#176 / ADR-015: the upgrade policy that FEEDS the daemon's
    # restart-on-upgrade. Windows cannot replace a running executable, so the task
    # does NOT call `uv tool upgrade` directly -- it runs the orchestration script
    # (only-if-newer -> defer-if-locked -> stop daemon -> upgrade -> start),
    # deployed from the dotfiles SSOT. Cadence matches Linux: every 15 min.
    # Self-healing: re-register when the action OR the S4U principal drift, so a
    # box that still has the old Interactive (windowed) task gets repaired.
    $hiveUvExe = Join-Path $env:USERPROFILE ".local\bin\uv.exe"
    $hiveUpgradeTask = "DotfilesHiveUpgrade"
    $hiveUpgradeSrc = Join-Path $DotfilesDir "windows\hive-upgrade.ps1"
    $hiveUpgradeDst = Join-Path $ClaudeHome "scripts\hive-upgrade.ps1"
    # -WindowStyle Hidden is belt-and-suspenders; the S4U principal applied by
    # Register-HiveScheduledTask (session 0) is what actually guarantees no
    # window when elevated. Non-admin boxes fall back to Interactive, where
    # Hidden is the only mitigation (brief console flash possible).
    $hiveUpgradeArg = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$hiveUpgradeDst`""
    if (Test-Path $hiveUpgradeSrc) {
        Ensure-Directory (Split-Path $hiveUpgradeDst -Parent)
        Copy-Item $hiveUpgradeSrc $hiveUpgradeDst -Force
    } else {
        Write-Warn "hive-upgrade orchestration script not found at $hiveUpgradeSrc"
    }
    $existingHiveTask = Get-ScheduledTask -TaskName $hiveUpgradeTask -ErrorAction SilentlyContinue
    $existingHiveArg = $null
    $existingHiveExe = $null
    $existingHiveLogon = $null
    if ($existingHiveTask -and $existingHiveTask.Actions -and $existingHiveTask.Actions[0]) {
        $existingHiveArg = $existingHiveTask.Actions[0].Arguments
        $existingHiveExe = $existingHiveTask.Actions[0].Execute
        if ($existingHiveTask.Principal) { $existingHiveLogon = "$($existingHiveTask.Principal.LogonType)" }
    }
    if ($existingHiveTask -and ($existingHiveExe -eq "powershell.exe") -and ($existingHiveArg -eq $hiveUpgradeArg) -and ($existingHiveLogon -eq $script:HiveTaskLogonType)) {
        Write-Info "hive-upgrade task already correctly configured, skipping"
    } else {
        try {
            $hiveAction = New-ScheduledTaskAction -Execute "powershell.exe" -Argument $hiveUpgradeArg
            $hiveTrigger = New-ScheduledTaskTrigger -Once -At (Get-Date) `
                -RepetitionInterval (New-TimeSpan -Minutes 15)
            $hiveSettings = New-ScheduledTaskSettingsSet -StartWhenAvailable
            Register-HiveScheduledTask -TaskName $hiveUpgradeTask -Action $hiveAction -Trigger $hiveTrigger `
                -Settings $hiveSettings `
                -Description "hive-vault auto-upgrade every 15 min (ADR-015 orchestrated stop-before-upgrade; feeds hive serve restart-on-upgrade)"
            Write-Success "Installed hive-upgrade task (15-min, $($script:HiveTaskLogonType) principal, v$hiveVer)"
            if ($script:HiveTaskLogonType -ne 'S4U') {
                Write-Info "not elevated: windowless S4U registration unavailable; using Interactive logon with -WindowStyle Hidden"
            }
            if (-not (Test-Path $hiveUvExe)) {
                Write-Warn "hive-upgrade task registered but uv not found at $hiveUvExe"
            }
        } catch {
            Write-Warn "Could not register hive-upgrade task (non-fatal; setup still upgrades hive each run): $_"
        }
    }
} elseif (Get-Command hive -ErrorAction SilentlyContinue) {
    $shownVer = if ($hiveVer) { $hiveVer } else { '<unknown>' }
    Write-Warn "hive $shownVer predates 'hive service' (need >= 1.32.0); skipping daemon supervision"
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
# VaultRoot honors the ADR-025 seam ($env:VAULT_PATH, set by the sourced
# paths.ps1) with the legacy default as fallback — parity with setup-linux.sh.
$VaultRoot = if ($env:VAULT_PATH) { $env:VAULT_PATH } else { Join-Path $env:USERPROFILE "Projects\knowledge" }
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

# Vault-hosted skills are no longer junctioned into ~/.claude/skills (that was
# the BUG-100 fragility source). They are migrated into the vault SSOT, compiled
# into committed records, and rendered to a regular copy by Deploy-SkillRecord
# near the end of this script (SDD-008, option A).

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

# Install Bun (JS runtime -- the claude-mem worker daemon requires it for bun:sqlite)
$bunCmd = Get-Command bun -ErrorAction SilentlyContinue
if (-not $bunCmd) {
    Write-Info "Installing Bun..."
    try {
        $bunInstaller = Join-Path $env:TEMP "bun-install.ps1"
        Invoke-RestMethod https://bun.sh/install.ps1 -OutFile $bunInstaller
        & $bunInstaller 2>$null
        Remove-Item -Path $bunInstaller -Force -ErrorAction SilentlyContinue
        # Refresh PATH for current session
        $env:PATH = "$env:USERPROFILE\.bun\bin;$env:PATH"
        $bunCmd = Get-Command bun -ErrorAction SilentlyContinue
        if ($bunCmd) {
            Write-Success "Bun installed"
        } else {
            Write-Warn "Bun installation failed"
        }
    } catch {
        Write-Warn "Failed to install Bun: $_"
    }
} else {
    Write-Info "Bun already installed"
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
            & npm install -g 'obsidian-cli' 2>$null | Out-Null
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
# 2d. OPENCODE CONFIG + COMMANDS (AI-014)
# ============================================================================
# Binary install is handled in section 1c via winget SST.opencode. This block
# deploys the canonical config + skill-derived commands using the same
# reconcile-not-skip pattern as setup-linux.sh (AI-011, lines 415-465).
# Both files: SHA256 byte-equality test before overwrite so user-side edits
# that match upstream do not trigger a noisy "Deployed" log.

$opencodeConfigSrc = "$DotfilesDir\ai\opencode\opencode.jsonc"
$opencodeConfigDst = Join-Path $env:USERPROFILE '.config\opencode\opencode.jsonc'
if (Test-Path -LiteralPath $opencodeConfigSrc -PathType Leaf) {
    # SDD-009: stage source to a temp file, substitute {env:VAR} placeholders
    # with age-decrypted values (mirror of bash setup-linux.sh logic), then
    # deploy the substituted artifact. Deploy-File's SHA256 check then
    # operates on rendered-vs-deployed (not source-vs-deployed) so idempotence
    # still works after substitution.
    $opencodeConfigTmp = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), "opencode-$PID.jsonc")
    Copy-Item -LiteralPath $opencodeConfigSrc -Destination $opencodeConfigTmp -Force
    if (Get-Command Substitute-EnvPlaceholders -ErrorAction SilentlyContinue) {
        [void](Substitute-EnvPlaceholders -Path $opencodeConfigTmp)
    } else {
        Write-Warn "Substitute-EnvPlaceholders missing; deploying opencode.jsonc with literal {env:VAR} placeholders intact"
    }
    if (Get-Command Deploy-File -ErrorAction SilentlyContinue) {
        [void](Deploy-File -Source $opencodeConfigTmp -Destination $opencodeConfigDst)
    } else {
        Ensure-Directory (Split-Path $opencodeConfigDst -Parent)
        Copy-Item -LiteralPath $opencodeConfigTmp -Destination $opencodeConfigDst -Force
        Write-Success "Deployed opencode.jsonc to $opencodeConfigDst (fallback)"
    }
    Remove-Item -LiteralPath $opencodeConfigTmp -Force -ErrorAction SilentlyContinue
} else {
    Write-Warn "opencode.jsonc source missing: $opencodeConfigSrc"
}

# Deploy the canonical AGENTS.md as opencode's global system prompt.
# OpenCode reads ~/.config/opencode/AGENTS.md (per upstream docs); unlike
# claude/agy/copilot which use pointer files, opencode reads the filename
# "AGENTS.md" natively so we copy the full SSOT (~22KB) verbatim.
# Linux parity: setup-linux.sh:584+
$agentsSrc = Join-Path $DotfilesDir 'AGENTS.md'
$agentsDst = Join-Path $env:USERPROFILE '.config\opencode\AGENTS.md'
if (Test-Path -LiteralPath $agentsSrc -PathType Leaf) {
    if (Get-Command Deploy-File -ErrorAction SilentlyContinue) {
        [void](Deploy-File -Source $agentsSrc -Destination $agentsDst)
    } else {
        Copy-Item -LiteralPath $agentsSrc -Destination $agentsDst -Force
        Write-Success "Deployed AGENTS.md to $agentsDst (fallback)"
    }
} else {
    Write-Warn "AGENTS.md source missing at $agentsSrc"
}

# ============================================================================
# 2d-bis. YARN (npm-pinned global install, TERM-002 companion)
# ============================================================================
# Guarded on npm (same convention as pi). Reconcile-not-skip on version drift.
$yarnVersion = $null
if (Test-Path -LiteralPath $versionsSource) {
    foreach ($line in Get-Content -LiteralPath $versionsSource) {
        if ($line -match '^\s*YARN_VERSION\s*=\s*(.+?)\s*$') {
            $yarnVersion = $Matches[1].Trim().Trim('"').Trim("'")
            break
        }
    }
}
if (Get-Command npm -ErrorAction SilentlyContinue) {
    $yarnPkg = if ($yarnVersion) { "yarn@$yarnVersion" } else { "yarn" }
    if (-not (Get-Command yarn -ErrorAction SilentlyContinue)) {
        Write-Info "Installing yarn ($yarnPkg) via npm..."
        & npm install -g $yarnPkg 2>$null | Out-Null
        if (Get-Command yarn -ErrorAction SilentlyContinue) {
            Write-Success "yarn installed: $(& yarn --version 2>$null)"
        } else {
            Write-Warn "yarn install failed - run: npm install -g $yarnPkg"
        }
    } else {
        $yarnInstalled = (& yarn --version 2>$null | Select-Object -First 1)
        if ($yarnVersion -and -not (Test-VersionAtLeast $yarnInstalled $yarnVersion)) {
            Write-Info "yarn $yarnInstalled below pinned minimum $yarnVersion - upgrading..."
            & npm install -g $yarnPkg 2>$null | Out-Null
            Write-Success "yarn upgraded to $yarnVersion"
        } else {
            Write-Info "yarn already installed: $yarnInstalled"
        }
    }
} else {
    Write-Warn "npm not available, skipping yarn install (install Node.js then re-run)"
}

# ============================================================================
# 2e. PI CODING AGENT CONFIG (AI-025)
# ============================================================================
# Mirror of the opencode block + setup-linux.sh pi deploy so the two agents are
# interchangeable across Linux/Windows. pi reads %USERPROFILE%\.pi\agent\.
# npm-pinned install (guarded); models.json gets the same {env:VAR} deploy-time
# substitution (SDD-009); AGENTS.md is the same SSOT; settings.json seeds only
# when absent (pi mutates it at runtime).
$piVersion = $null
if (Test-Path -LiteralPath $versionsSource) {
    foreach ($line in Get-Content -LiteralPath $versionsSource) {
        if ($line -match '^\s*PI_VERSION\s*=\s*(.+?)\s*$') {
            $piVersion = $Matches[1].Trim().Trim('"').Trim("'")
            break
        }
    }
}
$piAgentDir = Join-Path $env:USERPROFILE '.pi\agent'
if (Get-Command npm -ErrorAction SilentlyContinue) {
    if (-not (Get-Command pi -ErrorAction SilentlyContinue)) {
        $piPkg = if ($piVersion) { "@earendil-works/pi-coding-agent@$piVersion" } else { "@earendil-works/pi-coding-agent" }
        Write-Info "Installing pi ($piPkg) via npm..."
        & npm install -g --ignore-scripts $piPkg 2>$null | Out-Null
        if (Get-Command pi -ErrorAction SilentlyContinue) {
            Write-Success "pi installed"
        } else {
            Write-Warn "pi install failed - run: npm install -g --ignore-scripts $piPkg"
        }
    } else {
        # REFACTOR-013: pin is a MINIMUM — upgrade only when installed < pin so
        # a newer pi is never downgraded by an exact-match reconcile.
        $piVerRaw = (& pi --version 2>&1 | Out-String)
        $piCurrent = if ($piVerRaw -match '(\d+\.\d+\.\d+)') { $Matches[1] } else { '' }
        if ($piVersion -and $piCurrent -and -not (Test-VersionAtLeast $piCurrent $piVersion)) {
            Write-Info "pi $piCurrent below pinned minimum $piVersion; upgrading"
            & npm install -g --ignore-scripts "@earendil-works/pi-coding-agent@$piVersion" 2>$null | Out-Null
            if (Get-Command pi -ErrorAction SilentlyContinue) {
                Write-Success "pi upgraded to pinned $piVersion"
            } else {
                Write-Warn "pi upgrade failed - run: npm install -g --ignore-scripts @earendil-works/pi-coding-agent@$piVersion"
            }
        } else {
            Write-Info "pi already installed"
        }
    }
} else {
    Write-Warn "npm not available, skipping pi install (install Node.js then re-run)"
}
Ensure-Directory $piAgentDir

$piModelsSrc = Join-Path $DotfilesDir 'ai\pi\models.json'
$piModelsDst = Join-Path $piAgentDir 'models.json'
if (Test-Path -LiteralPath $piModelsSrc -PathType Leaf) {
    $piModelsTmp = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), "pi-models-$PID.json")
    Copy-Item -LiteralPath $piModelsSrc -Destination $piModelsTmp -Force
    if (Get-Command Substitute-EnvPlaceholders -ErrorAction SilentlyContinue) {
        [void](Substitute-EnvPlaceholders -Path $piModelsTmp)
    } else {
        Write-Warn "Substitute-EnvPlaceholders missing; deploying pi models.json with literal {env:VAR} placeholders intact"
    }
    if (Get-Command Deploy-File -ErrorAction SilentlyContinue) {
        [void](Deploy-File -Source $piModelsTmp -Destination $piModelsDst)
    } else {
        Ensure-Directory (Split-Path $piModelsDst -Parent)
        Copy-Item -LiteralPath $piModelsTmp -Destination $piModelsDst -Force
        Write-Success "Deployed pi models.json to $piModelsDst (fallback)"
    }
    Remove-Item -LiteralPath $piModelsTmp -Force -ErrorAction SilentlyContinue
} else {
    Write-Warn "pi models.json source missing: $piModelsSrc"
}

$piAgentsDst = Join-Path $piAgentDir 'AGENTS.md'
if (Test-Path -LiteralPath $agentsSrc -PathType Leaf) {
    if (Get-Command Deploy-File -ErrorAction SilentlyContinue) {
        [void](Deploy-File -Source $agentsSrc -Destination $piAgentsDst)
    } else {
        Copy-Item -LiteralPath $agentsSrc -Destination $piAgentsDst -Force
        Write-Success "Deployed AGENTS.md to $piAgentsDst (fallback)"
    }
}

$piSettingsSrc = Join-Path $DotfilesDir 'ai\pi\settings.json'
$piSettingsDst = Join-Path $piAgentDir 'settings.json'
if (Test-Path -LiteralPath $piSettingsSrc -PathType Leaf) {
    if ((Test-Path -LiteralPath $piSettingsDst) -and (-not (Compare-Object (Get-Content $piSettingsSrc) (Get-Content $piSettingsDst)))) {
        Write-Info "pi settings.json already in sync"
    } else {
        Copy-Item -LiteralPath $piSettingsSrc -Destination $piSettingsDst -Force
        Write-Success "Deployed pi settings.json to $piSettingsDst"
    }
}

# Deploy opencode TUI config (theme + keybinds incl. the display_thinking toggle).
# Plain copy: unlike opencode.jsonc this file carries no secrets, so no env-var
# substitution (DX-004). opencode reads .config\opencode\tui.json natively.
# Linux parity: setup-linux.sh tui.json deploy block.
$tuiSrc = Join-Path $DotfilesDir 'ai\opencode\tui.json'
$tuiDst = Join-Path $env:USERPROFILE '.config\opencode\tui.json'
if (Test-Path -LiteralPath $tuiSrc -PathType Leaf) {
    if (Get-Command Deploy-File -ErrorAction SilentlyContinue) {
        [void](Deploy-File -Source $tuiSrc -Destination $tuiDst)
    } else {
        Copy-Item -LiteralPath $tuiSrc -Destination $tuiDst -Force
        Write-Success "Deployed tui.json to $tuiDst (fallback)"
    }
} else {
    Write-Warn "tui.json source missing at $tuiSrc"
}

# OpenCode commands are deployed from the committed vault skill records by
# Deploy-SkillRecord near the end of this script (SDD-008, option A): each
# record whose targets[] includes opencode is rendered to a
# .config\opencode\commands\<n>.md command file (name: dropped), with stale
# commands pruned. This replaces the former ai\opencode\commands copy loop; the
# per-skill targets[] now expresses what was the hard-coded Claude-only skip-list.

# ============================================================================
# 3. DEPLOY GEMINI & ANTIGRAVITY CONFIGURATION (config deploy is always safe)
# ============================================================================

Write-Info "Deploying Antigravity (agy) configuration..."

Ensure-Directory $GeminiHome
Ensure-Directory "$GeminiHome\config"
Ensure-Directory "$GeminiHome\prompts"
$AgyAppData = Join-Path $GeminiHome "antigravity-cli"
Ensure-Directory $AgyAppData

# Force production endpoint
[Environment]::SetEnvironmentVariable("ANTIGRAVITY_ENDPOINT", "https://cloudcode-pa.googleapis.com", "User")
[Environment]::SetEnvironmentVariable("CLOUDCODE_URL", "https://cloudcode-pa.googleapis.com", "User")
[Environment]::SetEnvironmentVariable("GEMINI_DIR", "$GeminiHome", "User")
$env:ANTIGRAVITY_ENDPOINT = "https://cloudcode-pa.googleapis.com"
$env:CLOUDCODE_URL = "https://cloudcode-pa.googleapis.com"
$env:GEMINI_DIR = "$GeminiHome"

# 1. Deploy agy settings.json (SDD-007: no legacy Gemini-CLI compat write)
$agySettingsSrc = "$DotfilesDir\ai\agy\settings.json"
if (Test-Path $agySettingsSrc) {
    Copy-Item $agySettingsSrc "$AgyAppData\settings.json" -Force
    Write-Success "Deployed agy settings.json"
}

# Deploy .geminiignore
$geminiIgnoreSrc = "$DotfilesDir\.geminiignore"
if (Test-Path $geminiIgnoreSrc) {
    Copy-Item $geminiIgnoreSrc "$GeminiHome\.geminiignore" -Force
    Write-Success "Deployed .geminiignore"
}

# 2. Consolidate MCP servers - master at ~/.gemini/config/mcp_config.json (agy's canonical read path)
$mcpServersSrc = "$DotfilesDir\ai\agy\mcp_servers.json"
$rootMcpSrc = "$DotfilesDir\mcp-servers.json"
if ((Test-Path $mcpServersSrc) -and (Test-Path $rootMcpSrc) -and (Get-Command jq -ErrorAction SilentlyContinue)) {
    Write-Info "Consolidating Antigravity MCP servers..."

    # Recover OpenRouter key from existing master config.
    # NOTE: canonical agy schema uses `mcpServers` (not `servers`).
    $oldKey = $env:OPENROUTER_API_KEY
    if ([string]::IsNullOrEmpty($oldKey)) {
        $existingMaster = Join-Path $GeminiHome "config\mcp_config.json"
        if (Test-Path $existingMaster) {
            $oldKey = & jq -r '.mcpServers["hive-vault"].env.OPENROUTER_API_KEY // empty' $existingMaster 2>$null | Where-Object { $_ -ne "null" }
        }
    }

    $mcpConfigJson = Get-Content $mcpServersSrc -Raw | ConvertFrom-Json
    if (-not [string]::IsNullOrEmpty($oldKey) -and $oldKey -ne '${OPENROUTER_API_KEY}') {
        $mcpConfigJson.mcpServers."hive-vault".env.OPENROUTER_API_KEY = $oldKey
    }

    # Substitute ${VAULT_PATH} placeholder with the canonical Windows vault dir.
    # The committed JSON uses ${VAULT_PATH} so it's OS-portable; agy does NOT
    # expand env vars inside JSON values, so substitution must happen here
    # before write. The path itself is by convention: $USERPROFILE\Projects\knowledge.
    # ADR-025: honor $env:VAULT_PATH (the cross-machine seam) before the legacy
    # default — parity with setup-linux.sh:402 ${VAULT_PATH:-...}.
    $vaultPath = if ($env:VAULT_PATH) { $env:VAULT_PATH } else { Join-Path $env:USERPROFILE 'Projects\knowledge' }
    $hiveEntry = $mcpConfigJson.mcpServers."hive-vault"
    if ($hiveEntry -and $hiveEntry.env.PSObject.Properties['VAULT_PATH']) {
        $hiveEntry.env.VAULT_PATH = $vaultPath
    }

    # Preflight: WARN (non-fatal) if the vault dir doesn't exist on disk.
    # hive-vault MCP server will fail at first tool call without it, but
    # encrypted secrets + AGY.md + everything else still deploys cleanly so
    # creating the vault later doesn't require a setup re-run.
    if (-not (Test-Path -LiteralPath $vaultPath -PathType Container)) {
        Write-Warn "Obsidian vault not found at $vaultPath"
        Write-Warn "  hive-vault MCP will error at runtime until this dir exists."
        Write-Warn "  Either clone/sync your vault to $vaultPath, or override via"
        Write-Warn "  the VAULT_PATH env var in your profile."
    }

    # Merge stdio servers from root mcp-servers.json into mcpServers map
    $rootServers = (Get-Content $rootMcpSrc -Raw | ConvertFrom-Json).servers
    foreach ($srv in $rootServers) {
        if ($srv.name -eq "hive") { continue }
        if ($srv.transport -ne "stdio") { continue }

        $parts = $srv.args -split '\s+'
        $cmd = $parts[0]
        # NOTE: $args is an automatic PowerShell variable; use serverArgs to avoid PSAvoidAssignmentToAutomaticVariable
        $serverArgs = $parts[1..($parts.Length-1)]
        $mcpConfigJson.mcpServers | Add-Member -MemberType NoteProperty -Name $srv.name -Value @{ command = $cmd; args = $serverArgs } -Force
    }

    # Write master to canonical path. No symlinks, no copies elsewhere.
    $masterConfig = Join-Path $GeminiHome "config\mcp_config.json"
    $mcpConfigJson | ConvertTo-Json -Depth 10 | Set-Content $masterConfig -Encoding UTF8
    Write-Success "Deployed master MCP config to $masterConfig"

    # Hive plugin discovery file
    $hivePluginDir = Join-Path $AgyAppData "plugins\hive-vault"
    Ensure-Directory $hivePluginDir
    '{"name": "hive-vault"}' | Set-Content (Join-Path $hivePluginDir "plugin.json") -Encoding UTF8
    $hiveMcp = @{ mcpServers = @{ "hive-vault" = $mcpConfigJson.mcpServers."hive-vault" } }
    $hiveMcp | ConvertTo-Json -Depth 10 | Set-Content (Join-Path $hivePluginDir "mcp_config.json") -Encoding UTF8

    Write-Success "Registered Hive plugin"
}

# Drop project-local agy state cache from the workspace if agy leaked one
$projAgyCli = Join-Path $DotfilesDir ".antigravitycli"
$projAgyBak = Join-Path $DotfilesDir ".antigravitycli.bak"
if (Test-Path $projAgyCli) { Remove-Item -Recurse -Force $projAgyCli -ErrorAction SilentlyContinue }
if (Test-Path $projAgyBak) { Remove-Item -Recurse -Force $projAgyBak -ErrorAction SilentlyContinue }

# SDD-007 one-time migration: gemini-cli -> agy. Remove the legacy
# ~/.gemini/GEMINI.md identity file so it doesn't linger as an orphan
# pointing to the retired binary. Safe to repeat (no-op if absent).
$geminiMdLegacy = Join-Path $GeminiHome 'GEMINI.md'
if (Test-Path $geminiMdLegacy) {
    Remove-Item -LiteralPath $geminiMdLegacy -Force -ErrorAction SilentlyContinue
    Write-Info "Removed legacy GEMINI.md (SDD-007 migration: agy replaces gemini-cli)"
}

# Deploy AGY.md (Neural Hive Protocol pointer to AGENTS.md). Linux-parity.
$agyMdSource = Join-Path $DotfilesDir 'ai\agy\AGY.md'
if (Test-Path $agyMdSource) {
    Copy-Item -LiteralPath $agyMdSource -Destination (Join-Path $GeminiHome 'AGY.md') -Force
    if (Select-String -Path (Join-Path $GeminiHome 'AGY.md') -Pattern 'First, read `AGENTS.md`' -SimpleMatch -Quiet) {
        Write-Success "AGY.md deployed successfully (verified pointer to AGENTS.md)"
    } else {
        Write-Err "AGY.md deployment failed verification (expected pointer to AGENTS.md)"
    }
}

# Agy skills (native Shared skills under $GeminiHome\skills and frontmatter-
# stripped flat prompts under $GeminiHome\prompts) are deployed from the
# committed vault skill records by Deploy-SkillRecord near the end of this
# script (SDD-008, option A), honoring each skill's targets[]. The former
# ai\skills extraction loops are gone.
Ensure-Directory (Join-Path $GeminiHome 'skills')
Ensure-Directory (Join-Path $GeminiHome 'prompts')

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

    # Marker for dotfiles section
    $startMarker = "# >>> DOTFILES PROFILE >>>"
    $endMarker = "# <<< DOTFILES PROFILE <<<"

    # BUG-021 preflight: refuse to touch a corrupted profile (size or marker
    # count outside healthy bounds). The previous version of this block
    # swallowed -replace OOM and printed [SUCCESS] regardless, which let
    # BUG-020 (26 MB / 689K-line profile) accumulate silently across runs.
    if (Test-Path $profileTarget) {
        $profileInfo = Get-Item -LiteralPath $profileTarget -ErrorAction Stop
        if ($profileInfo.Length -gt 1MB) {
            Write-Err ("PowerShell profile is suspiciously large ({0:N0} bytes); refusing to modify in place." -f $profileInfo.Length)
            Write-Err "Likely BUG-020 corruption. Run: scripts\profile-heal.ps1 to reconstruct from SSOT, then re-run setup-windows.ps1."
            exit 1
        }

        $rawForPreflight = Get-Content -LiteralPath $profileTarget -Raw -ErrorAction Stop
        $startMatches = [regex]::Matches($rawForPreflight, [regex]::Escape($startMarker))
        $endMatches = [regex]::Matches($rawForPreflight, [regex]::Escape($endMarker))
        $startMarkerCount = $startMatches.Count
        $endMarkerCount = $endMatches.Count
        if ($startMarkerCount -gt 2 -or $endMarkerCount -gt 2) {
            Write-Err ("PowerShell profile has unbalanced/duplicate dotfiles markers (start={0}, end={1}); refusing to modify." -f $startMarkerCount, $endMarkerCount)
            Write-Err "Likely BUG-020 corruption from a previous silent partial write. Run: scripts\profile-heal.ps1, then re-run setup-windows.ps1."
            exit 1
        }
    }

    try {
        # Read source profile (fail-fast if missing -- -ErrorAction Stop
        # promotes Get-Content's non-terminating error so the catch fires).
        $sourceContent = Get-Content -LiteralPath $profileSource -Raw -ErrorAction Stop

        if (Test-Path $profileTarget) {
            $existingContent = Get-Content -LiteralPath $profileTarget -Raw -ErrorAction Stop

            # Check if our section already exists
            if ($existingContent -match [regex]::Escape($startMarker)) {
                # Replace existing section using index-based split (BUG-022:
                # PowerShell -replace with [\s\S]*? expands large strings
                # instead of replacing — see debug-replace*.ps1 traces).
                $newSection = "$startMarker`r`n$sourceContent`r`n$endMarker"
                $markerIdx = $existingContent.IndexOf($startMarker)
                $endIdx = $existingContent.IndexOf($endMarker, $markerIdx)
                $before = $existingContent.Substring(0, $markerIdx)
                $after = $existingContent.Substring($endIdx + $endMarker.Length)
                $newContent = $before + $newSection + $after
                Set-Content -LiteralPath $profileTarget -Value $newContent -Encoding UTF8 -NoNewline -ErrorAction Stop
                Write-Success "Updated dotfiles section in PowerShell profile"
            } else {
                # Append our section
                $appendContent = "`r`n`r`n$startMarker`r`n$sourceContent`r`n$endMarker"
                Add-Content -LiteralPath $profileTarget -Value $appendContent -Encoding UTF8 -ErrorAction Stop
                Write-Success "Appended dotfiles section to PowerShell profile"
            }
        } else {
            # Create new profile with our section
            $newContent = "$startMarker`r`n$sourceContent`r`n$endMarker"
            Set-Content -LiteralPath $profileTarget -Value $newContent -Encoding UTF8 -NoNewline -ErrorAction Stop
            Write-Success "Created PowerShell profile at $profileTarget"
        }
    } catch {
        Write-Err "Failed to update PowerShell profile: $($_.Exception.Message)"
        Write-Err "If the profile appears corrupted, run scripts\profile-heal.ps1 then re-run setup-windows.ps1."
        exit 1
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
        $srcHash = (Get-FileHash -LiteralPath $gitconfigSource -Algorithm SHA256).Hash
        $dstHash = (Get-FileHash -LiteralPath $gitconfigTarget -Algorithm SHA256).Hash
        if ($srcHash -eq $dstHash) {
            Write-Info ".gitconfig already exists and matches repo (no action needed)"
        } else {
            Write-Warn ".gitconfig already exists at $gitconfigTarget (differs from repo version)"
            Write-Info "Skipping to avoid overwriting existing configuration"
            Write-Info "To merge manually: Copy-Item '$gitconfigSource' '$gitconfigTarget' -Force"
        }
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

# CLI-020: the init .ps1 were retired (repo scaffolding is now `dotf init`).
# Remove any copy a prior setup left in ~/scripts or ~/.claude so they don't
# linger as orphans (mirrors setup-linux.sh's init-project.sh cleanup, CLI-014).
foreach ($initOrphan in @(
        "$ScriptsDir\init-project.ps1", "$ClaudeHome\init-project.ps1",
        "$ScriptsDir\init-repo-agents.ps1", "$ClaudeHome\init-repo-agents.ps1",
        "$ScriptsDir\init-repo-github-defaults.ps1", "$ClaudeHome\init-repo-github-defaults.ps1")) {
    if (Test-Path $initOrphan) { Remove-Item $initOrphan -Force -ErrorAction SilentlyContinue }
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

$sessionHandoffSource = "$DotfilesDir\scripts\session-handoff.ps1"
if (Test-Path $sessionHandoffSource) {
    Copy-Item $sessionHandoffSource "$ScriptsDir\" -Force
    Write-Success "Deployed session-handoff.ps1 to $ScriptsDir\"
} else {
    Write-Warn "session-handoff.ps1 not found at $sessionHandoffSource"
}

$memHealSource = "$DotfilesDir\scripts\claude-mem-heal.ps1"
if (Test-Path $memHealSource) {
    Copy-Item $memHealSource "$ScriptsDir\" -Force
    Write-Success "Deployed claude-mem-heal.ps1 to $ScriptsDir\"
} else {
    Write-Warn "claude-mem-heal.ps1 not found at $memHealSource"
}

# BUG-020: profile-heal.ps1 reconstructs a corrupted PowerShell profile from
# the SSOT. Invoked by doctor.ps1 -Fix and pointed at by setup-windows.ps1's
# preflight error message (BUG-021).
$profileHealSource = "$DotfilesDir\scripts\profile-heal.ps1"
if (Test-Path $profileHealSource) {
    Copy-Item $profileHealSource "$ScriptsDir\" -Force
    Write-Success "Deployed profile-heal.ps1 to $ScriptsDir\"
} else {
    Write-Warn "profile-heal.ps1 not found at $profileHealSource"
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

# DX-006: deploy + re-apply the Orca/Copilot PreToolUse hook fix. Orca regenerates
# ~/.copilot/hooks/orca.json and ~/.orca/agent-hooks/copilot-hook.ps1 on every
# install/upgrade, reverting the fix (timeoutSec 5 + slow Invoke-WebRequest) and
# making every Copilot tool call fail with "hook errored". The script is idempotent
# and skips cleanly when Orca is not installed, so it is always safe to run.
$orcaHookTuneSource = "$DotfilesDir\scripts\orca-hook-tune.ps1"
if (Test-Path $orcaHookTuneSource) {
    Copy-Item $orcaHookTuneSource "$ScriptsDir\" -Force
    Write-Success "Deployed orca-hook-tune.ps1 to $ScriptsDir\"
    try {
        & "$ScriptsDir\orca-hook-tune.ps1"
    } catch {
        Write-Warn "orca-hook-tune.ps1 failed: $($_.Exception.Message)"
    }
} else {
    Write-Warn "orca-hook-tune.ps1 not found at $orcaHookTuneSource"
}

# WIN-005: deploy the HKCU engineering-defaults script (invoked opt-in below).
$winDefaultsSource = "$DotfilesDir\scripts\windows-defaults.ps1"
if (Test-Path $winDefaultsSource) {
    Copy-Item $winDefaultsSource "$ScriptsDir\" -Force
    Write-Success "Deployed windows-defaults.ps1 to $ScriptsDir\"
} else {
    Write-Warn "windows-defaults.ps1 not found at $winDefaultsSource"
}

$contractSource = "$DotfilesDir\env-contract.json"
if (Test-Path $contractSource) {
    Copy-Item $contractSource "$DotfilesDest\" -Force
    Write-Success "Deployed env-contract.json to $DotfilesDest\"
} else {
    Write-Warn "env-contract.json not found at $contractSource"
}

# Render the per-machine path file (ADR-025) from the contract + the
# %USERPROFILE%\.config\dotfiles\machine.json overrides, so profile.ps1 sources
# it. Guarded on dotf being installed (the Windows dotf install path is still on
# the WIN queue); until then profile.ps1 uses its bootstrap fallback.
if (Get-Command dotf -ErrorAction SilentlyContinue) {
    dotf env generate | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Generated $DotfilesDest\paths.ps1 (dotf env generate)"
        $pathsFile = Join-Path $DotfilesDest 'paths.ps1'
        if (Test-Path $pathsFile) { . $pathsFile }
    } else {
        Write-Warn "dotf env generate failed (profile.ps1 falls back to inline defaults)"
    }
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

# Preflight: warn if age identity key is missing. Without it, load-secrets.ps1
# silently no-ops at shell startup (Invoke-AgeDecrypt returns $null on failure)
# so $env:NAN_API_KEY / OPENROUTER_API_KEY / etc. stay empty -- opencode + agy
# then 401 with no clear cause. Non-fatal: encrypted files still get deployed
# so a key imported later still works without re-running setup. Mirrors the
# Linux preflight in setup-linux.sh.
$ageKey = if ($env:AGE_KEY_PATH) { $env:AGE_KEY_PATH } else { Join-Path $env:USERPROFILE '.config\age\key.txt' }
if (-not (Test-Path -LiteralPath $ageKey)) {
    Write-Warn "age identity key not found at $ageKey"
    Write-Warn "  Encrypted secrets will deploy but won't decrypt at shell startup."
    Write-Warn "  To enable: place your age identity at `$env:USERPROFILE\.config\age\key.txt"
    Write-Warn "  (or set `$env:AGE_KEY_PATH). Generate: age-keygen -o `$HOME\.config\age\key.txt"
    Write-Warn "  See: docs/SECRETS.md"
}

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

# Eager-load secrets: dot-source load-secrets.ps1 NOW (after the .secret.age
# files + env-mapping.conf are at $DotfilesDest\sensitive) so every subsequent
# block in this setup has $env:NAN_API_KEY / OPENROUTER_API_KEY / VAULT_PATH /
# TS_AUTHKEY / etc. available. Cross-OS parity with setup-linux.sh:115-122.
# Wrapped to swallow internal failures (no age key, no env-mapping, etc.) so
# setup never aborts on secrets issues -- preflight already warned the user.
$loadSecretsDeployed = Join-Path $DotfilesDest 'scripts\load-secrets.ps1'
if (Test-Path -LiteralPath $loadSecretsDeployed) {
    $env:DOTFILES_DIR = $DotfilesDest
    try { . $loadSecretsDeployed 2>&1 | Out-Null } catch { }
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
$sessionEndCmd = "$ScriptsDir\session-handoff.ps1"
$expectedSessionEndCommand = "pwsh -NoProfile -File `"$sessionEndCmd`""

Merge-ClaudeSettings -TemplatePath $ClaudeSettingsTemplate -TargetPath $ClaudeSettings -HookCommand $expectedHookCommand -SessionEndCommand $expectedSessionEndCommand

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

# Sync .github/copilot-instructions.md from ai/copilot/ (SDD-005 parity rule).
# The .github/ copy is what GitHub's web Copilot Chat reads; ai/copilot/ is the
# SSOT. Setup keeps them in sync so the docs-drift test never fails.
$ghCopilotDst = Join-Path $DotfilesDir '.github\copilot-instructions.md'
$aiCopilotSrc = Join-Path $DotfilesDir 'ai\copilot\copilot-instructions.md'
if (Test-Path -LiteralPath $aiCopilotSrc -PathType Leaf) {
    if ((Test-Path -LiteralPath $ghCopilotDst) -and (-not (Compare-Object (Get-Content $aiCopilotSrc) (Get-Content $ghCopilotDst)))) {
        Write-Info ".github/copilot-instructions.md already in sync"
    } else {
        Copy-Item -LiteralPath $aiCopilotSrc -Destination $ghCopilotDst -Force
        Write-Success "Synced .github/copilot-instructions.md from ai/copilot/"
    }
}

# ============================================================================
# SDD-008: deploy skills from committed records (option A, render-at-deploy)
# Windows port of `compile-harness.sh --deploy`. ENGINE-001's --refresh/--check
# stay Linux-only; Windows consumes the committed records under harness/skills/,
# the same way it consumes the committed harness override blocks. Each record is
# rendered to its per-agent path under the user profile as a regular copy (never
# a junction -- ends the BUG-100 class), honoring each skill's targets[]. Runs
# after the per-agent base files are deployed so copilot catalog injection sees
# the instructions file already in place.
# ============================================================================

function Test-SkillTargetsAgent {
    param([string]$SkillMd, [string]$Agent)
    if (-not (Test-Path -LiteralPath $SkillMd)) { return $false }
    $fm = 0
    foreach ($line in Get-Content -LiteralPath $SkillMd) {
        if ($line -match '^---\s*$') { $fm++; if ($fm -ge 2) { break }; continue }
        if ($fm -eq 1 -and $line -match '^targets:\s*(.+)$') {
            return [bool]($Matches[1] -match ('\b' + [regex]::Escape($Agent) + '\b'))
        }
    }
    return $true   # no targets: -> all agents
}

function Get-SkillField {
    param([string]$SkillMd, [string]$Key)
    $fm = 0
    foreach ($line in Get-Content -LiteralPath $SkillMd) {
        if ($line -match '^---\s*$') { $fm++; if ($fm -ge 2) { break }; continue }
        if ($fm -eq 1 -and $line -match ('^' + [regex]::Escape($Key) + ':\s*(.+)$')) { return $Matches[1].Trim() }
    }
    return ''
}

function Convert-SkillRecord {
    param([string]$Kind, [string]$RecordMd, [string]$SrcPath)
    $sha = (Get-FileHash -LiteralPath $RecordMd -Algorithm SHA256).Hash.Substring(0, 16).ToLower()
    if ($Kind -eq 'prompt') {
        $raw = Get-Content -LiteralPath $RecordMd -Raw
        if ($raw -match '^---\r?\n[\s\S]*?\r?\n---\r?\n([\s\S]*)$') { $body = $Matches[1] } else { $body = $raw }
        return "<!-- generated: true; from: $SrcPath; sha256:$sha; edit the vault source + re-run setup -->`n`n" + $body.Trim()
    }
    $fm = 0
    $out = New-Object System.Collections.Generic.List[string]
    foreach ($line in Get-Content -LiteralPath $RecordMd) {
        if ($line -match '^---\s*$') {
            $fm++
            $out.Add($line)
            if ($fm -eq 1) {
                $out.Add('generated: true')
                $out.Add("generated_from: $SrcPath")
                $out.Add("generated_sha: $sha")
            }
            continue
        }
        if ($fm -eq 1 -and $Kind -eq 'command' -and $line -match '^name:') { continue }
        $out.Add($line)
    }
    return ($out -join "`n")
}

function Deploy-SkillRecord {
    param([string]$DotfilesDir)
    $manifestPath = Join-Path $DotfilesDir 'harness\manifest.json'
    if (-not (Test-Path -LiteralPath $manifestPath)) { Write-Warn "harness manifest not found: $manifestPath"; return }
    $manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
    if (-not $manifest.skills) { Write-Info "no skills block in manifest; skipping skill deploy"; return }
    $recordDir = Join-Path $DotfilesDir ($manifest.skills.record_dir -replace '/', '\')
    $vsub = $manifest.skills.vault_subpath
    if (-not (Test-Path -LiteralPath $recordDir)) { Write-Warn "no skill records at $recordDir; skipping skill deploy"; return }
    $records = @(Get-ChildItem -LiteralPath $recordDir -Directory -ErrorAction SilentlyContinue)

    foreach ($d in $manifest.skills.deploy) {
        $destBase = Join-Path $env:USERPROFILE ($d.dir -replace '/', '\')
        Ensure-Directory $destBase
        foreach ($rec in $records) {
            $recMd = Join-Path $rec.FullName 'SKILL.md'
            if (-not (Test-Path -LiteralPath $recMd)) { continue }
            if (-not (Test-SkillTargetsAgent -SkillMd $recMd -Agent $d.agent)) { continue }
            $src = "$vsub/$($rec.Name)/SKILL.md"
            if ($d.render -eq 'command' -or $d.render -eq 'prompt') {
                $target = Join-Path $destBase ("{0}.md" -f $rec.Name)
                if ((Test-Path -LiteralPath $target) -and ((Get-Item -LiteralPath $target -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) {
                    Remove-Item -LiteralPath $target -Force -ErrorAction SilentlyContinue
                }
                Set-Content -LiteralPath $target -Value (Convert-SkillRecord -Kind $d.render -RecordMd $recMd -SrcPath $src) -Encoding UTF8
            } else {
                $destDir = Join-Path $destBase $rec.Name
                if (Test-Path -LiteralPath $destDir) {
                    $item = Get-Item -LiteralPath $destDir -Force
                    if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { cmd /c rmdir "$destDir" 2>&1 | Out-Null }
                }
                Ensure-Directory $destDir
                Copy-Item -Path (Join-Path $rec.FullName '*') -Destination $destDir -Recurse -Force -ErrorAction SilentlyContinue
                Set-Content -LiteralPath (Join-Path $destDir 'SKILL.md') -Value (Convert-SkillRecord -Kind $d.render -RecordMd $recMd -SrcPath $src) -Encoding UTF8
            }
        }
        # Prune our own stale outputs (skill removed, or targets[] dropped this agent).
        if ($d.render -eq 'command' -or $d.render -eq 'prompt') {
            foreach ($f in Get-ChildItem -LiteralPath $destBase -Filter '*.md' -File -ErrorAction SilentlyContinue) {
                if (-not (Select-String -LiteralPath $f.FullName -Pattern 'generated: true' -SimpleMatch -Quiet)) { continue }
                $recMd = Join-Path (Join-Path $recordDir $f.BaseName) 'SKILL.md'
                if (-not (Test-Path -LiteralPath $recMd) -or -not (Test-SkillTargetsAgent -SkillMd $recMd -Agent $d.agent)) {
                    Remove-Item -LiteralPath $f.FullName -Force -ErrorAction SilentlyContinue
                }
            }
        } else {
            foreach ($sd in Get-ChildItem -LiteralPath $destBase -Directory -ErrorAction SilentlyContinue) {
                $deployedMd = Join-Path $sd.FullName 'SKILL.md'
                if (-not (Test-Path -LiteralPath $deployedMd)) { continue }
                if (-not (Select-String -LiteralPath $deployedMd -Pattern 'generated: true' -SimpleMatch -Quiet)) { continue }
                $recMd = Join-Path (Join-Path $recordDir $sd.Name) 'SKILL.md'
                if (-not (Test-Path -LiteralPath $recMd) -or -not (Test-SkillTargetsAgent -SkillMd $recMd -Agent $d.agent)) {
                    Remove-Item -LiteralPath $sd.FullName -Recurse -Force -ErrorAction SilentlyContinue
                }
            }
        }
    }

    # Copilot catalog: inject a bullet list into the deployed instructions file.
    if ($manifest.skills.catalog) {
        $catFile = Join-Path $env:USERPROFILE ($manifest.skills.catalog.file -replace '/', '\')
        $catAgent = $manifest.skills.catalog.agent
        $beginPrefix = '<!-- BEGIN HARNESS GENERATED'
        $endMarker = '<!-- END HARNESS GENERATED -->'
        if ((Test-Path -LiteralPath $catFile) -and (Select-String -LiteralPath $catFile -Pattern $beginPrefix -SimpleMatch -Quiet)) {
            $bullets = New-Object System.Collections.Generic.List[string]
            foreach ($rec in ($records | Sort-Object Name)) {
                $recMd = Join-Path $rec.FullName 'SKILL.md'
                if (-not (Test-Path -LiteralPath $recMd)) { continue }
                if (-not (Test-SkillTargetsAgent -SkillMd $recMd -Agent $catAgent)) { continue }
                $desc = Get-SkillField -SkillMd $recMd -Key 'description'
                $bullets.Add(("- **{0}** -- {1}" -f $rec.Name, $desc))
            }
            $catBody = ($bullets -join "`n")
            $bytes = [System.Text.Encoding]::UTF8.GetBytes($catBody + "`n")
            $sha = (([System.Security.Cryptography.SHA256]::Create().ComputeHash($bytes) | ForEach-Object { $_.ToString('x2') }) -join '').Substring(0, 16)
            $beginLine = "$beginPrefix (sha256:$sha) -- skill catalog from vault $vsub; edit there + re-run setup, do NOT edit between markers -->"
            $result = New-Object System.Collections.Generic.List[string]
            $state = 'before'
            foreach ($line in Get-Content -LiteralPath $catFile) {
                if ($state -eq 'before' -and $line.StartsWith($beginPrefix)) {
                    $result.Add($beginLine)
                    foreach ($b in $bullets) { $result.Add($b) }
                    $state = 'inside'
                    continue
                }
                if ($state -eq 'inside') {
                    if ($line -eq $endMarker) { $result.Add($line); $state = 'after' }
                    continue
                }
                $result.Add($line)
            }
            Set-Content -LiteralPath $catFile -Value $result -Encoding UTF8
            Write-Success "Injected skill catalog into $catFile"
        }
    }

    Write-Success "Skills deployed from records (claude / opencode / agy / copilot)"
}

Deploy-SkillRecord -DotfilesDir $DotfilesDir

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

if (-not $env:SCRIPTS_DIR)       { $env:SCRIPTS_DIR       = "$env:DOTFILES_DIR\scripts" }
if (-not $env:GEMINI_HOME)       { $env:GEMINI_HOME       = "$env:USERPROFILE\.gemini" }
# AGY_HOME (SSOT name per env-contract.json) lives inside GEMINI_HOME, since
# agy inherits the legacy ~/.gemini/ path for backwards compat with gemini-cli.
if (-not $env:AGY_HOME)          { $env:AGY_HOME          = "$env:GEMINI_HOME\antigravity-cli" }
if (-not $env:COPILOT_HOME)      { $env:COPILOT_HOME      = "$env:USERPROFILE\.copilot" }
if (-not $env:OPENCODE_HOME)     { $env:OPENCODE_HOME     = "$env:USERPROFILE\.config\opencode" }
# BUG-021 (2026-05-21): added DOTFILES_REPO_DIR pre-export -- BUG-020 (PR #86)
# added the export to profile.ps1 but the running shell doesn't reload profile,
# so `dotf doctor` (the repo/deploy drift check, CLI-019) still sees the var
# unset until next shell restart. Mirror the same pre-export pattern as the other
# 4 vars to surface PASS immediately in the post-setup checks.
if (-not $env:DOTFILES_REPO_DIR) { $env:DOTFILES_REPO_DIR = "$env:USERPROFILE\Projects\dotfiles" }

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

# OPS-001: opt-in self-deploy Scheduled Task (Windows parity of the systemd timer).
# Tri-state, gated on DOTFILES_AUTODEPLOY so a normal setup never touches it:
#   1     -> deploy the selfupdate script + register a daily scheduled task
#   0     -> unregister the task (clean opt-out)
#   unset -> no-op (opt-in, default OFF)
# Routes through Register-HiveScheduledTask (the strongest-principal registrar:
# windowless S4U when elevated, Interactive fallback when not), same as the
# hive tasks.
$selfUpdateTask = "DotfilesSelfUpdate"
switch ("$env:DOTFILES_AUTODEPLOY") {
    "1" {
        $selfUpdateSrc = Join-Path $DotfilesDir "scripts\dotfiles-selfupdate.ps1"
        $selfUpdateDst = Join-Path $ClaudeHome "scripts\dotfiles-selfupdate.ps1"
        if (Test-Path $selfUpdateSrc) {
            Ensure-Directory (Split-Path $selfUpdateDst -Parent)
            Copy-Item $selfUpdateSrc $selfUpdateDst -Force
            $selfUpdateArg = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$selfUpdateDst`""
            try {
                $suAction = New-ScheduledTaskAction -Execute "powershell.exe" -Argument $selfUpdateArg
                $suTrigger = New-ScheduledTaskTrigger -Daily -At 3am
                $suSettings = New-ScheduledTaskSettingsSet -StartWhenAvailable
                Register-HiveScheduledTask -TaskName $selfUpdateTask -Action $suAction -Trigger $suTrigger `
                    -Settings $suSettings `
                    -Description "dotfiles self-deploy: daily git pull --ff-only + idempotent setup (OPS-001)"
                Write-Success "Enabled DotfilesSelfUpdate task (daily, $($script:HiveTaskLogonType) principal)"
            } catch {
                Write-Warn "Could not register DotfilesSelfUpdate task (non-fatal): $_"
            }
        } else {
            Write-Warn "self-update script not found at $selfUpdateSrc"
        }
    }
    "0" {
        Unregister-ScheduledTask -TaskName $selfUpdateTask -Confirm:$false -ErrorAction SilentlyContinue
        Write-Info "Disabled DotfilesSelfUpdate task (DOTFILES_AUTODEPLOY=0)"
    }
    default { }
}

# ============================================================================
# WIN-005: OPT-IN WINDOWS ENGINEERING DEFAULTS (HKCU)
# ============================================================================
# OFF by default (proposal R5): a setup run must never mass-edit user
# preferences without explicit consent. Runs the deployed copy so what
# executes is exactly what landed in $ScriptsDir.
if ($WithDefaults) {
    $winDefaultsDeployed = "$ScriptsDir\windows-defaults.ps1"
    if (Test-Path $winDefaultsDeployed) {
        Write-Info "Applying Windows engineering defaults (-WithDefaults)..."
        & $winDefaultsDeployed
    } else {
        Write-Warn "windows-defaults.ps1 not deployed; skipping -WithDefaults"
    }
} else {
    Write-Info "Windows defaults not applied (opt-in: re-run with -WithDefaults)"
}

Write-Host ""
Write-Host "============================================" -ForegroundColor Green
Write-Host "  Setup Complete!                          " -ForegroundColor Green
Write-Host "============================================" -ForegroundColor Green
Write-Host ""
Write-Host "Deployed:" -ForegroundColor Cyan
Write-Host "  - Claude config:  $ClaudeHome\CLAUDE.md"
Write-Host "  - Claude skills:  $ClaudeHome\skills\"
Write-Host "  - Antigravity:    $GeminiHome\AGY.md  ($AgyAppData\)"
Write-Host "  - Agy prompts:    $GeminiHome\prompts\"
Write-Host "  - Scripts:        $ScriptsDir\"
Write-Host "  - Secrets:        $DotfilesDest\sensitive\"
Write-Host "  - Copilot config: $env:USERPROFILE\.copilot\"
Write-Host "  - Profile:        $profileTarget"
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Restart PowerShell to load the new profile"
Write-Host "  2. Verify setup:"
Write-Host "       Test-Path `"$ClaudeHome\CLAUDE.md`""
Write-Host "       Test-Path `"$GeminiHome\AGY.md`""
Write-Host "       `$env:PATH -like `"*scripts*`""
Write-Host "  3. Initialize a project:"
Write-Host "       project-init test-project python"
Write-Host ""
