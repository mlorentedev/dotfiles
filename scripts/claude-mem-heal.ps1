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

# Replace a broken .mcp.json with a healthy form. BUG-016 (2026-05-21):
# extended to detect v13.x cascading-printf pattern alongside v12.7.4's
# `${_R%/}` literal. v13.x triggers the upstream EPIPE race documented in
# thedotmack/claude-mem#2607 (causes `/mcp ... -32000` failures intermittently
# on Windows Git Bash).
#
# The healthy form mirrors the v13.x cascade structure (so it works whether
# or not Claude Code sets CLAUDE_PLUGIN_ROOT) but pipes the consumer's
# matches through `head -n1` instead of breaking the inner `while` loop --
# this drains the entire producer pipe, eliminating the EPIPE writes that
# trigger the upstream bug.
#
# Idempotent: skips when neither v12.7.4 nor v13.x signature present.
function Repair-McpJson {
    param([string]$Target)

    if (-not (Test-Path $Target)) {
        Write-HealVerbose "no .mcp.json at $Target"
        return
    }

    $content = Get-Content $Target -Raw -ErrorAction SilentlyContinue
    if (-not $content) {
        Write-HealVerbose ".mcp.json unreadable: $Target"
        return
    }
    $hasV12 = $content -match '\$\{_R%/\}'
    $hasV13 = ($content -match '"sh"\s*,\s*\r?\n?\s*"args"') -and ($content -match 'while IFS=')
    if (-not $hasV12 -and -not $hasV13) {
        Write-HealVerbose ".mcp.json already healthy: $Target"
        return
    }

    $healthy = @'
{
  "mcpServers": {
    "mcp-search": {
      "type": "stdio",
      "command": "sh",
      "args": [
        "-c",
        "_C=\"${CLAUDE_CONFIG_DIR:-$HOME/.claude}\"; _E=\"${CLAUDE_PLUGIN_ROOT:-${PLUGIN_ROOT:-}}\"; _P=$({ [ -n \"$_E\" ] && printf '%s\\n' \"$_E\"; ls -dt \"$_C/plugins/cache/thedotmack/claude-mem\"/[0-9]*/ 2>/dev/null; printf '%s\\n' \"$_C/plugins/marketplaces/thedotmack-claude-mem/plugin\" \"$_C/plugins/marketplaces/thedotmack/plugin\"; } | while IFS= read -r _R; do _R=\"${_R%/}\"; [ -d \"$_R/plugin/scripts\" ] && _Q=\"$_R/plugin\" || _Q=\"$_R\"; [ -f \"$_Q/scripts/mcp-server.cjs\" ] && printf '%s\\n' \"$_Q\"; done | head -n1); [ -n \"$_P\" ] || { echo 'claude-mem: mcp server not found' >&2; exit 1; }; exec node \"$_P/scripts/mcp-server.cjs\""
      ]
    }
  }
}
'@
    Set-Content -Path $Target -Value $healthy -Encoding UTF8 -NoNewline
    if ($hasV13) {
        Write-HealLog "patched .mcp.json (v13.x cascade -> head -n1 race-free form): $Target"
    } else {
        Write-HealLog "patched .mcp.json (v12.7.4 -> head -n1 race-free form): $Target"
    }
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

# BUG-017 (2026-05-21) + 2026-05-27-claude-mem-heal-consumer-epipe (2026-06-06):
# rewrite the path-resolution pipe so the consumer never closes early while the
# inner `while` is still writing. BUG-017 used `}; done | head -n1`, but
# `head -n1` IS an early-closing consumer: with >=2 matching candidates the loop
# writes a 2nd line after head has closed its stdin -> consumer-side EPIPE. Node
# ignores SIGPIPE and Claude Code spawns hooks with that disposition, so the
# failed write is REPORTED (`printf: write error`) instead of dying silently,
# surfacing as a hook-error banner on every tool call. Fix: `sed -n 1p` prints
# only line 1 but reads to EOF (no early `q`), draining the writer so it never
# closes under it. Normalise BOTH the pristine upstream form (`break; }; done`)
# and any already-deployed `}; done | head -n1` form, so existing installs
# converge on the next session with no `/plugin update`. Mirrors
# scripts/claude-mem-heal.sh::heal_hooks_json.
function Repair-HooksJson {
    param([string]$Target)

    if (-not (Test-Path $Target)) {
        Write-HealVerbose "no hooks.json at $Target"
        return
    }
    $content = Get-Content $Target -Raw -ErrorAction SilentlyContinue
    if (-not $content) { return }

    # EPIPE drain: converge both the pristine pipe and the BUG-017-era head -n1
    # consumer onto `sed -n 1p` (prints line 1 but reads to EOF -> no early close).
    $brokenBreak  = 'break; }; done'
    $brokenHeadN1 = '}; done | head -n1'
    $drainForm    = '}; done | sed -n 1p'
    # BUG-018: ALL hooks that terminate with `hook claude-code <X>"` lack
    # Claude Code's {"continue":true,"suppressOutput":true} directive. The
    # bun-runner empty-stdin diagnostic (upstream claude-mem#2188) goes to
    # stdout and Claude Code blocks the operation. Regex captures all 5
    # (session-init / context / observation / file-context / summarize). The
    # Setup hook's version-check.js terminator is left untouched -- not on the
    # user hot path (fires only on plugin install/update).
    $pattern018     = 'hook claude-code ([a-z][a-z-]*)"'
    $replacement018 = 'hook claude-code $1 2>/dev/null; echo ''{\"continue\":true,\"suppressOutput\":true}''"'

    $hasBreak  = $content.Contains($brokenBreak)
    $hasHeadN1 = $content.Contains($brokenHeadN1)
    $has018    = $content -match $pattern018
    if (-not $hasBreak -and -not $hasHeadN1 -and -not $has018) {
        Write-HealVerbose "hooks.json already healthy: $Target"
        return
    }

    $patched = $content
    $epipeFixed = 0
    if ($hasBreak) {
        $epipeFixed += ([regex]::Matches($patched, [regex]::Escape($brokenBreak))).Count
        $patched = $patched.Replace($brokenBreak, $drainForm)
    }
    if ($hasHeadN1) {
        $epipeFixed += ([regex]::Matches($patched, [regex]::Escape($brokenHeadN1))).Count
        $patched = $patched.Replace($brokenHeadN1, $drainForm)
    }
    if ($epipeFixed -gt 0) {
        Write-HealLog "patched hooks.json (EPIPE drain, $epipeFixed hook(s) -> sed -n 1p): $Target"
    }
    if ($has018) {
        $count018 = ([regex]::Matches($patched, $pattern018)).Count
        $patched = $patched -replace $pattern018, $replacement018
        Write-HealLog "patched hooks.json ($count018 hook(s) -> BUG-018 continue directive): $Target"
    }
    Set-Content -Path $Target -Value $patched -Encoding UTF8 -NoNewline
}

function Repair-PluginDir {
    param([string]$Dir)

    if (-not (Test-Path $Dir -PathType Container)) { return }
    Repair-McpJson -Target (Join-Path $Dir '.mcp.json')
    Repair-HooksJson -Target (Join-Path $Dir 'hooks\hooks.json')
    Repair-HooksJson -Target (Join-Path $Dir 'plugin\hooks\hooks.json')
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
