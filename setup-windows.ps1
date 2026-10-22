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
# This script is verified under pwsh 7 only: CI starts it from Windows
# PowerShell 5.1 and this block re-execs it. 5.1 differs at run time, not just
# in syntax (Set-Content and Out-File default encodings, for one), so a run that
# stayed on 5.1 would execute code nothing tests. BUG-005 was that failure: the
# settings merge used `ConvertFrom-Json -AsHashtable`, absent from 5.1, and its
# try/catch skipped the merge silently. That merge is now `dotf deploy` (#2000);
# the single tested runtime is why the re-exec stays.
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
        Write-Host "        This script is verified under PowerShell 7+ only." -ForegroundColor Red
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
# SCRIPTS_DIR per env-contract.json: under the deploy dir, the same shape as
# Linux ($HOME/.dotfiles/scripts) and what dotf doctor's contract check, the
# profile fallback and required_path_entries all expect (WIN-013, #1310). The
# pre-contract location is kept only to be cleaned.
$ScriptsDir = "$DotfilesDest\scripts"
$LegacyScriptsDir = "$env:USERPROFILE\scripts"

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

# SDD-007: dot-source shared deploy helpers (Deploy-File).
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
# can yield a false negative for a plugin not present in that listing, so a setup
# run can trigger a real install and hit #59870 -- this wrapper is the second
# layer that catches the false-negative case.
# Complementary to SDD-021 session-start canary in dotf mem session-start
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

# packages.json (CLI-029): the declarative tool catalog `dotf tools` consumes.
# Deploy it beside versions.conf so `dotf tools list` / `install` resolve it from
# $DOTFILES_DIR on this machine, not just from a repo checkout.
$packagesSource = "$DotfilesDir\packages.json"
if (Test-Path $packagesSource) {
    Ensure-Directory $DotfilesDest
    Copy-Item $packagesSource "$DotfilesDest\" -Force
    Write-Success "packages.json deployed to $DotfilesDest\"
} else {
    Write-Warn "packages.json not found at $packagesSource"
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
        # copilot left this list for packages.json (npm on every OS, AI-038/#1321,
        # ADR-036): `dotf tools install` below converges it, pin as floor.
        # Node.js is the prerequisite of the npm channel (ADR-036 class 3: no
        # cross-OS channel, so winget here, nvm on Linux). It must exist before
        # `dotf tools install` runs below, or bw and opencode cannot install on
        # a clean box. A scoop/nvm node already on PATH is left alone.
        @{ Name = "Node.js LTS"; Cmd = "node"; Id = "OpenJS.NodeJS.LTS" }
        # opencode left this list for packages.json (npm on every OS, AI-034/#1294,
        # ADR-036): `dotf tools install` below converges it. No winget tool carries
        # a version pin any more, so the loop only installs what is absent.
    )
    foreach ($tool in $tools) {
        if (-not (Get-Command $tool.Cmd -ErrorAction SilentlyContinue)) {
            Write-Info "Installing $($tool.Name)..."
            # Install-WingetTool (utils.ps1) reads winget's exit code and checks
            # the command resolves; the old inline call printed "installed"
            # whatever winget did (#2157).
            $result = Install-WingetTool -Cmd $tool.Cmd -Id $tool.Id
            if ($result.Installed) {
                Write-Success "$($tool.Name) installed"
            } else {
                Write-Warn ("$($tool.Name) is not on PATH after $($result.Attempts) winget attempt(s)" +
                    " (exit code $($result.ExitCode)): $($result.Detail)")
            }
        } else {
            Write-Info "$($tool.Name) already installed"
        }
    }
    # Refresh PATH so freshly-installed winget tools are visible to subsequent
    # blocks of this same setup run (otherwise Get-Command misses them until
    # the next shell start; first introduced for BUG-003 so the Copilot config
    # deploy block sees the just-installed `copilot` binary).
    # Sync-SessionPath (utils.ps1) keeps the process PATH too: an entry that
    # exists only in this process (a CI GITHUB_PATH addition, a shell that put a
    # build dir on PATH for the run) was silently dropped by a registry-only
    # rebuild here, and every dotf block after this line then ran against
    # whatever the registry PATH resolved -- on the CI runner, nothing
    # (TEST-003/#1298). Registry first so fresh winget installs still win.
    Sync-SessionPath
} else {
    Write-Warn "winget not found, skipping developer tools installation"
}

# ============================================================================
# 1d. AGENT BINARIES WITH AN OFFICIAL INSTALLER (AI-041, #1325)
# ============================================================================
# On Linux and macOS mise installs Claude Code and the Antigravity CLI
# (versions.conf `# mise: latest`, #2013 P5c), as setup-linux.sh did through
# the vendor installers since ADR-009. Windows never did, so on a clean box
# every block below that gates on `claude` (MCP registration, plugins, both
# session hooks) and the agy config deploy were
# dead until a human installed the binaries by hand. Both vendors ship a
# Windows installer (install.ps1); ADR-036 class 3 (official installer on every
# OS, like uv), run in a child pwsh exactly as the uv/Bun installers are, and
# skipped when the binary is already on PATH - the pattern-setup-script-
# idempotence shape of the Linux blocks.
function Install-AgentBinary {
    param(
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)][string]$Command,
        [Parameter(Mandatory)][string]$InstallerUrl,
        [Parameter(Mandatory)][string]$BinDir
    )
    if (Get-Command $Command -ErrorAction SilentlyContinue) {
        Write-Info "$Name already installed"
        return
    }
    Write-Info "Installing $Name via official installer..."
    $installer = Join-Path $env:TEMP ("$Command-install.ps1")
    try {
        Invoke-RestMethod $InstallerUrl -OutFile $installer
        & (Get-Process -Id $PID).Path -NoProfile -ExecutionPolicy Bypass -File $installer 2>$null
        Remove-Item -Path $installer -Force -ErrorAction SilentlyContinue
        if (($env:PATH -split ';') -notcontains $BinDir) { $env:PATH = "$BinDir;$env:PATH" }
        if (Get-Command $Command -ErrorAction SilentlyContinue) {
            Write-Success "$Name installed"
        } else {
            Write-Warn "$Name installer ran but '$Command' is not on PATH -- open a new shell or install manually ($InstallerUrl)"
        }
    } catch {
        Write-Warn "$Name install failed -- re-run setup or install manually ($InstallerUrl): $_"
    }
}
Install-AgentBinary -Name "Claude Code" -Command "claude" -InstallerUrl "https://claude.ai/install.ps1" -BinDir "$env:USERPROFILE\.local\bin"
Install-AgentBinary -Name "Antigravity CLI (agy)" -Command "agy" -InstallerUrl "https://antigravity.google/cli/install.ps1" -BinDir "$env:LOCALAPPDATA\agy\bin"

# ============================================================================
# 2. DEPLOY CLAUDE CONFIGURATION
# ============================================================================

Write-Info "Deploying Claude configuration..."

# Bulk copy the Claude config files EXCEPT settings.json and CLAUDE.md:
# `dotf deploy` merges settings.json (the claude-settings entry), its hooks
# come from `dotf harness bind`, and CLAUDE.md is a harness instruction file
# that `dotf harness deploy` writes near the end of this script.
$claudeSource = "$DotfilesDir\ai\claude"
if (Test-Path $claudeSource) {
    Copy-Item "$claudeSource\*" "$ClaudeHome\" -Recurse -Force -Exclude 'settings.json', 'CLAUDE.md' -ErrorAction SilentlyContinue
}

# Skills (claude, opencode, agy) are deployed from the committed vault skill
# records by `dotf harness deploy` near the end of this script (SDD-008,
# CLI-026), honoring each skill's targets[].
Ensure-Directory "$ClaudeHome\skills"

# dotf CLI (ADR-020 / WIN-006): install the pinned release binary into ~/.local/bin
# (user-space, no admin) - the PowerShell twin of setup-linux.sh sourcing
# install.sh. goreleaser already publishes the Windows zip, so nothing is
# compiled. Makes the `dotf env path` / `dotf env generate` steps below resolve
# automatically. The preflight of every dotf step below, as on Linux: setup
# cannot finish without it, so a failure stops here (#2013 W4).
$installDotfScript = Join-Path $PSScriptRoot 'install.ps1'
if (Test-Path $installDotfScript) {
    . $installDotfScript
    if (-not (Install-Dotf)) {
        Write-Err "dotf installation failed -- see the error above; setup cannot continue without it"
        exit 1
    }
} else {
    Write-Err "install.ps1 not found -- run setup from the root of a dotfiles checkout"
    exit 1
}

# ============================================================================
# DEPLOY SECRETS SYSTEM (early: before the opencode/agy config blocks below that
# substitute secrets at deploy time, and after dotf is installed above so the
# deploy-time fetch can read the registry). Cross-OS parity with setup-linux.sh,
# whose sensitive/registry deploy is likewise early.
# ============================================================================
Write-Info "Setting up secrets system..."

# Preflight: warn if age identity key is missing. Without it `dotf secrets` can't
# decrypt, so $env:NAN_API_KEY / OPENROUTER_API_KEY / etc. resolve empty --
# opencode + agy then 401 with no clear cause. Non-fatal: encrypted files still
# deploy so a key imported later works without re-running setup.
$ageKey = if ($env:AGE_KEY_PATH) { $env:AGE_KEY_PATH } else { Join-Path $env:USERPROFILE '.config\age\key.txt' }
if (-not (Test-Path -LiteralPath $ageKey)) {
    Write-Warn "age identity key not found at $ageKey"
    Write-Warn "  Encrypted secrets will deploy but won't decrypt on demand."
    Write-Warn "  To enable: place your age identity at `$env:USERPROFILE\.config\age\key.txt"
    Write-Warn "  (or set `$env:AGE_KEY_PATH). Generate: age-keygen -o `$HOME\.config\age\key.txt"
    Write-Warn "  See: docs/runbooks/guide-secrets-governance.md#protocol--first-machine-age-key--ssh"
}

$sensitiveSource = "$DotfilesDir\sensitive"
$sensitiveDest = "$DotfilesDest\sensitive"
if (Test-Path $sensitiveSource) {
    Ensure-Directory $sensitiveDest
    # The var->file mapping moved to secrets/registry.yaml (deployed below); sensitive/
    # holds the encrypted .age blobs and, under dr/, the Bitwarden DR escrow.
    # Recursive, like setup-linux.sh's `cp -rf sensitive/*`: the top-level-only
    # copy this used to be never mirrored dr/, and doctor reported total-loss risk
    # over an escrow sitting in the checkout (WIN-011/#1292).
    Copy-Item -Path (Join-Path $sensitiveSource '*') -Destination $sensitiveDest -Recurse -Force
    $ageCount = (Get-ChildItem -Path $sensitiveDest -Filter '*.age' -File -Recurse -ErrorAction SilentlyContinue | Measure-Object).Count
    Write-Success "Deployed $ageCount encrypted secret files (sensitive/ mirrored recursively)"
} else {
    Write-Warn "Sensitive directory not found at $sensitiveSource"
}

# Deploy the secrets registry (ADR-028 section 2 mapping SSOT). dotf secrets reads it
# from $DotfilesDest\secrets\registry.yaml; without it `dotf secrets {ls,show,run}`
# and the AI-CLI wrappers fail. Mirrors the sensitive/ deploy above.
$registrySource = "$DotfilesDir\secrets\registry.yaml"
if (Test-Path -LiteralPath $registrySource) {
    Ensure-Directory "$DotfilesDest\secrets"
    Copy-Item $registrySource "$DotfilesDest\secrets\" -Force
    Write-Success "Deployed secrets/registry.yaml"
} else {
    Write-Warn "secrets/registry.yaml not found at $registrySource"
}

# Catalog tools (CLI-029): download + checksum-verify the declarative packages.json
# tools (currently sops) into ~/.local/bin via dotf - the same deterministic pattern
# as Install-Dotf, driven by data instead of a per-OS winget loop. Best-effort: an
# offline box or a single failed download must not abort setup (parity with the
# Install-Dotf warning above). Guarded on dotf being on PATH, mirroring the env
# blocks below.
if (Get-Command dotf -ErrorAction SilentlyContinue) {
    dotf tools install
    if ($LASTEXITCODE -ne 0) {
        Write-Warn "dotf tools install failed (continuing; re-run 'dotf tools install')"
    }
    # The catalog installs uv through winget (astral-sh.uv, #2754), which puts
    # its directory on the user PATH only; the MCP block below needs uvx here.
    Sync-SessionPath
}

# Register MCP servers (requires Claude Code CLI, Node.js, uv for the uvx servers)
# It runs after `dotf tools install`, which installs uv: registered before uv,
# a clean box's first run skipped the uvx servers with "prerequisite 'uv' not
# found" and only the second converged (OPS-044, #1361).
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

# Mirror the harness inputs into the deploy dir (WIN-007/#1288): harness\ plus
# every file harness\manifest.json declares as an injection target, so
# `dotf doctor` reads model-map.json / model-pins.json from the copy it checks.
# One Go implementation for both OSes (setup-linux.sh calls the same command).
# Windows never had this block: doctor failed both registries after every setup
# with a remedy ("re-run setup") that could not clear them. Idempotent (prints
# "N updated, M unchanged, K pruned"). It prunes .zsh\, ssh\ and scripts\ of
# files the checkout's history deleted (#2266); harness\ and the secrets stay
# with doctor --fix (#802).
if (Get-Command dotf -ErrorAction SilentlyContinue) {
    dotf harness mirror --repo $DotfilesDir
    if ($LASTEXITCODE -ne 0) {
        Write-Warn "dotf harness mirror reported a gap (above); 'dotf doctor' will report harness drift"
    }
} else {
    Write-Warn "dotf not found; harness inputs were not mirrored"
}

# Phase C daemon supervision (HIVE-118 / hive#176). Install the supervised
# `hive serve` Scheduled Task now that the MCP loop's prerequisite installed/upgraded
# the tool. Gated on hive-vault >= 1.32.0 via the package version (NOT by probing
# `hive service`, which an older hive routes to the blocking stdio server). The MCP
# entry itself is already `hive client` (re-added from mcp-servers.json above).
# Non-fatal: any failure leaves the in-process `hive client` fallback.
# Probe the BINARY, not the installer registry: hive moved to its own installer
# and `uv tool list` stopped seeing a healthy install (AI-028/#791), so this gate
# skipped daemon supervision with "hive <unknown> predates 'hive service'" while
# `hive --version` answered 3.0.0 on the work box. `dotf tools version` is the one
# semver extraction both OSes share (ADR-036).
$hiveVer = $null
if (Get-Command dotf -ErrorAction SilentlyContinue) {
    $hiveVer = (& dotf tools version hive 2>$null | Select-Object -First 1)
    if ($LASTEXITCODE -ne 0 -or -not $hiveVer) { $hiveVer = $null }
}
if ((Get-Command hive -ErrorAction SilentlyContinue) -and $hiveVer -and ([version]$hiveVer -ge [version]'1.32.0')) {
    # Keep the output: since hive 4.3.1, install waits for the daemon to answer and
    # exits 1 with its state, the only clue to why it is down.
    $hiveSvcOut = (& hive service install 2>&1 | Out-String).Trim()
    $hiveSvcRc = $LASTEXITCODE
    if ($hiveSvcRc -eq 0) { Write-Success "Installed hive daemon service (Scheduled Task, v$hiveVer)" }
    else { Write-Warn "hive service install exited $hiveSvcRc (non-fatal; client works via fallback): $hiveSvcOut" }

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
} elseif (Get-Command hive -ErrorAction SilentlyContinue) {
    $shownVer = if ($hiveVer) { $hiveVer } else { '<unknown>' }
    Write-Warn "hive $shownVer predates 'hive service' (need >= 1.32.0); skipping daemon supervision"
}

# Claude Code plugins are installed by the bare `dotf deploy` below, from
# ai/claude/plugins.json: it counts each failure and names its cause (#2336).

# Claude's auto-memory dir is linked to its vault source per project by
# memlink (cli/internal/memlink): the session-start hook links the project a
# session opens, on every OS, and `dotf doctor --fix` repairs the current one,
# a dangling link included. A real memory dir is never moved aside (#1843 B13).

# Vault-hosted skills are no longer junctioned into ~/.claude/skills (that was
# the BUG-100 fragility source). They are migrated into the vault SSOT, compiled
# into committed records, and rendered to a regular copy by `dotf harness
# deploy` near the end of this script (SDD-008, CLI-026).

# poetry is a packages.json uv-tool entry, converged by `dotf tools install`
# above; uv is installed earlier in this script (OPS-044).

# Bun was installed here until OPS-040, for "some Claude Code plugin workers for
# bun:sqlite" -- the conversation-memory plugin worker MEM-002 retired on
# 2026-06-23. No code in this repository invokes bun, bunx or bun run, so setup
# no longer installs a runtime nothing here uses. The shell rc files keep their
# BUN_INSTALL PATH export: a bun the user installed themselves stays reachable,
# setup just stops providing one.

# What the rest of setup will see. The npm-driven pi block below skips silently
# when npm is missing (obsidian and yarn moved to packages.json, OPS-042); naming the state here
# turns "npm not available" into a finding with a cause.
$npmSeen = Get-Command npm -ErrorAction SilentlyContinue
Write-Info ("PATH after the tool installers: {0} entries; npm: {1}" -f (($env:PATH -split ';' | Where-Object { $_ }).Count), $(if ($npmSeen) { $npmSeen.Source } else { 'absent' }))

# ============================================================================
# 2d. OPENCODE (AI-014)
# ============================================================================
# Binary install: packages.json (npm) via dotf tools install (AI-034, ADR-036).
# opencode.jsonc is the `opencode` entry of ai/deploy.json, rendered with its
# {env:VAR} secrets by `dotf deploy` below (SDD-009, #1843 B12).

# opencode's and pi's AGENTS.md (the canonical SSOT, read natively under that
# name) are harness/manifest.json agents.presence targets, deployed by
# `dotf harness deploy` near the end of this script.

# ============================================================================
# 2e. PI CODING AGENT CONFIG (AI-025)
# ============================================================================
# Mirror of the opencode block + setup-linux.sh pi deploy so the two agents are
# interchangeable across Linux/Windows. pi reads %USERPROFILE%\.pi\agent\.
# npm install at the PI_VERSION floor (guarded): Linux and macOS get pi from
# mise (versions.conf `# mise: latest`, #2013 D9), which Windows does not run
# yet (#2013). models.json gets the same {env:VAR} deploy-time
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
        # REFACTOR-013: pin is a MINIMUM - upgrade only when installed < pin so
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

# Agent configs are deployed by `dotf deploy` (CLI-039): one implementation for
# every OS, replacing the per-config copies that lived here and their twins in
# setup-linux.sh. ADR-020 C7 keeps this script on the thin bootstrap; staging,
# rendering, comparing and installing a config is tooling logic and belongs in
# the CLI. The call names NO config: bare `dotf deploy` installs every entry
# ai/deploy.json declares, so a new entry is a manifest edit and not a change to
# two setup scripts -- `dotf deploy pi` left orca-keybindings declared and never
# installed (CLI-054, #1301).
if (Get-Command dotf -ErrorAction SilentlyContinue) {
    # pi's settings.json before deploy, whose pi-compaction entry merges into
    # it: a merge onto a missing file creates one holding only compaction
    # (#1938). The step seeds the repo's copy when the file is absent and keeps
    # its enabledModels equal to the repo's, leaving pi's own keys alone (#1948).
    & dotf converge --only pi-settings
    if ($LASTEXITCODE -ne 0) {
        Write-Warn "dotf converge --only pi-settings failed -- pi's settings.json was not seeded or synced; see 'dotf doctor'"
    }
    & dotf deploy
    if ($LASTEXITCODE -ne 0) {
        Write-Err "dotf deploy failed -- the failed rows above name each config; fix them and run setup again"
        exit 1
    }
    # After deploy, which writes the file the include names (#2207).
    & dotf converge --only git-config
    if ($LASTEXITCODE -ne 0) {
        Write-Warn "dotf converge --only git-config failed -- a dotf older than the flag cannot run it; re-run setup once DOTF_VERSION carries it, or see 'dotf doctor'"
    }
    # Its own call: a dotf older than the step refuses the name, and must not
    # take git-config down with it (#2013 X1).
    & dotf converge --only git-hooks
    if ($LASTEXITCODE -ne 0) {
        Write-Warn "dotf converge --only git-hooks failed -- a dotf older than the step cannot run it; re-run setup once DOTF_VERSION carries it, or see 'dotf doctor'"
    }
} else {
    Write-Err "dotf not on PATH after installing it -- no agent config can be deployed"
    exit 1
}

# pi packages (HARNESS-139, #1628): `dotf pi packages apply` converges pi on
# ai\pi\packages.json in BOTH directions -- installs what is declared, removes
# what is not, and archives the paths the manifest retires -- through pi's own
# CLI. It replaced a loop here that only ever installed. The Go command keeps
# the captured output, the elapsed time, the DOTFILES_SKIP_PI_PACKAGES skip
# (CI-002, #1478), and a warning rather than a failure without pi or npm.
$dotfBin = Get-Command dotf -ErrorAction SilentlyContinue
$dotfPath = if ($dotfBin) { $dotfBin.Source } elseif (Test-Path "$env:USERPROFILE\.local\bin\dotf.exe") { "$env:USERPROFILE\.local\bin\dotf.exe" } else { $null }
if ($dotfPath) {
    & $dotfPath pi packages apply --repo $DotfilesDir
    if ($LASTEXITCODE -ne 0) {
        Write-Warn "pi package reconcile reported a failure (above); run 'dotf pi packages check'"
    }
} else {
    Write-Warn "dotf not found - pi packages not reconciled"
}

# opencode's tui.json is the `opencode-tui` entry of ai/deploy.json, installed
# by the bare `dotf deploy` above (#1843 B11).

# OpenCode commands are deployed from the committed vault skill records by
# `dotf harness deploy` near the end of this script (SDD-008, CLI-026): each
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
$AgyHome = Join-Path $GeminiHome "antigravity-cli"
Ensure-Directory $AgyHome

# agy reads GEMINI_DIR; set it for this session and persist it for later ones.
# ANTIGRAVITY_ENDPOINT / CLOUDCODE_URL were set here too until OPS-040. They had
# no observed effect on routing -- agy issues its metadata-plane calls to a
# hardcoded host regardless -- so setup no longer sets them. Values a previous
# run persisted to the user environment stay behind, inert; scrubbing them would
# mean adding the kind of one-time migration OPS-040 exists to remove.
[Environment]::SetEnvironmentVariable("GEMINI_DIR", "$GeminiHome", "User")
$env:GEMINI_DIR = "$GeminiHome"

# 1. agy settings.json is a `dotf deploy` entry (ai/deploy.json `agy-settings`,
#    AI-042/#1334), merged so agy's own trust and grants survive (#902).

# .geminiignore and AGY.md are `dotf deploy` entries (ai/deploy.json
# `agy-geminiignore`, `agy-instructions`; #1843 B11), deployed by the bare
# `dotf deploy` this script makes: one implementation for every OS.

# 2. agy's MCP server configs (the master under ~/.gemini/config and the
#    hive-vault plugin) are written by the bare `dotf deploy` this script makes
#    and by `dotf converge` (CLI-105, #2384): one implementation for every OS.

# Drop project-local agy state cache from the workspace if agy leaked one
$projAgyCli = Join-Path $DotfilesDir ".antigravitycli"
$projAgyBak = Join-Path $DotfilesDir ".antigravitycli.bak"
if (Test-Path $projAgyCli) { Remove-Item -Recurse -Force $projAgyCli -ErrorAction SilentlyContinue }
if (Test-Path $projAgyBak) { Remove-Item -Recurse -Force $projAgyBak -ErrorAction SilentlyContinue }


# Agy skills (native Shared skills under $GeminiHome\skills and frontmatter-
# stripped flat prompts under $GeminiHome\prompts) are deployed from the
# committed vault skill records by `dotf harness deploy` near the end of this
# script (SDD-008, CLI-026), honoring each skill's targets[]. The former
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
                # instead of replacing - see debug-replace*.ps1 traces).
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

# ~/.gitconfig is co-owned by git, gh and the user, so nothing deploys over it
# (#2207): `dotf deploy` (section above) writes ~/.config/git/dotfiles.gitconfig
# and `dotf converge --only git-config` right after it adds the include and
# gh's credential helper. docs/runbooks/guide-git-config.md.

# tmux: intentionally skipped on Windows (Linux-only -- see tmux.conf in repo root)

# ============================================================================
# 5b. SSH CONFIG
# ============================================================================
# ~/.ssh/config and the public key are `dotf deploy` entries on every OS
# (ai/deploy.json ssh-config, ssh-pubkey; #2013 P7), deployed by the bare
# `dotf deploy` above.

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
# linger as orphans. The Linux twin (CLI-014, one `rm -f` of init-project.sh)
# was deleted by OPS-040 after probing its target absent on msi; this one stays
# because the WIN-013 note below records leftovers of exactly this kind measured
# still present on a real Windows box on 2026-08-27, and that box cannot be
# probed from the Linux session that would delete it.
foreach ($initOrphan in @(
        "$ScriptsDir\init-project.ps1", "$ClaudeHome\init-project.ps1",
        "$ScriptsDir\init-repo-agents.ps1", "$ClaudeHome\init-repo-agents.ps1",
        "$ScriptsDir\init-repo-github-defaults.ps1", "$ClaudeHome\init-repo-github-defaults.ps1")) {
    if (Test-Path $initOrphan) { Remove-Item $initOrphan -Force -ErrorAction SilentlyContinue }
}

# CLI-050: knowledge-crystallize.ps1 retired - `dotf vault crystallize` is the
# sole implementation now, and it needs no per-machine deploy step since it
# ships inside the dotf binary itself.

# WIN-013 (#1310): every script this section deploys now lands in $ScriptsDir
# (the contract's ~\.dotfiles\scripts). Two kinds of leftovers are removed on
# every run, idempotently: scripts retired by earlier tickets (measured still
# present on a real box on 2026-08-27), from both locations; and the live
# scripts' copies in the pre-contract ~\scripts, which would otherwise shadow
# nothing today but drift the day one of them changes. The legacy directory
# itself and the User PATH entry that names it are left alone: pruning a User
# PATH entry is the 2048-char hazard #148 warned about, and the directory may
# hold the user's own scripts.
# MEM-002: claude-mem-heal.ps1 is in this removal list on purpose (the guard
# tests/guard-no-claude-mem.bats strips MEM-002 cleanup blocks before scanning).
$retiredScripts = @(
    "claude-mem-heal.ps1", "claude-session-start.ps1", "diff-check.ps1",
    "doctor.ps1", "healthcheck.ps1", "knowledge-crystallize.ps1",
    "session-handoff.ps1", "orca-hook-tune.ps1")
$deployedScripts = @(
    "profile-heal.ps1", "windows-defaults.ps1",
    "dotfiles-sync.ps1", "obs-cli.ps1")
$removedLeftovers = 0
foreach ($dir in @($ScriptsDir, $LegacyScriptsDir) | Select-Object -Unique) {
    foreach ($name in $retiredScripts) {
        $leftover = Join-Path $dir $name
        if (Test-Path -LiteralPath $leftover) {
            Remove-Item -LiteralPath $leftover -Force -ErrorAction SilentlyContinue
            $removedLeftovers++
        }
    }
}
if ($LegacyScriptsDir -ne $ScriptsDir) {
    foreach ($name in $deployedScripts) {
        $stale = Join-Path $LegacyScriptsDir $name
        if (Test-Path -LiteralPath $stale) {
            Remove-Item -LiteralPath $stale -Force -ErrorAction SilentlyContinue
            $removedLeftovers++
        }
    }
}
if ($removedLeftovers -gt 0) {
    Write-Info "Removed $removedLeftovers retired or relocated script(s) from $ScriptsDir / $LegacyScriptsDir (WIN-013)"
}

# CLI-025: claude-session-start.ps1 + session-handoff.ps1 retired - both session
# hooks now call agnostic `dotf mem` nouns directly (registered below), so there
# is no per-OS shim script left to deploy.

# BUG-020: profile-heal.ps1 reconstructs a corrupted PowerShell profile from
# the SSOT. Pointed at by setup-windows.ps1's preflight error message (BUG-021).
$profileHealSource = "$DotfilesDir\scripts\profile-heal.ps1"
if (Test-Path $profileHealSource) {
    Copy-Item $profileHealSource "$ScriptsDir\" -Force
    Write-Success "Deployed profile-heal.ps1 to $ScriptsDir\"
} else {
    Write-Warn "profile-heal.ps1 not found at $profileHealSource"
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
    # Seed DOTFILES_REPO_DIR into machine.json to the checkout ($DotfilesDir), BEFORE
    # generating the path file, so the cascade (and the generated paths.ps1) resolve
    # the real repo instead of the phantom contract default -- otherwise 'dotf
    # update'/'mem' no-op on a fresh machine (BUG-029/#696). Idempotent; preserves
    # any other overrides.
    dotf env set DOTFILES_REPO_DIR $DotfilesDir | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Seeded DOTFILES_REPO_DIR=$DotfilesDir in machine.json"
    } else {
        Write-Warn "dotf env set DOTFILES_REPO_DIR failed (update/mem fall back to the git walk-up)"
    }
    dotf env generate | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Generated $DotfilesDest\paths.ps1 (dotf env generate)"
        $pathsFile = Join-Path $DotfilesDest 'paths.ps1'
        if (Test-Path $pathsFile) { . $pathsFile }
    } else {
        Write-Warn "dotf env generate failed (profile.ps1 falls back to inline defaults)"
    }
    # The same values at User scope (HKCU\Environment): paths.ps1 only reaches
    # shells that load the profile, and Copilot runs its tool calls with
    # -NoProfile, so DOTFILES_REPO_DIR/DOTFILES_DIR/VAULT_PATH were empty there
    # and dotf could not find the checkout (CLI-058, #1324). Idempotent: only
    # values that differ are written.
    dotf env persist | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Persisted the contract variables at User scope (dotf env persist)"
    } else {
        Write-Warn "dotf env persist failed (profile-less processes will not see the contract variables)"
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

# 7b. DEPLOY SECRETS SYSTEM moved earlier (right after Install-Dotf) so the
# opencode/agy config blocks above substitute their secrets at deploy time; the
# load-secrets eager dot-source was retired (ADR-028 / #587).

# ============================================================================
# 7c. REGISTER SESSIONSTART HOOK
# ============================================================================

# ~/.claude/settings.json is the `claude-settings` entry of ai/deploy.json, merged
# by the `dotf deploy` above (CLI-063, #2000). Its per-key policy and the no-trailer
# attribution are pinned by cli/internal/deploy/claude_settings_test.go.

# HARNESS-045 AC1: hooks are emitted by `dotf harness bind`, for every harness
# declared in harness/manifest.json's `agents.bind`. One Go binary on both OSes,
# not a per-OS shim -- the emitted command (quoted path, .exe suffix) is resolved
# inside it, so this call is byte-identical to the Linux one.
#
# Gate on the CAPABILITY, never on the exit status (lesson 219): a dotf predating
# the subcommand rejects the flags with exit 1, indistinguishable from a genuine
# refusal. TestHarnessHelpListsSubcommands pins the string from the Go side.
$dotfBin = Get-Command dotf -ErrorAction SilentlyContinue
$dotfPath = if ($dotfBin) { $dotfBin.Source } elseif (Test-Path "$env:USERPROFILE\.local\bin\dotf.exe") { "$env:USERPROFILE\.local\bin\dotf.exe" } else { $null }
if (-not $dotfPath) {
    Write-Warn "dotf not found (PATH or ~/.local/bin) -- harness hooks NOT emitted; sessions run without the memory hooks and the gate"
} else {
    $harnessHelp = (& $dotfPath harness --help 2>&1 | Out-String)
    if ($harnessHelp -match '(?m)^\s*bind\s') {
        # -RepoRoot passed EXPLICITLY -- see the twin comment in setup-linux.sh.
        # It matters more here: a PowerShell script does not change the cwd, so
        # `PowerShell -File C:\path\to\dotfiles\setup-windows.ps1` run from
        # anywhere else leaves the resolver with no checkout to walk up to.
        & $dotfPath harness bind --repo-root $DotfilesDir
        if ($LASTEXITCODE -ne 0) {
            Write-Warn "dotf harness bind reported a problem (above) -- 'dotf doctor' will report hook drift"
        }
    } else {
        Write-Warn "installed dotf predates 'dotf harness bind' -- harness hooks NOT emitted; rebuild/reinstall dotf and re-run setup"
    }
}

# ============================================================================
# 8. GITHUB COPILOT CLI
# ============================================================================

Write-Info "Setting up GitHub Copilot CLI..."

# BUG-003: detect the new standalone `copilot` CLI (agentic interface, closer
# to Claude Code than to the legacy gh-copilot extension's suggest/explain
# wrappers). Since AI-038 (#1321, ADR-036) it is an npm catalog tool that
# `dotf tools install` above converges on every OS; this block only deploys
# config when the binary is on PATH.
# Note: AWS Copilot CLI (Amazon.CopilotCLI) also exposes itself as `copilot`.
# If both are installed, Get-Command resolves to the first on PATH. Out-of-
# scope to disambiguate here; <1% population.
$copilotCmd = Get-Command copilot -ErrorAction SilentlyContinue
if ($copilotCmd) {
    Write-Info "GitHub Copilot CLI detected at $($copilotCmd.Source), deploying configuration..."
    $CopilotHome = "$env:USERPROFILE\.copilot"
    Ensure-Directory $CopilotHome

    # copilot-instructions.md is an agents.presence target gated on
    # `requires_command: copilot`, deployed by `dotf harness deploy` near the
    # end of this script. settings.json, config.json and mcp-config.json are
    # `dotf deploy` entries (ai/deploy.json, AI-039/#1322): the first two by
    # MERGE, because the CLI writes both files itself.

    Write-Success "GitHub Copilot CLI configured (aliases cop/cops in profile.ps1)"
} else {
    Write-Info "GitHub Copilot CLI not installed; 'dotf tools install' above converges it from packages.json (npm). Re-run setup or open a new shell if the binary was just installed and PATH needs refresh."
}

# SDD-005 parity (.github/copilot-instructions.md vs ai/copilot/): NOT synced here.
# Setup MUST NEVER write into the checkout -- a checkout write leaves git status
# dirty, and `dotf update` skips any dirty worktree, so a self-deploying machine
# silently stops updating after the first run (dotfiles#694). Parity between the
# two copilot-instructions files is enforced by the fail-loud CI test
# tests/docs-drift.bats, which blocks drift at merge instead of rewriting a
# committed file at deploy time.

# ============================================================================
# Harness deploy (CLI-026): instruction files, skills, agent records, persona
# presence and doctrine, rendered from the committed records under harness\ to
# each agent's path under the user profile as regular copies (never a junction,
# BUG-100). One Go implementation for every OS: setup-linux.sh makes the same
# call. Runs after the per-agent config is deployed; a failure stops setup.
# ============================================================================
& dotf harness deploy --repo-root $DotfilesDir
if ($LASTEXITCODE -ne 0) {
    Write-Err "dotf harness deploy failed -- see its errors above; the agents' harness is incomplete"
    exit 1
}
Write-Success "Harness deployed from records (instructions, skills, agents, presence, doctrine)"

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

# CLI-018: dotf doctor is the single post-setup diagnostic on Windows, replacing
# the retired doctor + healthcheck shell scripts. Non-fatal -- surfaces gaps but
# does NOT alter setup's $LASTEXITCODE.
if (Get-Command dotf -ErrorAction SilentlyContinue) {
    Write-Info "Running post-setup dotf doctor..."
    Write-Host ""
    & dotf doctor
    $doctorExit = $LASTEXITCODE
    Write-Host ""
    if ($doctorExit -ne 0) {
        Write-Warn "dotf doctor reported one or more issues (exit $doctorExit) -- review output above; re-run with the 'hc' alias"
    }
} else {
    Write-Warn "dotf not found on PATH, skipping post-setup diagnostics"
}

# ============================================================================
# 9. SUMMARY
# ============================================================================

# The opt-in self-deploy Scheduled Task (OPS-001) is the `dotf converge` step
# self-deploy, gated on the self-deploy machine role (OPS-053).

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
Write-Host "  - Antigravity:    $GeminiHome\AGY.md  ($AgyHome\)"
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
