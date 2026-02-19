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
# 2. DEPLOY CLAUDE CONFIGURATION
# ============================================================================

Write-Info "Deploying Claude configuration..."

# Copy CLAUDE.md
$claudeMdSource = "$DotfilesDir\ai\claude\CLAUDE.md"
if (Test-Path $claudeMdSource) {
    Copy-Item $claudeMdSource "$ClaudeHome\" -Force
    Write-Success "Deployed CLAUDE.md"
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
        Write-Success "MCP servers registered"
        Write-Warn "Claude Code plugins require manual installation. Run in Claude Code: /install claude-mem@thedotmack"
    } catch {
        Write-Warn "Failed to register MCP servers or install plugins: $_"
    }
} else {
    Write-Warn "Claude Code CLI or npx not found, skipping MCP server registration"
}

# ============================================================================
# 3. DEPLOY GEMINI CONFIGURATION
# ============================================================================

Write-Info "Deploying Gemini configuration..."

# Copy GEMINI.md
$geminiMdSource = "$DotfilesDir\ai\gemini\GEMINI.md"
if (Test-Path $geminiMdSource) {
    Copy-Item $geminiMdSource "$GeminiHome\" -Force
    Write-Success "Deployed GEMINI.md"
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
    Write-Success "Deployed init-project.ps1 to $ScriptsDir\"
} else {
    Write-Warn "init-project.ps1 not found at $initProjectSource"
}

# ============================================================================
# 8. SUMMARY
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
