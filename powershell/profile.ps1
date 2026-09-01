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

# agyp: launch agy with a saved prompt, the PowerShell twin of .zsh/functions.sh's
# agyp (PARITY-001, #764). Same contract on both sides: the prompt lives at
# $GEMINI_DIR\prompts\<name>.md, extra words are appended after a blank line,
# and a missing name or file is a non-terminating error (so $? is false) rather
# than a launch with an empty prompt.
#   agyp <prompt-name> [extra words]
function agyp {
    param(
        [Parameter(Position = 0)][string]$Name,
        [Parameter(ValueFromRemainingArguments = $true)][string[]]$Rest
    )
    if (-not $Name) {
        Write-Error 'usage: agyp <prompt-name> [args]'
        return
    }
    $geminiDir = if ($env:GEMINI_DIR) { $env:GEMINI_DIR } else { Join-Path $env:USERPROFILE '.gemini' }
    $promptFile = Join-Path (Join-Path $geminiDir 'prompts') "$Name.md"
    if (-not (Test-Path -LiteralPath $promptFile)) {
        Write-Error "agyp: prompt not found at $promptFile"
        return
    }
    $prompt = Get-Content -LiteralPath $promptFile -Raw
    & agy -i ($prompt + "`n`n" + ($Rest -join ' '))
}

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

# PowerShell resolves commands Alias -> Function -> Cmdlet -> Application, so a
# built-in alias silently makes a same-named function unreachable - no parse
# error, no warning, the function is just dead (BUG-034). The four names below
# carry over from .zsh/aliases.zsh, where they already mean exactly this;
# cross-OS parity is the point of this profile, so clear the built-ins rather
# than rename out of the collision. Only the *alias* is given up - each cmdlet
# stays reachable under its full name:
#   gp  -> Get-ItemProperty     gl  -> Get-Location
#   gcs -> Get-PSCallStack      gbp -> Get-PSBreakpoint
# `Remove-Item Alias:` rather than `Remove-Alias` so this also works on Windows
# PowerShell 5.1; -Force is required because the built-ins ship ReadOnly.
# Guarded by tests/powershell-profile-alias.Tests.ps1, which fails CI if any
# function defined here is left shadowed.
foreach ($shadowingAlias in 'gp', 'gl', 'gcs', 'gbp') {
    Remove-Item -Path "Alias:$shadowingAlias" -Force -ErrorAction SilentlyContinue
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
# The setup-windows.ps1 script adds ~\.dotfiles\scripts (the env contract's
# SCRIPTS_DIR, WIN-013) to PATH at User level, so the deployed scripts (hooks,
# helpers) are available globally; ~\scripts is the retired legacy location.

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
# CONSOLE ENCODING
# ============================================================================

# UTF-8 console I/O (WIN-009/#1290). PowerShell decodes captured native output
# with [Console]::OutputEncoding, which defaults to the system OEM code page, so
# `dotf` output carrying an em dash rendered as OEM glyphs and any function
# parsing it read corrupted bytes. Mirrors the block scripts/utils.ps1 applies
# to setup itself.
try {
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [Console]::OutputEncoding = $utf8NoBom
    [Console]::InputEncoding = $utf8NoBom
    $OutputEncoding = $utf8NoBom
} catch {
    Write-Verbose "console encoding not set: $_"
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
# into that child process only, so a key lives in the agent's process and never
# in this session. Recursion-safe: dotf resolves the real binary on PATH, not
# this function.
#
# Each wrapper is scoped with `--only`, and the scope is load-bearing. The
# unscoped form resolved the WHOLE registry on every launch: deliberate parity
# with the old ambient export, and free while every secret was age-backed, which
# is a local decrypt in milliseconds. After the Bitwarden migration (#961) each
# resolution is a `bw` shell-out at ~1.5s, so an unscoped launch paid ~45s of
# latency before the agent started, and died on the first locked entry even when
# the agent had no use for it (#976). Scoping is least privilege and startup
# time at once.
if (Get-Command dotf -ErrorAction SilentlyContinue) {
    function opencode { dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY,OPENAI_API_KEY -- opencode @args }
    function pi { dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY -- pi @args }
    # agy is deliberately NOT wrapped. It authenticates with its own stored
    # credentials and reads no variable this registry exposes, verified against
    # both its settings files and the strings of the binary itself. Wrapping it
    # injected nothing and cost a full registry resolution per launch.
}

# ============================================================================
# STARTUP MESSAGE
# ============================================================================

Write-Host "Dotfiles profile loaded. Type 'project-init -?' for help." -ForegroundColor DarkGray
