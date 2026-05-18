#!/usr/bin/env bats
# Tests for setup-windows.ps1 (structural + PSScriptAnalyzer)

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export PS1_SCRIPT="$DOTFILES_DIR/setup-windows.ps1"
}

@test "setup-windows.ps1 exists" {
    [[ -f "$PS1_SCRIPT" ]]
}

# --- Structural checks ---

@test "setup-windows.ps1 has .SYNOPSIS block" {
    grep -q '\.SYNOPSIS' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 has .DESCRIPTION block" {
    grep -q '\.DESCRIPTION' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 has CmdletBinding" {
    grep -q 'CmdletBinding' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys Claude configuration" {
    grep -q 'Deploying Claude configuration' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys Gemini configuration" {
    grep -q 'Deploying Gemini configuration' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys PowerShell profile" {
    grep -q 'Setting up PowerShell profile' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys Git configuration" {
    grep -q 'Setting up Git configuration' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys versions.conf" {
    grep -q 'Deploying versions.conf' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 no longer deploys Aider configuration (sunset)" {
    ! grep -q 'Deploying Aider configuration' "$PS1_SCRIPT"
    ! grep -q 'Installing aider-chat via uv' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 registers MCP servers" {
    grep -q 'Registering Claude Code MCP servers' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 sets up GitHub Copilot CLI" {
    grep -q 'Setting up GitHub Copilot CLI' "$PS1_SCRIPT"
}

# --- BUG-003: drop gh-copilot extension path, detect new copilot CLI v2 ---
# The legacy 'gh extension install github/gh-copilot' product was the v1 wrapper
# around suggest/explain. The new 'copilot' standalone CLI (winget GitHub.Copilot,
# binary `copilot`) is agentic (closer to Claude Code). Setup detects the new one
# via 'Get-Command copilot' and stops referencing the old extension entirely.

@test "setup-windows.ps1 detects copilot via Get-Command, not gh extension list" {
    grep -qF "Get-Command copilot" "$PS1_SCRIPT"
    ! grep -qF "gh extension list" "$PS1_SCRIPT"
    ! grep -qF "github/gh-copilot" "$PS1_SCRIPT"
}

@test "setup-windows.ps1 auto-installs GitHub.Copilot via winget" {
    grep -qF "GitHub.Copilot" "$PS1_SCRIPT"
    # In the dev tools winget block, the binary check uses `copilot`
    grep -qE 'Name = "GitHub Copilot CLI"; Cmd = "copilot"; Id = "GitHub\.Copilot"' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys init-project.ps1" {
    grep -q 'init-project.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys knowledge-crystallize.ps1" {
    grep -q 'knowledge-crystallize.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys claude-session-start.ps1" {
    grep -q 'claude-session-start.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys claude-mem-heal.ps1" {
    grep -q 'claude-mem-heal.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys dotfiles-sync.ps1" {
    grep -q 'dotfiles-sync.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys obs-cli.ps1" {
    grep -q 'obs-cli.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys load-secrets.ps1" {
    grep -q 'load-secrets.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 sets up secrets system" {
    grep -q 'Setting up secrets system' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 copies env-mapping.conf" {
    grep -q 'env-mapping.conf' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 registers SessionStart hook" {
    grep -q 'SessionStart' "$PS1_SCRIPT"
}

# SessionStart hook command must point at the deploy directory ($ScriptsDir),
# not the dotfiles staging area. Regression guard for issue #20.
@test "setup-windows.ps1 SessionStart hook command uses ScriptsDir, not DotfilesDest" {
    grep -Eq '\$sessionStartCmd\s*=\s*"\$ScriptsDir' "$PS1_SCRIPT"
    ! grep -Eq '\$sessionStartCmd\s*=\s*"\$DotfilesDest' "$PS1_SCRIPT"
}

# Hook registration must self-heal -- never trust "an entry exists" to mean
# "the entry is correct". Post-SDD-002 (PR #51): Merge-ClaudeSettings ALWAYS
# rewrites hooks.SessionStart from the template (with __HOOK_COMMAND__
# substituted), which is a stronger guarantee than the previous compare-then-
# rewrite -- self-heal runs unconditionally on every setup invocation.
@test "setup-windows.ps1 SessionStart hook self-heals on path drift" {
    grep -qF '$expectedHookCommand' "$PS1_SCRIPT"
    grep -qF 'Merge-ClaudeSettings -TemplatePath' "$PS1_SCRIPT"
}

# --- Scheduled task self-heal (same class as #20) ---

# DotfilesVaultMaintenance task must reconcile arguments, not skip when present;
# otherwise a stale task pointing at a moved/renamed script stays broken forever.
@test "setup-windows.ps1 vault maintenance task self-heals on argument drift" {
    grep -q '\$expectedTaskArgument' "$PS1_SCRIPT"
    grep -Eq '\$existingTaskArgument\s+-eq\s+\$expectedTaskArgument' "$PS1_SCRIPT"
}

# --- MCP server registration (SSOT + idempotence) ---

# MCP server list must live in mcp-servers.json, not hardcoded in the setup script.
@test "setup-windows.ps1 MCP registration reads from mcp-servers.json" {
    grep -q 'mcp-servers\.json' "$PS1_SCRIPT"
    ! grep -Eq 'claude mcp add --transport (stdio|http) (drawio|socket|context7|sequential-thinking|hive)' "$PS1_SCRIPT"
}

# MCP registration must check existence before adding, not blindly retry.
@test "setup-windows.ps1 MCP registration checks existence with claude mcp get" {
    grep -q 'claude mcp get' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys SSH config" {
    grep -q 'Setting up SSH config' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 configures PATH" {
    grep -q 'Configuring PATH' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 has summary section" {
    grep -q 'Setup Complete' "$PS1_SCRIPT"
}

# --- Developer tools ---

@test "setup-windows.ps1 has developer tools section" {
    grep -q 'DEVELOPER TOOLS' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 checks for winget" {
    grep -q 'Get-Command winget' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs age via winget" {
    grep -q 'FiloSottile.age' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs eza via winget" {
    grep -q 'eza-community.eza' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs jq via winget" {
    grep -q 'jqlang.jq' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs GitHub CLI via winget" {
    grep -q 'GitHub.cli' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs zoxide via winget" {
    grep -q 'ajeetdsouza.zoxide' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs uv if missing" {
    grep -q 'Installing uv' "$PS1_SCRIPT"
    grep -q 'astral.sh/uv/install.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs poetry via uv" {
    grep -q 'Installing poetry' "$PS1_SCRIPT"
    grep -q 'uv tool install poetry' "$PS1_SCRIPT"
}

# --- Cross-platform parity: developer tools ---

@test "parity: both scripts install uv" {
    grep -q 'Installing uv' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'Installing uv' "$PS1_SCRIPT"
}

@test "parity: both scripts install poetry via uv" {
    grep -q 'uv tool install poetry' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'uv tool install poetry' "$PS1_SCRIPT"
}

@test "parity: both scripts install age" {
    grep -q 'command -v age' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'FiloSottile.age' "$PS1_SCRIPT"
}

@test "parity: both scripts install eza" {
    grep -q 'command -v eza' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'eza-community.eza' "$PS1_SCRIPT"
}

@test "parity: both scripts install jq" {
    grep -q 'command -v jq' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'jqlang.jq' "$PS1_SCRIPT"
}

@test "parity: both scripts install gh" {
    grep -q 'command -v gh' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'GitHub.cli' "$PS1_SCRIPT"
}

@test "parity: both scripts install zoxide" {
    grep -q 'command -v zoxide' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'ajeetdsouza.zoxide' "$PS1_SCRIPT"
}

# --- Cross-platform parity: AI config deploy verification ---
# AI-013 refactored CLAUDE.md/GEMINI.md/copilot-instructions.md to pointer-style
# files starting with 'First, read `AGENTS.md`'. The old marker 'CORE PRINCIPLE'
# no longer exists in any deployed file. setup-linux.sh was updated; this guard
# keeps setup-windows.ps1 in lockstep (regression class: BUG-001 + BUG-002).

@test "parity: both scripts verify CLAUDE.md deploy with AGENTS.md pointer marker" {
    grep -qF "grep -q 'First, read \`AGENTS.md\`' \"\$HOME/.claude/CLAUDE.md\"" "$DOTFILES_DIR/setup-linux.sh"
    grep -qF "Select-String -Path \"\$ClaudeHome\\CLAUDE.md\" -Pattern 'First, read \`AGENTS.md\`'" "$PS1_SCRIPT"
}

@test "parity: both scripts verify GEMINI.md deploy with AGENTS.md pointer marker" {
    grep -qF "grep -q 'First, read \`AGENTS.md\`' \"\$HOME/.gemini/GEMINI.md\"" "$DOTFILES_DIR/setup-linux.sh"
    grep -qF "Select-String -Path \"\$GeminiHome\\GEMINI.md\" -Pattern 'First, read \`AGENTS.md\`'" "$PS1_SCRIPT"
}

@test "setup-windows.ps1 has no stale 'CORE PRINCIPLE' verify-pattern references" {
    # AGENTS.md itself may mention 'CORE PRINCIPLE' as documentation text -- that
    # is fine. The bug is using it as a Select-String -Pattern, which post-AI-013
    # never matches the deployed file and emits a spurious [ERROR] every run.
    ! grep -qF -- '-Pattern "CORE PRINCIPLE"' "$PS1_SCRIPT"
}

@test "parity: both setup scripts detect copilot via binary, not gh extension" {
    grep -qF "command -v copilot" "$DOTFILES_DIR/setup-linux.sh"
    grep -qF "Get-Command copilot" "$PS1_SCRIPT"
    # Neither should still reference the legacy extension path
    ! grep -qF "github/gh-copilot" "$DOTFILES_DIR/setup-linux.sh"
    ! grep -qF "github/gh-copilot" "$PS1_SCRIPT"
}

# --- SDD-002: settings.json template merge ---
# Both setup scripts read ai/claude/settings.json as SSOT for the dotfiles-owned
# subset of ~/.claude/settings.json. Per-key merge policy documented in
# specs/SDD-002-settings-portability/proposal.md.

@test "SDD-002: setup-windows.ps1 defines Merge-ClaudeSettings function" {
    grep -qE '^function Merge-ClaudeSettings' "$PS1_SCRIPT"
}

@test "SDD-002: setup-windows.ps1 calls Merge-ClaudeSettings (not inline hashtable)" {
    grep -qF 'Merge-ClaudeSettings -TemplatePath' "$PS1_SCRIPT"
    # The legacy inline hook hashtable must be gone
    ! grep -qE '\$hookEntry\s*=\s*@\{' "$PS1_SCRIPT"
}

@test "SDD-002: setup-windows.ps1 references the template path ai\\claude\\settings.json" {
    grep -qF 'ai\claude\settings.json' "$PS1_SCRIPT"
}

@test "SDD-002: setup-windows.ps1 bulk copy of ai/claude/* excludes settings.json" {
    # -- separator before pattern starting with dash so grep does not parse it
    # as a flag (same fix pattern as the BUG-002 CORE PRINCIPLE assert).
    grep -qF -- "-Exclude 'settings.json'" "$PS1_SCRIPT"
}

@test "SDD-002: setup-linux.sh defines merge_claude_settings function" {
    grep -qE '^merge_claude_settings\(\)' "$DOTFILES_DIR/setup-linux.sh"
}

@test "SDD-002: setup-linux.sh calls merge_claude_settings (not inline HOOK_ENTRY)" {
    grep -qF 'merge_claude_settings "$CLAUDE_SETTINGS_TEMPLATE"' "$DOTFILES_DIR/setup-linux.sh"
    # The legacy inline HOOK_ENTRY jq -n heredoc must be gone
    ! grep -qF 'HOOK_ENTRY=$(jq -n' "$DOTFILES_DIR/setup-linux.sh"
}

@test "SDD-002: setup-linux.sh references the template path ai/claude/settings.json" {
    grep -qF 'ai/claude/settings.json' "$DOTFILES_DIR/setup-linux.sh"
}

@test "SDD-002: setup-linux.sh bulk copy of ai/claude/* skips settings.json" {
    grep -qF "= \"settings.json\" ] && continue" "$DOTFILES_DIR/setup-linux.sh"
}

@test "SDD-002: parity -- both scripts log the bootstrap message" {
    grep -qF "Bootstrapping ~/.claude/settings.json from template" "$PS1_SCRIPT"
    grep -qF "Bootstrapping ~/.claude/settings.json from template" "$DOTFILES_DIR/setup-linux.sh"
}

@test "SDD-002: parity -- both scripts substitute __HOOK_COMMAND__ placeholder" {
    grep -qF '__HOOK_COMMAND__' "$PS1_SCRIPT"
    grep -qF 'SessionStart[0].hooks[0].command) = $cmd' "$DOTFILES_DIR/setup-linux.sh"
}

# --- PSScriptAnalyzer ---

@test "setup-windows.ps1 passes PSScriptAnalyzer (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    run pwsh -NonInteractive -Command "
        \$ErrorActionPreference = 'Stop'
        try {
            Install-Module PSScriptAnalyzer -Force -Scope CurrentUser -ErrorAction SilentlyContinue
            \$results = Invoke-ScriptAnalyzer -Path '$PS1_SCRIPT' -Settings '$DOTFILES_DIR/.PSScriptAnalyzerSettings.psd1' -Severity Error,Warning
            if (\$results) {
                \$results | Format-Table -AutoSize
                exit 1
            }
            Write-Host 'PSScriptAnalyzer: OK'
        } catch {
            Write-Warning \"PSScriptAnalyzer not available: \$_\"
            exit 0
        }
    "
    [[ "$status" -eq 0 ]]
}

@test "setup-windows.ps1 valid PowerShell syntax (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    run pwsh -NonInteractive -Command "
        \$errors = \$null
        [System.Management.Automation.Language.Parser]::ParseFile(
            '$PS1_SCRIPT', [ref]\$null, [ref]\$errors
        ) | Out-Null
        if (\$errors) {
            \$errors | ForEach-Object { Write-Error \$_.Message }
            exit 1
        }
        Write-Host 'Syntax OK'
    "
    [[ "$status" -eq 0 ]]
}
