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
Ensure-Directory $GeminiHome
Ensure-Directory "$GeminiHome\prompts"
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
        @{ Name = "zoxide"; Cmd = "zoxide"; Id = "ajeetdsouza.zoxide" }
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
} else {
    Write-Warn "winget not found, skipping developer tools installation"
}

# ============================================================================
# 2. DEPLOY CLAUDE CONFIGURATION
# ============================================================================

Write-Info "Deploying Claude configuration..."

# Copy CLAUDE.md
$claudeMdSource = "$DotfilesDir\ai\claude\CLAUDE.md"
if (Test-Path $claudeMdSource) {
    Copy-Item $claudeMdSource "$ClaudeHome\" -Force
    if (Select-String -Path "$ClaudeHome\CLAUDE.md" -Pattern "CORE PRINCIPLE" -Quiet) {
        Write-Success "CLAUDE.md deployed successfully (verified)"
    } else {
        Write-Err "CLAUDE.md deployment failed verification"
    }
} else {
    Write-Warn "CLAUDE.md not found at $claudeMdSource"
}

# Copy skills (full directories)
$skillsSource = "$DotfilesDir\ai\skills"
if (Test-Path $skillsSource) {
    $skillDirs = Get-ChildItem $skillsSource -Directory -ErrorAction SilentlyContinue
    foreach ($skillDir in $skillDirs) {
        $targetDir = "$ClaudeHome\skills\$($skillDir.Name)"
        Ensure-Directory $targetDir
        Copy-Item "$($skillDir.FullName)\*" $targetDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    Write-Success "Deployed skills to $ClaudeHome\skills\"
}

# Register MCP servers (requires Claude Code CLI and Node.js)
$claudeCmd = Get-Command claude -ErrorAction SilentlyContinue
$npxCmd = Get-Command npx -ErrorAction SilentlyContinue
if ($claudeCmd -and $npxCmd) {
    Write-Info "Registering Claude Code MCP servers..."
    try {
        & claude mcp add --transport stdio drawio --scope user -- npx -y @drawio/mcp 2>$null
        & claude mcp add --transport http socket --scope user -- https://mcp.socket.dev/ 2>$null
        & claude mcp add --transport stdio sequential-thinking --scope user -- npx -y @modelcontextprotocol/server-sequential-thinking 2>$null
        & claude mcp add --transport http context7 --scope user -- https://mcp.context7.com/mcp 2>$null
        # Hive vault server (pypi.org/project/hive-vault)
        $uvCmd = Get-Command uv -ErrorAction SilentlyContinue
        if ($uvCmd) {
            & uv tool install --upgrade hive-vault 2>$null
            & claude mcp add --transport stdio hive --scope user -- uvx hive-vault 2>$null
        }
        Write-Success "MCP servers registered"
    } catch {
        Write-Warn "Failed to register MCP servers: $_"
    }
} else {
    Write-Warn "Claude Code CLI or npx not found, skipping MCP server registration"
}

# Claude Code plugins (requires claude CLI)
if ($claudeCmd) {
    Write-Info "Installing Claude Code plugins..."
    $plugins = @(
        "claude-mem@thedotmack",
        "code-simplifier@claude-plugins-official",
        "github@claude-plugins-official",
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
    foreach ($plugin in $plugins) {
        try {
            & claude plugin install $plugin 2>$null | Out-Null
        } catch {
            # Silently continue if a plugin fails
        }
    }
    Write-Success "Claude Code plugins installed"
} else {
    Write-Warn "Claude Code CLI not found, skipping plugin installation"
}

# Deploy auto-memory from vault (see ADR-007)
$VaultProjects = Join-Path $env:USERPROFILE "Projects\knowledge\10_projects"
if (Test-Path $VaultProjects) {
    Write-Info "Deploying auto-memory from vault..."
    $projectDirs = Get-ChildItem -Path $VaultProjects -Directory
    foreach ($projDir in $projectDirs) {
        $memorySource = Join-Path $projDir.FullName "memory"
        if (-not (Test-Path $memorySource)) { continue }

        $projectName = $projDir.Name
        $projectPath = Join-Path $env:USERPROFILE "Projects\$projectName"
        $encodedPath = $projectPath.Replace('\', '-').Replace(':', '')
        $targetDir = Join-Path $env:USERPROFILE ".claude\projects\$encodedPath\memory"

        Ensure-Directory $targetDir
        Copy-Item "$memorySource\*" $targetDir -Recurse -Force -ErrorAction SilentlyContinue
        Write-Success "Deployed auto-memory: $projectName"
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
                    Rename-Item $memDir "$($memDir).migrated"
                    Write-Success "Migrated: $projectName"
                }
            }
        }
    }
}

# ============================================================================
# 2b. DEPLOY AIDER CONFIGURATION
# ============================================================================

Write-Info "Deploying Aider configuration..."

$aiderSource = "$DotfilesDir\ai\aider"
if (Test-Path $aiderSource) {
    $aiderConf = "$aiderSource\aider.conf.yml"
    if (Test-Path $aiderConf) {
        Copy-Item $aiderConf "$env:USERPROFILE\.aider.conf.yml" -Force
        Write-Success "Deployed .aider.conf.yml"
    }
    $aiderModelSettings = "$aiderSource\aider.model.settings.yml"
    if (Test-Path $aiderModelSettings) {
        Copy-Item $aiderModelSettings "$env:USERPROFILE\.aider.model.settings.yml" -Force
        Write-Success "Deployed .aider.model.settings.yml"
    }
}

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

# Install aider via uv (requires Python 3.12 - audioop removed in 3.13)
$uvCmd = Get-Command uv -ErrorAction SilentlyContinue
if ($uvCmd) {
    Write-Info "Installing aider-chat via uv..."
    try {
        & uv tool install --python 3.12 aider-chat 2>$null
        Write-Success "Aider installed"
    } catch {
        Write-Warn "Failed to install aider: $_"
    }
} else {
    Write-Warn "uv not found, skipping aider installation"
}

# ============================================================================
# 3. DEPLOY GEMINI CONFIGURATION
# ============================================================================

Write-Info "Deploying Gemini configuration..."

# Copy GEMINI.md
$geminiMdSource = "$DotfilesDir\ai\gemini\GEMINI.md"
if (Test-Path $geminiMdSource) {
    Copy-Item $geminiMdSource "$GeminiHome\" -Force
    if (Select-String -Path "$GeminiHome\GEMINI.md" -Pattern "CORE PRINCIPLE" -Quiet) {
        Write-Success "GEMINI.md deployed successfully (verified)"
    } else {
        Write-Err "GEMINI.md deployment failed verification"
    }
} else {
    Write-Warn "GEMINI.md not found at $geminiMdSource"
}

# Extract Gemini prompts (strip YAML frontmatter from skills)
if (Test-Path $skillsSource) {
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
    Write-Success "Extracted Gemini prompts to $GeminiHome\prompts\"
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

$syncSource = "$DotfilesDir\scripts\dotfiles-sync.ps1"
if (Test-Path $syncSource) {
    Copy-Item $syncSource "$ScriptsDir\" -Force
    Write-Success "Deployed dotfiles-sync.ps1 to $ScriptsDir\"
} else {
    Write-Warn "dotfiles-sync.ps1 not found at $syncSource"
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

Write-Info "Registering Claude Code SessionStart hook..."

$ClaudeSettings = "$ClaudeHome\settings.json"
$sessionStartCmd = "$DotfilesDest\scripts\claude-session-start.ps1"

if (Test-Path $ClaudeSettings) {
    try {
        $settings = Get-Content $ClaudeSettings -Raw | ConvertFrom-Json

        if ($settings.hooks -and $settings.hooks.SessionStart) {
            Write-Info "SessionStart hook already configured, skipping"
        } else {
            # Build the hook entry
            $hookEntry = @{
                matcher = ''
                hooks = @(
                    @{
                        type = 'command'
                        command = "pwsh -NoProfile -File `"$sessionStartCmd`""
                        timeout = 30
                    }
                )
            }

            # Ensure hooks object exists
            if (-not $settings.hooks) {
                $settings | Add-Member -NotePropertyName 'hooks' -NotePropertyValue @{} -Force
            }

            $settings.hooks | Add-Member -NotePropertyName 'SessionStart' -NotePropertyValue @($hookEntry) -Force
            $settings | ConvertTo-Json -Depth 10 | Set-Content $ClaudeSettings -Encoding UTF8
            Write-Success "SessionStart hook registered"
        }
    } catch {
        Write-Warn "Failed to register SessionStart hook: $_"
    }
} else {
    Write-Warn "Claude Code settings.json not found, skipping hook registration"
}

# ============================================================================
# 8. GITHUB COPILOT CLI
# ============================================================================

Write-Info "Setting up GitHub Copilot CLI..."

$ghCmd = Get-Command gh -ErrorAction SilentlyContinue
if ($ghCmd) {
    Write-Info "Installing GitHub Copilot CLI extension..."
    try {
        & gh extension install github/gh-copilot 2>$null
    } catch {
        # Extension may already be installed
    }

    $CopilotHome = "$env:USERPROFILE\.copilot"
    Ensure-Directory $CopilotHome

    $copilotSource = "$DotfilesDir\ai\copilot"
    if (Test-Path $copilotSource) {
        Copy-Item "$copilotSource\*" "$CopilotHome\" -Recurse -Force -ErrorAction SilentlyContinue
        Write-Success "Deployed Copilot config to $CopilotHome\"
    }

    Write-Success "GitHub Copilot CLI configured"
} else {
    Write-Warn "GitHub CLI (gh) not found, skipping Copilot installation"
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
