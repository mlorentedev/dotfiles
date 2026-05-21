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
Write-Section '1/12' 'Core Tools in PATH'

$coreTools = @('git', 'pwsh', 'curl', 'jq', 'eza', 'direnv', 'node', 'npm', 'zoxide', 'docker', 'kubectl', 'terraform')
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
Write-Section '3/12' 'Version Match (versions.conf)'

$appsHome = if ($env:APPS_HOME) { $env:APPS_HOME } else { Join-Path $env:USERPROFILE 'Applications' }

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

# BUG-012 marketplace junction (claude-mem plugin discovery compat)
$marketplaceLegacy = Join-Path $env:USERPROFILE '.claude\plugins\marketplaces\thedotmack'
$marketplaceReal   = Join-Path $env:USERPROFILE '.claude\plugins\marketplaces\thedotmack-claude-mem'
if (Test-Path -LiteralPath $marketplaceReal -PathType Container) {
    if (Test-Path -LiteralPath $marketplaceLegacy) {
        Write-Pass 'claude-mem marketplace legacy junction present (BUG-012)'
    } else {
        Write-Fail "claude-mem marketplace legacy junction missing: $marketplaceLegacy (run claude-mem-heal.ps1)"
    }
} else {
    Write-Skip 'claude-mem marketplace junction' 'marketplace not installed'
}

# ==================================================
# 5/12 Environment Variables
# ==================================================
Write-Section '5/12' 'Environment Variables'

$envVars = @('DOTFILES_DIR', 'APPS_HOME', 'JAVA_HOME', 'MAVEN_HOME', 'PYTHON_HOME', 'GO_HOME', 'MINIKUBE_HOME')
foreach ($v in $envVars) {
    $val = [Environment]::GetEnvironmentVariable($v)
    if ($val) {
        Write-Pass "$v is set"
    } else {
        Write-Fail "$v is not set"
    }
}

# ==================================================
# 6/12 Optional Tools
# ==================================================
Write-Section '6/12' 'Optional Tools'

$optionalTools = @('age', 'gh', 'claude', 'gemini', 'bats', 'helm', 'ansible', 'pip', 'copilot', 'opencode', 'uv')
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

# vault-health.sh is Linux-only (no .ps1 sibling yet); skip with explanation
$vaultHealthSh = Join-Path $script:ScriptDir 'vault-health.sh'
if (Test-Path -LiteralPath $vaultHealthSh) {
    Write-Skip 'vault-health.sh' 'Linux-only script (no .ps1 sibling; runs under WSL or Git Bash if needed)'
} else {
    Write-Fail "vault-health.sh missing at $vaultHealthSh"
}

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
    Write-Skip 'opencode binary' 'not installed (AI-014 admin-conditional; deploy via setup-windows.ps1 when ready)'
}

if (Test-Path -LiteralPath $opencodeCfg -PathType Leaf) {
    Write-Pass "opencode.jsonc deployed: $opencodeCfg"
    if (Select-String -LiteralPath $opencodeCfg -Pattern '"\$schema":' -Quiet) {
        Write-Pass 'opencode.jsonc has $schema declaration'
    } else {
        Write-Fail 'opencode.jsonc missing $schema declaration (run setup-windows.ps1 to redeploy)'
    }
} else {
    Write-Skip 'opencode.jsonc' "not deployed at $opencodeCfg (AI-014 pending)"
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
Write-Skip 'diff-check' 'diff-check.ps1 not implemented (REFACTOR-003 queued)'

# ==================================================
Write-Host ''
Write-Host '========================================'
Write-Host ("Results: {0} passed, {1} failed, {2} skipped" -f $script:Passed, $script:Failed, $script:Skipped)
Write-Host '========================================'

if ($script:Failed -gt 0) {
    exit 1
}
exit 0
