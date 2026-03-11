# PowerShell Profile for Dotfiles
# ================================
# This profile is deployed by setup-windows.ps1
# Source: dotfiles/powershell/profile.ps1

# ============================================================================
# ALIASES
# ============================================================================

# AI Tools
Set-Alias -Name c -Value claude
Set-Alias -Name g -Value gemini

# Aider tiers (OpenRouter)
function ai { aider @Args }
function aic { aider --model openrouter/qwen/qwen3-coder-next @Args }
function aia { aider --architect --model openrouter/deepseek/deepseek-v3.2-speciale @Args }

# ============================================================================
# FUNCTIONS
# ============================================================================

# Project initialization (AI-agnostic)
function project-init {
    <#
    .SYNOPSIS
    Initialize a new project with AI Memory (Claude, Gemini & Aider)
    .DESCRIPTION
    Creates project structure and copies AI configurations from global config.
    .PARAMETER ProjectName
    Name of the project directory to create. Use "." for current directory.
    .PARAMETER Stack
    Technology stack: python, go, node, ts
    .EXAMPLE
    project-init my-project python
    .EXAMPLE
    project-init . node
    #>
    param(
        [Parameter(Position=0)]
        [string]$ProjectName = ".",
        [Parameter(Position=1)]
        [string]$Stack = "python"
    )

    $ScriptsDir = "$env:USERPROFILE\scripts"
    $InitScript = "$ScriptsDir\init-project.ps1"

    if (Test-Path $InitScript) {
        & $InitScript -ProjectName $ProjectName -Stack $Stack
    } else {
        Write-Host "[ERROR] init-project.ps1 not found at $InitScript" -ForegroundColor Red
        Write-Host "Run setup-windows.ps1 from your dotfiles repository first." -ForegroundColor Yellow
    }
}

# Quick navigation to projects (customize paths as needed)
function gprj { Set-Location "$env:USERPROFILE\Apps" }

# Git shortcuts
function gs { git status }
function gd { git diff }
function gl { git log --oneline -10 }
function gp { git pull }

# ============================================================================
# PATH AUGMENTATION (informational - actual PATH is set by setup script)
# ============================================================================
# The setup-windows.ps1 script adds ~/scripts to PATH at User level.
# This ensures init-project.ps1 and other scripts are available globally.

# ============================================================================
# PROMPT CUSTOMIZATION (optional)
# ============================================================================

function prompt {
    $currentPath = Get-Location
    $shortPath = $currentPath.Path.Replace($env:USERPROFILE, "~")

    # Git branch detection
    $gitBranch = ""
    if (Test-Path .git) {
        $branch = git branch --show-current 2>$null
        if ($branch) {
            $gitBranch = " ($branch)"
        }
    }

    Write-Host "$shortPath" -NoNewline -ForegroundColor Cyan
    Write-Host $gitBranch -NoNewline -ForegroundColor Yellow
    return "> "
}

# ============================================================================
# ENVIRONMENT VARIABLES
# ============================================================================

# Uncomment and set if needed:
# $env:GEMINI_API_KEY = "your-api-key"
# $env:ANTHROPIC_API_KEY = "your-api-key"

# ============================================================================
# STARTUP MESSAGE
# ============================================================================

Write-Host "Dotfiles profile loaded. Type 'project-init -?' for help." -ForegroundColor DarkGray
