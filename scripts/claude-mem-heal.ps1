<#
.SYNOPSIS
    Idempotently repair the thedotmack/claude-mem plugin cache when the
    published artifact ships broken (v12.7.4, v13.0.0+).

.DESCRIPTION
    Two known issues this script papers over:

      1. .mcp.json embeds shell parameter expansion (${_R%/}) that Claude
         Code's MCP loader misreads as an env-var name. See upstream
         issue #2385. Rewrite the file to the simpler v10.6.3 form.

      2. v13.0.0+ declares "zod": "^4.3.6" in package.json but the
         published bun.lock and node_modules/ omit it, so
         worker-service.cjs crashes with
         "Cannot find module 'zod/v3'". Install zod in place with
         --ignore-scripts so other native deps (tree-sitter) do not
         trigger MSBuild on Windows.

    Both upstream errors revert on /plugin update, so this is invoked
    from claude-session-start.ps1 to self-heal at session start.

    Mirrors scripts/claude-mem-heal.sh (Linux).

.NOTES
    Behaviour: silent on healthy installs (exit 0, no output). Prints one
    line per heal action taken so the SessionStart hook can surface it.
    Always exits 0 -- never blocks session start on transient failures.

.PARAMETER Verbose
    When passed, always logs what was checked.

.EXAMPLE
    pwsh -NoProfile -File claude-mem-heal.ps1
    pwsh -NoProfile -File claude-mem-heal.ps1 -VerboseOutput
#>

[CmdletBinding()]
param(
    [switch]$VerboseOutput
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'

# Promote param to script-scope so PSScriptAnalyzer sees the reference
# explicitly; downstream functions read it via $script:VerboseOutput.
$script:VerboseOutput = $VerboseOutput.IsPresent

$ClaudeDir = if ($env:CLAUDE_CONFIG_DIR) { $env:CLAUDE_CONFIG_DIR } else { Join-Path $env:USERPROFILE '.claude' }
$CacheRoot = Join-Path $ClaudeDir 'plugins\cache\thedotmack\claude-mem'
$MarketplaceDir = Join-Path $ClaudeDir 'plugins\marketplaces\thedotmack\plugin'
$MarketplaceDirActual = Join-Path $ClaudeDir 'plugins\marketplaces\thedotmack-claude-mem\plugin'

function Write-HealLog {
    param([string]$Message)
    Write-Output "[claude-mem-heal] $Message"
}

function Write-HealVerbose {
    param([string]$Message)
    if ($script:VerboseOutput) { Write-HealLog $Message }
}

# BUG-012: Claude Code clones the claude-mem marketplace under the GitHub repo
# name `thedotmack-claude-mem\`, but the plugin's bundled hooks.json hardcodes
# the legacy fallback `marketplaces\thedotmack\plugin\scripts\...`. Without a
# compatibility junction, plugin hooks fail discovery when CLAUDE_PLUGIN_ROOT
# is unset (UserPromptSubmit blocked with `printf: write error: Permission
# denied` under Git Bash on Windows). Create a Junction so the legacy path
# resolves to the actual install. Idempotent: skip if source dir missing or
# target path already present. Junction requires no admin privileges on NTFS.
function Repair-MarketplaceCompatJunction {
    $legacy = Join-Path $ClaudeDir 'plugins\marketplaces\thedotmack'
    $actual = Join-Path $ClaudeDir 'plugins\marketplaces\thedotmack-claude-mem'

    if (-not (Test-Path $actual -PathType Container)) {
        Write-HealVerbose "no thedotmack-claude-mem marketplace at $actual"
        return
    }
    if (Test-Path $legacy) {
        Write-HealVerbose "legacy marketplace path already present: $legacy"
        return
    }
    try {
        $null = New-Item -ItemType Junction -Path $legacy -Target $actual -ErrorAction Stop
        Write-HealLog "created legacy marketplace junction: $legacy -> thedotmack-claude-mem"
    } catch {
        Write-HealLog "ERROR: failed to create junction ${legacy}: $($_.Exception.Message)"
    }
}

# Replace a broken .mcp.json with the v10.6.3 form. Idempotent: only
# rewrites if the file contains the offending ${_R%/} pattern.
function Repair-McpJson {
    param([string]$Target)

    if (-not (Test-Path $Target)) {
        Write-HealVerbose "no .mcp.json at $Target"
        return
    }

    $content = Get-Content $Target -Raw -ErrorAction SilentlyContinue
    if (-not $content -or ($content -notmatch '\$\{_R%/\}')) {
        Write-HealVerbose ".mcp.json already healthy: $Target"
        return
    }

    $healthy = @'
{
  "mcpServers": {
    "mcp-search": {
      "type": "stdio",
      "command": "node",
      "args": [
        "${CLAUDE_PLUGIN_ROOT}/scripts/mcp-server.cjs"
      ]
    }
  }
}
'@
    Set-Content -Path $Target -Value $healthy -Encoding UTF8 -NoNewline
    Write-HealLog "patched .mcp.json: $Target"
}

# Install the zod runtime dep if package.json declares it but it isn't
# installed. Idempotent: skips when node_modules/zod exists.
# --ignore-scripts is mandatory on Windows: other deps (tree-sitter)
# trigger node-gyp + MSBuild on postinstall and fail on machines without
# the C++ build toolchain. We only need zod's pure-JS files.
function Repair-ZodDep {
    param([string]$PluginDir)

    $pkg = Join-Path $PluginDir 'package.json'
    if (-not (Test-Path $pkg)) {
        Write-HealVerbose "no package.json at $PluginDir"
        return
    }

    $pkgContent = Get-Content $pkg -Raw -ErrorAction SilentlyContinue
    if (-not $pkgContent -or ($pkgContent -notmatch '"zod"')) {
        Write-HealVerbose "no zod dep in $pkg"
        return
    }

    $zodDir = Join-Path $PluginDir 'node_modules\zod'
    if (Test-Path $zodDir) {
        Write-HealVerbose "zod present in $PluginDir"
        return
    }

    if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
        Write-HealLog "ERROR: zod missing in $PluginDir but npm not on PATH -- skipping"
        return
    }

    Push-Location $PluginDir
    try {
        $npmArgs = @('install', '--no-save', '--no-package-lock', '--no-audit', '--no-fund', '--ignore-scripts', 'zod@^4.3.6')
        if (-not $script:VerboseOutput) { $npmArgs += '--silent' }
        $null = & npm @npmArgs 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-HealLog "installed missing zod dep in $PluginDir"
        } else {
            Write-HealLog "ERROR: npm install zod failed in $PluginDir (exit $LASTEXITCODE)"
        }
    } finally {
        Pop-Location
    }
}

function Repair-PluginDir {
    param([string]$Dir)

    if (-not (Test-Path $Dir -PathType Container)) { return }
    Repair-McpJson -Target (Join-Path $Dir '.mcp.json')
    Repair-ZodDep -PluginDir $Dir
}

# Heal every cached version (so a rolled-back /plugin doesn't surprise us)
# plus the marketplace copy used as fallback by the discovery logic.
if (Test-Path $CacheRoot -PathType Container) {
    foreach ($versionDir in (Get-ChildItem $CacheRoot -Directory -ErrorAction SilentlyContinue)) {
        Repair-PluginDir -Dir $versionDir.FullName
    }
} else {
    Write-HealVerbose "no cache dir at $CacheRoot"
}

Repair-MarketplaceCompatJunction
Repair-PluginDir -Dir $MarketplaceDir
Repair-PluginDir -Dir $MarketplaceDirActual

exit 0
