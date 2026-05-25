<#
.SYNOPSIS
    Post-setup tool and version verification for Windows. Cross-OS sibling of healthcheck.sh.

.DESCRIPTION
    Runs 12 sections of structural assertions against a deployed dotfiles install
    on Windows. Sections 9 (tmux), 11 (ghostty), 12 (drift) emit SKIP with
    explanation because those tools are Linux-only by design or pending follow-up
    specs (WIN-001b, REFACTOR-003, TERM-002).

    Numbered identically to healthcheck.sh so bats parity asserts work cross-OS.

    Exit 0 if no FAIL; exit 1 otherwise. SKIP does not affect exit code.

.EXAMPLE
    pwsh -NoProfile -File scripts\healthcheck.ps1

.EXAMPLE
    hc
    # alias defined in powershell/profile.ps1
#>

[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'

# --- Resolve dotfiles location and versions.conf ---
$script:DotfilesDir = if ($env:DOTFILES_DIR) { $env:DOTFILES_DIR } else { Join-Path $env:USERPROFILE '.dotfiles' }
$script:ScriptDir = $PSScriptRoot

function Read-VersionsConf {
    param([string]$Path)
    $vars = @{}
    if (-not (Test-Path $Path)) { return $vars }
    foreach ($line in Get-Content -LiteralPath $Path) {
        $trimmed = $line.Trim()
        if (-not $trimmed -or $trimmed.StartsWith('#')) { continue }
        if ($trimmed -match '^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$') {
            $vars[$Matches[1]] = $Matches[2].Trim().Trim('"').Trim("'")
        }
    }
    return $vars
}

$versionsCandidates = @(
    (Join-Path $script:DotfilesDir 'versions.conf'),
    (Join-Path (Split-Path $script:ScriptDir -Parent) 'versions.conf')
)
$script:Versions = @{}
foreach ($cand in $versionsCandidates) {
    if (Test-Path -LiteralPath $cand) {
        $script:Versions = Read-VersionsConf -Path $cand
        break
    }
}

# --- Counters and output helpers ---
$script:Passed = 0
$script:Failed = 0
$script:Skipped = 0

function Write-Pass {
    param([string]$Msg)
    $script:Passed++
    Write-Host "  PASS: $Msg" -ForegroundColor Green
}

function Write-Fail {
    param([string]$Msg)
    $script:Failed++
    Write-Host "  FAIL: $Msg" -ForegroundColor Red
}

function Write-Skip {
    param([string]$Name, [string]$Reason)
    $script:Skipped++
    Write-Host "  SKIP: $Name - $Reason" -ForegroundColor Yellow
}

function Write-Section {
    param([string]$Number, [string]$Title)
    Write-Host ''
    Write-Host "[$Number] $Title" -ForegroundColor Cyan
}

function Test-Command {
    param([string]$Name)
    return [bool](Get-Command -Name $Name -ErrorAction SilentlyContinue)
}

function Test-BinaryInDir {
    # Mirrors check_tool_home in healthcheck.sh: looks for <dir>/bin/<binary>(.exe)
    param([string]$Name, [string]$Dir, [string]$Binary)
    if (-not $Dir) { Write-Skip $Name 'variable not set'; return }
    if (-not (Test-Path -LiteralPath $Dir -PathType Container)) {
        Write-Fail "$Name directory missing: $Dir"
        return
    }
    $candidates = @(
        (Join-Path $Dir "bin\$Binary.exe"),
        (Join-Path $Dir "bin\$Binary")
    )
    foreach ($c in $candidates) {
        if (Test-Path -LiteralPath $c -PathType Leaf) {
            Write-Pass "$Name ($Dir)"
            return
        }
    }
    Write-Fail "$Name directory exists but $Binary(.exe) not found in $Dir\bin\"
}

# ============================================================
Write-Host '========================================'
Write-Host '   DOTFILES HEALTH CHECK (Windows)'
Write-Host '========================================'
Write-Host "Checking from: $script:DotfilesDir"

# ==================================================
# 1/12 Core Tools in PATH
# ==================================================
# Required = installed by setup-windows.ps1 (winget array) + the bootstrap
# prerequisites git/pwsh. Workflow-dependent tools (node/npm/docker/kubectl/
# terraform/direnv) moved to section 6 (Optional) because dotfiles does not
# deploy them on Windows -- they're user choice per task.
Write-Section '1/12' 'Core Tools in PATH'

$coreTools = @('git', 'pwsh', 'curl', 'jq', 'eza', 'gh', 'zoxide')
foreach ($tool in $coreTools) {
    if (Test-Command $tool) {
        Write-Pass "$tool found"
    } else {
        Write-Fail "$tool not in PATH"
    }
}

# ==================================================
# 2/12 Versioned Tool Paths
# ==================================================
Write-Section '2/12' 'Versioned Tool Paths'

Test-BinaryInDir -Name 'JAVA_HOME'   -Dir $env:JAVA_HOME   -Binary 'java'
Test-BinaryInDir -Name 'MAVEN_HOME'  -Dir $env:MAVEN_HOME  -Binary 'mvn'
Test-BinaryInDir -Name 'PYTHON_HOME' -Dir $env:PYTHON_HOME -Binary 'python'
Test-BinaryInDir -Name 'GO_HOME'     -Dir $env:GO_HOME     -Binary 'go'

# Minikube has no bin/ subdirectory (binary at root)
if ($env:MINIKUBE_HOME -and (Test-Path -LiteralPath $env:MINIKUBE_HOME -PathType Container)) {
    $mkCandidates = @(
        (Join-Path $env:MINIKUBE_HOME 'minikube.exe'),
        (Join-Path $env:MINIKUBE_HOME 'minikube')
    )
    $found = $false
    foreach ($c in $mkCandidates) {
        if (Test-Path -LiteralPath $c -PathType Leaf) { $found = $true; break }
    }
    if ($found) {
        Write-Pass "MINIKUBE_HOME ($env:MINIKUBE_HOME)"
    } else {
        Write-Fail 'MINIKUBE_HOME directory exists but minikube binary not found'
    }
} elseif ($env:MINIKUBE_HOME) {
    Write-Fail "MINIKUBE_HOME directory missing: $env:MINIKUBE_HOME"
} else {
    Write-Skip 'MINIKUBE_HOME' 'variable not set'
}

# ==================================================
# 3/12 Version Match (versions.conf)
# ==================================================
# Linux deploys language toolchains under $APPS_HOME/{jdk-VERSION,go-VERSION,...}
# and pins versions via versions.conf. Windows installs language toolchains via
# winget into user-chosen paths and does NOT use $APPS_HOME. So this section is
# only meaningful when $env:APPS_HOME is explicitly set; otherwise SKIP the
# whole block (a single line, not 5 individual SKIPs).
Write-Section '3/12' 'Version Match (versions.conf)'

if (-not $env:APPS_HOME) {
    Write-Skip 'version match' '$env:APPS_HOME not set (Windows uses winget, not $APPS_HOME -- skip section)'
} else {
    $appsHome = $env:APPS_HOME

    function Test-VersionMatch {
        param([string]$Name, [string]$Expected, [string]$DirPath)
        if (-not $Expected) {
            Write-Skip "$Name version" 'not set in versions.conf'
            return
        }
        if (Test-Path -LiteralPath $DirPath -PathType Container) {
            Write-Pass "$Name version $Expected (directory exists)"
        } else {
            Write-Fail "$Name expected version $Expected but directory missing: $DirPath"
        }
    }

    Test-VersionMatch 'Java'     $script:Versions['JAVA_VERSION']     (Join-Path $appsHome "jdk-$($script:Versions['JAVA_VERSION'])")
    Test-VersionMatch 'Maven'    $script:Versions['MAVEN_VERSION']    (Join-Path $appsHome "apache-maven-$($script:Versions['MAVEN_VERSION'])")
    Test-VersionMatch 'Python'   $script:Versions['PYTHON_VERSION']   (Join-Path $appsHome "python-$($script:Versions['PYTHON_VERSION'])")
    Test-VersionMatch 'Minikube' $script:Versions['MINIKUBE_VERSION'] (Join-Path $appsHome "minikube-$($script:Versions['MINIKUBE_VERSION'])")
    Test-VersionMatch 'Go'       $script:Versions['GO_VERSION']       (Join-Path $appsHome "go-$($script:Versions['GO_VERSION'])")
}

# ==================================================
# 4/12 Key Files / Junctions
# ==================================================
# Windows uses file copies + the .dotfiles junction (BUG-012) instead of POSIX
# symlinks. We check existence rather than symlink-ness.
Write-Section '4/12' 'Key Files / Junctions'

function Test-DeployedFile {
    param([string]$Path, [string]$Name)
    if (Test-Path -LiteralPath $Path) {
        Write-Pass "$Name exists ($Path)"
    } else {
        Write-Fail "$Name missing: $Path"
    }
}

Test-DeployedFile (Join-Path $env:USERPROFILE '.dotfiles')          '.dotfiles directory'
# Use $PROFILE so the check tracks whatever setup-windows.ps1 deployed
# (Microsoft.PowerShell_profile.ps1 under pwsh 7, profile.ps1 under WinPS 5.1).
Test-DeployedFile $PROFILE                                          "PowerShell profile ($([System.IO.Path]::GetFileName($PROFILE)))"
Test-DeployedFile (Join-Path $env:USERPROFILE '.claude\CLAUDE.md')  'CLAUDE.md'
Test-DeployedFile (Join-Path $env:USERPROFILE '.gemini\GEMINI.md')  'GEMINI.md'
Test-DeployedFile (Join-Path $env:USERPROFILE '.ssh\config')        'SSH config'

# BUG-014 canonical install-state assertion (primary).
# The BUG-012 junction check below validates a proxy artifact (filesystem
# junction) and PASSes even when the plugin is never registered in
# installed_plugins.json. This primary check closes that false-positive class
# by grepping the canonical install record.
$installedPluginsJson = Join-Path $env:USERPROFILE '.claude\plugins\installed_plugins.json'
if (Test-Path -LiteralPath $installedPluginsJson) {
    if (Select-String -Path $installedPluginsJson -SimpleMatch 'claude-mem@thedotmack' -Quiet) {
        Write-Pass 'claude-mem@thedotmack installed (BUG-014 canonical check)'
    } else {
        Write-Fail "claude-mem@thedotmack NOT in installed_plugins.json -- re-run setup-windows.ps1 (BUG-014)"
    }
} else {
    Write-Skip 'claude-mem install state' 'installed_plugins.json missing (Claude Code never ran)'
}

# BUG-012 marketplace junction (claude-mem plugin discovery compat) -- SECONDARY
# diagnostic. Different failure class than BUG-014: install OK in
# installed_plugins.json but plugin hooks still break because the legacy
# junction is missing (UserPromptSubmit fails). Kept after BUG-014 so a true
# install miss surfaces as FAIL on the primary check; this only flags the
# narrower discovery-path issue.
$marketplaceLegacy = Join-Path $env:USERPROFILE '.claude\plugins\marketplaces\thedotmack'
$marketplaceReal   = Join-Path $env:USERPROFILE '.claude\plugins\marketplaces\thedotmack-claude-mem'
if (Test-Path -LiteralPath $marketplaceReal -PathType Container) {
    if (Test-Path -LiteralPath $marketplaceLegacy) {
        Write-Pass 'claude-mem marketplace legacy junction present (BUG-012 secondary)'
    } else {
        Write-Fail "claude-mem marketplace legacy junction missing: $marketplaceLegacy (run claude-mem-heal.ps1)"
    }
} else {
    Write-Skip 'claude-mem marketplace junction' 'marketplace not installed'
}

# BUG-015: assert the claude-mem hook's path-resolution cascade succeeds.
# The upstream hook command is bash; run the same one-liner via Git Bash so
# this check reflects what the hook ACTUALLY does at runtime. Prevention
# layer: detects breakage BEFORE the user encounters the intermittent
# UserPromptSubmit "printf: write error: Permission denied" symptom.
$bashCmd = Get-Command bash -ErrorAction SilentlyContinue
if ($bashCmd) {
    # BUG-023 (2026-05-21): the BUG-022 pipe-to-head form still raced
    # under Linux `set -euo pipefail` when 2+ candidates matched (the
    # consumer closed after the first line, leftover printfs got EPIPE,
    # pipefail killed the script). Cross-OS parity: rewrite to
    # materialize candidates first, then iterate in pure bash with
    # `break` -- no consumer-close, no pipe race. Git Bash on Windows
    # doesn't set pipefail in this sub-context so it never observed the
    # symptom, but the race-free form keeps both probes structurally
    # identical (lesson: probe must use the race-free pattern, not
    # faithfully reproduce broken upstream behaviour).
    $cmHookProbe = '_C="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"; _E="${PLUGIN_ROOT:-}"; _CANDS=$({ [ -n "$_E" ] && printf ''%s\n'' "$_E"; ls -dt "$_C/plugins/cache/thedotmack/claude-mem"/[0-9]*/ 2>/dev/null; printf ''%s\n'' "$_C/plugins/marketplaces/thedotmack-claude-mem/plugin"; }); _P=""; while IFS= read -r _R; do _R="${_R%/}"; [ -d "$_R/plugin/scripts" ] && _Q="$_R/plugin" || _Q="$_R"; if [ -f "$_Q/scripts/bun-runner.js" ] && [ -f "$_Q/scripts/worker-service.cjs" ]; then _P="$_Q"; break; fi; done <<<"$_CANDS"; [ -n "$_P" ] && printf ''%s'' "$_P" || exit 1'
    $resolved = & bash -c $cmHookProbe 2>&1
    if ($LASTEXITCODE -eq 0 -and $resolved) {
        Write-Pass "claude-mem hook path resolves to: $resolved (BUG-015)"
    } else {
        Write-Fail 'claude-mem hook path resolution FAILED -- run claude-mem-heal.ps1 (BUG-015)'
    }
} else {
    Write-Skip 'claude-mem hook path check' 'bash not available (install Git for Windows)'
}

# ==================================================
# 5/12 Environment Variables
# ==================================================
# DOTFILES_DIR is the only Windows-required env var (set by powershell/profile.ps1).
# APPS_HOME + language _HOME vars are Linux-deploy-pattern vars; on Windows they
# are optional (user may set them per workflow) -- SKIP if unset, not FAIL.
Write-Section '5/12' 'Environment Variables'

$requiredVars = @('DOTFILES_DIR', 'ANTIGRAVITY_ENDPOINT', 'CLOUDCODE_URL', 'GEMINI_DIR')
$optionalVars = @('APPS_HOME', 'JAVA_HOME', 'MAVEN_HOME', 'PYTHON_HOME', 'GO_HOME', 'MINIKUBE_HOME')

foreach ($v in $requiredVars) {
    $val = [Environment]::GetEnvironmentVariable($v)
    if ($val) {
        Write-Pass "$v is set"
    } else {
        Write-Fail "$v is not set"
    }
}

foreach ($v in $optionalVars) {
    $val = [Environment]::GetEnvironmentVariable($v)
    if ($val) {
        Write-Pass "$v is set"
    } else {
        Write-Skip $v 'optional on Windows (Linux-deploy var)'
    }
}

# ==================================================
# 6/12 Optional Tools
# ==================================================
Write-Section '6/12' 'Optional Tools'

$optionalTools = @(
    'age', 'claude', 'gemini', 'bats', 'helm', 'ansible', 'pip', 'copilot', 'opencode', 'uv',
    # Workflow-dependent tools moved here from sec 1 (Windows doesn't deploy
    # them by default; they're user choice per workflow). Linux's sec 1 keeps
    # them required because setup-linux.sh installs the standard dev set.
    'node', 'npm', 'docker', 'kubectl', 'terraform', 'direnv'
)
foreach ($tool in $optionalTools) {
    if (Test-Command $tool) {
        Write-Pass "$tool found"
    } else {
        Write-Skip $tool 'not installed'
    }
}

# ==================================================
# 7/12 Knowledge Vault
# ==================================================
Write-Section '7/12' 'Knowledge Vault'

$vaultDir = if ($env:VAULT_DIR) { $env:VAULT_DIR } else { Join-Path $env:USERPROFILE 'Projects\knowledge' }

if (Test-Path -LiteralPath $vaultDir -PathType Container) {
    Write-Pass "Vault directory exists ($vaultDir)"
} else {
    Write-Fail "Vault directory missing: $vaultDir"
}

$obsidianDir = Join-Path $vaultDir '.obsidian'
if (Test-Path -LiteralPath $obsidianDir -PathType Container) {
    Write-Pass '.obsidian/ configured'
} else {
    Write-Fail '.obsidian/ directory missing'
}

$typesFile = Join-Path $obsidianDir 'types.json'
if (Test-Path -LiteralPath $typesFile -PathType Leaf) {
    Write-Pass 'types.json present'
} else {
    Write-Fail 'types.json missing (property schema)'
}

if (Test-Command 'obsidian') {
    Write-Pass 'Obsidian CLI in PATH'
} else {
    Write-Fail 'Obsidian CLI not in PATH'
}

# WIN-002a: vault-health.sh is Linux-only by design (no .ps1 sibling). On
# Windows the .sh is intentionally NOT deployed to $ScriptsDir, so an
# existence check always FAILs and a Skip-when-present/Fail-when-missing
# branch is inverted relative to its leading comment. Collapse to an
# unconditional Skip with the same rationale.
Write-Skip 'vault-health.sh' 'Linux-only script (no .ps1 sibling; runs under WSL or Git Bash if needed)'

$linterConfig = Join-Path $obsidianDir 'plugins\obsidian-linter\data.json'
if (Test-Path -LiteralPath $linterConfig -PathType Leaf) {
    if (Select-String -LiteralPath $linterConfig -Pattern '"lintOnSave": true' -Quiet) {
        Write-Pass 'Linter lintOnSave enabled'
    } else {
        Write-Fail 'Linter lintOnSave disabled'
    }
} else {
    Write-Skip 'Linter config' 'obsidian-linter not installed'
}

foreach ($subdir in @('00_meta', '10_projects', '40_resources')) {
    if (Test-Path -LiteralPath (Join-Path $vaultDir $subdir) -PathType Container) {
        Write-Pass "Vault directory: $subdir/"
    } else {
        Write-Fail "Vault directory missing: $subdir/"
    }
}

# ==================================================
# 8/12 Secrets Integrity
# ==================================================
Write-Section '8/12' 'Secrets Integrity'

$secretsDir = Join-Path $script:DotfilesDir 'sensitive'
$mappingFile = Join-Path $secretsDir 'env-mapping.conf'

if (Test-Path -LiteralPath $mappingFile -PathType Leaf) {
    Write-Pass 'env-mapping.conf exists'

    $mappedBases = New-Object 'System.Collections.Generic.HashSet[string]'

    foreach ($rawLine in Get-Content -LiteralPath $mappingFile) {
        $line = $rawLine.Trim()
        if (-not $line -or $line.StartsWith('#')) { continue }
        if ($line -notmatch '=') { continue }

        $parts = $line -split '=', 2
        $key = $parts[0].Trim()
        $val = $parts[1].Trim()

        # File-secrets syntax: @VAR=filename>dest. Extract base before '>'.
        if ($key.StartsWith('@')) {
            $base = ($val -split '>', 2)[0]
            $displayKey = "$($key.Substring(1)) [file]"
        } else {
            $base = $val
            $displayKey = $key
        }

        $ageFile = Join-Path $secretsDir "$base.secret.age"
        if (Test-Path -LiteralPath $ageFile -PathType Leaf) {
            Write-Pass "$displayKey -> $base.secret.age"
        } else {
            Write-Fail "$displayKey -> $base.secret.age (missing)"
        }
        [void]$mappedBases.Add($base)
    }

    # Orphan detection: .age files with no mapping
    if (Test-Path -LiteralPath $secretsDir -PathType Container) {
        Get-ChildItem -LiteralPath $secretsDir -Filter '*.secret.age' -ErrorAction SilentlyContinue | ForEach-Object {
            $base = $_.Name -replace '\.secret\.age$', ''
            if (-not $mappedBases.Contains($base)) {
                Write-Fail "Orphan: $base.secret.age (no mapping)"
            }
        }
    }
} else {
    Write-Fail 'env-mapping.conf not found'
}

# ==================================================
# 9/12 tmux
# ==================================================
Write-Section '9/12' 'tmux'
Write-Skip 'tmux' 'Linux-only by design (no Windows port planned; use WSL if needed)'

# ==================================================
# 10/12 OpenCode
# ==================================================
Write-Section '10/12' 'OpenCode'

$opencodeCfg = Join-Path $env:USERPROFILE '.config\opencode\opencode.jsonc'

if (Test-Command 'opencode') {
    $ocVersion = (& opencode --version 2>&1 | Select-Object -First 1)
    Write-Pass "opencode in PATH: $ocVersion"
} else {
    Write-Skip 'opencode binary' 'not installed (winget install SST.opencode, or re-run setup-windows.ps1)'
}

if (Test-Path -LiteralPath $opencodeCfg -PathType Leaf) {
    Write-Pass "opencode.jsonc deployed: $opencodeCfg"
    if (Select-String -LiteralPath $opencodeCfg -Pattern '"\$schema":' -Quiet) {
        Write-Pass 'opencode.jsonc has $schema declaration'
    } else {
        Write-Fail 'opencode.jsonc missing $schema declaration (run setup-windows.ps1 to redeploy)'
    }
} else {
    Write-Skip 'opencode.jsonc' "not deployed at $opencodeCfg (run setup-windows.ps1)"
}

# ==================================================
# 11/12 Ghostty
# ==================================================
Write-Section '11/12' 'Ghostty'
Write-Skip 'ghostty' 'Windows port not yet scheduled (TERM-001 is Linux-only; open TERM-002 to enable this section)'

# ==================================================
# 12/12 Repo - Deploy-Dir Drift
# ==================================================
Write-Section '12/12' 'Repo - Deploy-Dir Drift'

# REFACTOR-003: invoke diff-check.ps1 (port of diff-check.sh). Non-fatal:
# surfaces drift as FAIL line but does NOT alter healthcheck's overall
# exit code beyond the existing FAIL-count semantics.
$diffCheckScript = Join-Path $script:ScriptDir 'diff-check.ps1'
if (Test-Path -LiteralPath $diffCheckScript -PathType Leaf) {
    # Capture output; suppress the script's own colored echoes since we want
    # the healthcheck pass/fail framing only.
    $null = & pwsh -NoProfile -File $diffCheckScript 2>&1
    $diffExit = $LASTEXITCODE
    switch ($diffExit) {
        0 { Write-Pass 'No drift between repo and deploy-dir' }
        1 { Write-Fail 'Drift detected -- run setup-windows.ps1 or `dch -VerboseOutput` to see details' }
        default { Write-Fail "diff-check.ps1 exited with code $diffExit (setup error)" }
    }
} else {
    Write-Skip 'diff-check' "diff-check.ps1 not deployed at $diffCheckScript (run setup-windows.ps1)"
}

# ==================================================
# 13/13 Antigravity CLI Health
# ==================================================
Write-Section '13/13' 'Antigravity CLI Health'

if (Test-Command 'agy') {
    # Verify production endpoint
    $agyEndpoint = [Environment]::GetEnvironmentVariable('ANTIGRAVITY_ENDPOINT')
    if ($agyEndpoint -eq 'https://cloudcode-pa.googleapis.com') {
        Write-Pass 'ANTIGRAVITY_ENDPOINT set to production'
    } else {
        Write-Fail "ANTIGRAVITY_ENDPOINT is not production: $agyEndpoint"
    }

    $geminiHome = if ($env:GEMINI_HOME) { $env:GEMINI_HOME } else { Join-Path $env:USERPROFILE '.gemini' }
    $agyData = if ($env:AGY_APP_DATA) { $env:AGY_APP_DATA } else { Join-Path $geminiHome 'antigravity-cli' }

    # Verify valid MCP config (Flat-file strategy BUG-100 parity)
    $masterConfig = Join-Path $geminiHome 'mcp_config.json'
    if (Test-Path $masterConfig -PathType Leaf) {
        if (Get-Command jq -ErrorAction SilentlyContinue) {
            $null = Get-Content $masterConfig -Raw | jq '.' 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Pass 'Master mcp_config.json is valid (flat file)'
            } else {
                Write-Fail 'Master mcp_config.json invalid JSON'
            }
        } else {
            Write-Pass 'Master mcp_config.json exists (jq not available for validation)'
        }
    } else {
        Write-Fail "Master mcp_config.json missing at $masterConfig"
    }

    $agyConfig = Join-Path $agyData 'mcp_config.json'
    if (Test-Path $agyConfig -PathType Leaf) {
        Write-Pass 'Agy mcp_config.json copy exists'
    } else {
        Write-Fail "Agy mcp_config.json missing at $agyConfig"
    }

    # Verify legacy compatibility
    if (Test-Command 'gemini') {
        $null = & gemini --version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Pass 'Legacy gemini-cli is compatible'
        } else {
            Write-Fail 'Legacy gemini-cli schema error (settings.json incompatible)'
        }
    }
} else {
    Write-Skip 'Antigravity CLI' 'not installed'
}

# ==================================================
Write-Host ''
Write-Host '========================================'
Write-Host ("Results: {0} passed, {1} failed, {2} skipped" -f $script:Passed, $script:Failed, $script:Skipped)
Write-Host '========================================'

if ($script:Failed -gt 0) {
    exit 1
}
exit 0
