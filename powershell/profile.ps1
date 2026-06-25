# PowerShell Profile for Dotfiles
# ================================
# This profile is deployed by setup-windows.ps1
# Source: dotfiles/powershell/profile.ps1

# ============================================================================
# ALIASES
# ============================================================================

# AI Tools
Set-Alias -Name c -Value claude
Set-Alias -Name g -Value agy

# AI provider endpoints -- NaN community (primary, OpenAI-compatible).
# API key in $env:NAN_API_KEY -- injected on demand via `dotf secrets run` (wrappers below), not the ambient session.
$env:NAN_BASE_URL = 'https://api.nan.builders/v1'

# OpenCode (primary AI coding agent -- install is admin-only on Windows, so
# the alias is conditional. On machines without opencode installed, this
# block is a no-op and `oc` reports "command not found" as expected.)
if (Get-Command opencode -ErrorAction SilentlyContinue) {
    # oc: opencode TUI dispatch. Plain (no --pure flag) -- Windows PowerShell
    # does NOT exhibit the Linux tool-resolution hang. Empirical 2026-05-26:
    # opencode launched bare from a Windows terminal works correctly with
    # MCPs+plugins enabled. The Linux side keeps `opencode --pure` pending
    # the abandoned DX-003 terminal-hang investigation -- documented
    # cross-OS asymmetry, not parity drift.
    Set-Alias -Name oc -Value opencode

    # qq / qf: one-shot quick-question wrappers. Mirrors .zsh/aliases.zsh and
    # .bashrc. One-shot: each call is a fresh session.
    #   qq -> nan/qwen3.6           (default daily, multilingual, 262K ctx)
    #   qf -> nan/deepseek-v4-flash (long-context 500K)
    # Note: '??' is the null-coalescing operator in PowerShell 7+, so the
    # name 'qq' is the cross-platform compromise (works in bash, zsh, pwsh).
    function qq {
        if ($args.Count -eq 0) {
            Write-Error 'usage: qq <consulta libre>'
            return
        }
        opencode run -m nan/qwen3.6 ($args -join ' ')
    }
    function qf {
        if ($args.Count -eq 0) {
            Write-Error 'usage: qf <consulta libre>'
            return
        }
        opencode run -m nan/deepseek-v4-flash ($args -join ' ')
    }

    # dbg: deepseek con reasoning chain VISIBLE (opencode TUI oculta reasoning_content).
    # Windows uses bash via WSL or git-bash for the underlying nan-debug.sh.
    function dbg {
        if ($args.Count -eq 0) {
            Write-Error 'usage: dbg <consulta libre>'
            return
        }
        # Resolve the repo via the ADR-025 var (rendered into paths.ps1 by
        # `dotf env generate`, overridable in machine.json), not a hardcoded
        # ~/Projects/dotfiles literal that breaks on a relocated clone.
        $script = Join-Path $env:DOTFILES_REPO_DIR 'scripts\nan-debug.sh'
        if (Test-Path -LiteralPath $script) {
            bash $script ($args -join ' ')
        } else {
            Write-Error "nan-debug.sh not found at $script (check DOTFILES_REPO_DIR / run dotf env generate)"
        }
    }
}

# CLI-019: `dch` now wraps `dotf doctor`. The repo/deploy drift check moved
# into the Go sweep (PR-A, #513) and the standalone drift-check twins were
# retired; mirrors the Linux `dch` alias. Thin pass-through so flags reach dotf.
function dch {
    if (Get-Command dotf -ErrorAction SilentlyContinue) {
        dotf doctor @args
    } else {
        Write-Error "dotf not on PATH -- run setup-windows.ps1"
    }
}

# ============================================================================
# FUNCTIONS
# ============================================================================

# Project initialization (AI-agnostic)
function project-init {
    <#
    .SYNOPSIS
    Initialize a new project with AI Memory (Claude, Antigravity, OpenCode)
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

    if (Get-Command dotf -ErrorAction SilentlyContinue) {
        dotf init $ProjectName --stack $Stack
    } else {
        Write-Host "[ERROR] dotf not found on PATH." -ForegroundColor Red
        Write-Host "Run setup-windows.ps1 from your dotfiles repository first." -ForegroundColor Yellow
    }
}

# Quick navigation to projects
function gprj { Set-Location "$env:USERPROFILE\Projects" }
function gcs { Set-Location "$env:USERPROFILE\Projects\cheat-sheets" }
function gbp { Set-Location "$env:USERPROFILE\Projects\boilerplates" }

# Git shortcuts
function gs { git status }
function gd { git diff }
function gl { git log --oneline -10 }
function gp { git pull }

# GitHub Copilot CLI v2 aliases (BUG-003: standalone agentic `copilot`, not gh extension)
# - cop  -> interactive agent (default safe: tool use needs confirmation)
# - cops -> single-shot non-interactive prompt; --allow-all-tools is REQUIRED by
#          the CLI for non-interactive mode and gives the agent edit/exec power.
#          Use 'cop' instead when you want per-tool confirmation.
if (Get-Command copilot -ErrorAction SilentlyContinue) {
    Set-Alias -Name cop -Value copilot
    function cops { copilot -p "$args" --allow-all-tools -s }
}

# Healthcheck (WIN-001): mirrors the Linux `hc` alias. Runs the full 13-section
# structural verification of the deployed dotfiles install.
#
# CLI-018: `hc` runs `dotf doctor`, the single diagnostic that replaced the
# retired healthcheck + doctor shell scripts.
function hc {
    if (Get-Command dotf -ErrorAction SilentlyContinue) {
        & dotf doctor @args
    } else {
        Write-Host "[ERROR] dotf not found on PATH." -ForegroundColor Red
        Write-Host "Run setup-windows.ps1 from your dotfiles repository first." -ForegroundColor Yellow
    }
}

# Enhanced listing (requires eza)
if (Get-Command eza -ErrorAction SilentlyContinue) {
    function ll { eza -la --icons @args }
    function lla { eza -la --icons --git @args }
}

# ============================================================================
# PATH AUGMENTATION (informational - actual PATH is set by setup script)
# ============================================================================
# The setup-windows.ps1 script adds ~/scripts to PATH at User level.
# This ensures user scripts (hooks, helpers) are available globally.

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

# Structural paths (ADR-025). Source of truth: env-contract.json (defaults) +
# ~/.config/dotfiles/machine.json (per-machine overrides), rendered into
# paths.ps1 by `dotf env generate`. Edit those, not this block. DOTFILES_DIR is
# bootstrapped first because it locates the generated file itself.
if (-not $env:DOTFILES_DIR) { $env:DOTFILES_DIR = "$env:USERPROFILE\.dotfiles" }
$DotfPathsFile = Join-Path $env:DOTFILES_DIR 'paths.ps1'
# Zero-touch: auto-render paths.ps1 on first run when it is missing and dotf is on
# PATH (a fresh machine self-configures without a manual `dotf env generate`).
# Best-effort + silent; the bootstrap fallback below covers any failure.
if (-not (Test-Path $DotfPathsFile) -and (Get-Command dotf -ErrorAction SilentlyContinue)) {
    & dotf env generate 2>$null | Out-Null
}
if (Test-Path $DotfPathsFile) {
    . $DotfPathsFile
} else {
    # Bootstrap fallback: paths.ps1 not generated yet (fresh machine before the
    # first `dotf env generate`). Contract defaults inline, conditional so a
    # later source wins. DOTFILES_REPO_DIR keeps the legacy default; a relocated
    # repo is picked up once generate renders machine.json (BUG-020 / REFACTOR-003).
    if (-not $env:DOTFILES_REPO_DIR) { $env:DOTFILES_REPO_DIR = "$env:USERPROFILE\Projects\dotfiles" }
    if (-not $env:CLAUDE_CONFIG_DIR) { $env:CLAUDE_CONFIG_DIR = "$env:USERPROFILE\.claude" }
    if (-not $env:SCRIPTS_DIR) { $env:SCRIPTS_DIR = "$env:DOTFILES_DIR\scripts" }
    if (-not $env:AGY_HOME) { $env:AGY_HOME = "$env:USERPROFILE\.gemini\antigravity-cli" }
    if (-not $env:COPILOT_HOME) { $env:COPILOT_HOME = "$env:USERPROFILE\.copilot" }
    if (-not $env:OPENCODE_HOME) { $env:OPENCODE_HOME = "$env:USERPROFILE\.config\opencode" }
}

# AI provider endpoint -- NaN community (primary, OpenAI-compatible).
# Mirrors NAN_BASE_URL set in .bashrc/.zshrc. The API key is injected on demand
# via `dotf secrets run` (see the wrappers below), not the ambient session.
$env:NAN_BASE_URL = 'https://api.nan.builders/v1'

# Secrets are NOT auto-loaded into the ambient session (ADR-028 "not always
# exposed"). On demand: `dotf secrets run -- <cmd>` injects the decrypted secrets
# into that child process only. The AI CLIs that read their keys from the
# environment are wrapped to launch through it, so the secret lives only in their
# process, never this session. Recursion-safe: dotf resolves the real binary on
# PATH, not this function.
if (Get-Command dotf -ErrorAction SilentlyContinue) {
    function opencode { dotf secrets run -- opencode @args }
    function pi { dotf secrets run -- pi @args }
    function agy { dotf secrets run -- agy @args }
}

# ============================================================================
# STARTUP MESSAGE
# ============================================================================

Write-Host "Dotfiles profile loaded. Type 'project-init -?' for help." -ForegroundColor DarkGray
