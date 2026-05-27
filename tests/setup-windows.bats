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

@test "setup-windows.ps1 deploys Antigravity (agy) configuration (SDD-007)" {
    grep -q 'Deploying Antigravity (agy) configuration' "$PS1_SCRIPT"
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

@test "setup-windows.ps1 MCP loop is StrictMode-safe (probes PSObject.Properties)" {
    # Under Set-StrictMode -Version Latest, $srv.prerequisite_binary throws when
    # the field is absent. Only the hive MCP declares this property; the other
    # servers (sequential-thinking, context7) don't. The loop must probe via
    # PSObject.Properties to treat missing fields as $null, matching Linux which
    # uses jq's `(.prerequisite_binary // "")` null-coalesce.
    grep -qF "PSObject.Properties['prerequisite_binary']" "$PS1_SCRIPT"
    grep -qF "PSObject.Properties['prerequisite_command']" "$PS1_SCRIPT"
}

@test "setup-windows.ps1 eager-sources load-secrets.ps1 after sensitive deploy" {
    # Every block AFTER sensitive/ copy must see $env:NAN_API_KEY / VAULT_PATH /
    # TS_AUTHKEY etc. populated. Cross-OS parity with setup-linux.sh's same
    # eager source. Wrapped in try/catch so setup never aborts on secrets issues.
    grep -qF 'Eager-load secrets' "$PS1_SCRIPT"
    grep -qE '\. \$loadSecretsDeployed' "$PS1_SCRIPT"
    grep -qF 'try { . $loadSecretsDeployed' "$PS1_SCRIPT"
}

@test "parity: both setups eager-load secrets after sensitive deploy" {
    grep -qF 'Eager-load secrets' "$PS1_SCRIPT"
    grep -qF 'Eager-load secrets' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-windows.ps1 preflights the age identity key (warn, not fail)" {
    # Without ~/.config/age/key.txt, load-secrets.ps1 silently no-ops at
    # shell startup and opencode/agy 401 with no clue. Preflight WARNs (must
    # not abort -- encrypted files still deploy so a key imported later works).
    grep -qF 'age identity key not found' "$PS1_SCRIPT"
    grep -qF 'AGE_KEY_PATH' "$PS1_SCRIPT"
    grep -qF 'docs/SECRETS.md' "$PS1_SCRIPT"
    # Must use Write-Warn (non-fatal), not Write-Err.
    grep -B2 'age identity key not found' "$PS1_SCRIPT" | grep -q 'Test-Path'
}

@test "parity: age key preflight present in both setup-windows.ps1 and setup-linux.sh" {
    grep -qF 'age identity key not found' "$PS1_SCRIPT"
    grep -qF 'age identity key not found' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-windows.ps1 removes legacy GEMINI.md (SDD-007 migration)" {
    # Pre-SDD-007 setups deployed ~/.gemini/GEMINI.md. After agy replaced
    # gemini-cli, AGY.md is the new identity file. Without explicit cleanup the
    # orphan GEMINI.md lingers and confuses any tool that picks up either file.
    grep -qF 'Removed legacy GEMINI.md' "$PS1_SCRIPT"
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

# BUG-012: claude-mem-heal.ps1 creates a Junction `marketplaces\thedotmack`
# pointing at `marketplaces\thedotmack-claude-mem` so the plugin's hardcoded
# `marketplaces/thedotmack/plugin/scripts/...` fallback paths resolve when
# CLAUDE_PLUGIN_ROOT is unset/stale (the UserPromptSubmit hook failure mode).
@test "claude-mem-heal.ps1 creates legacy marketplace junction (BUG-012)" {
    local heal="$DOTFILES_DIR/scripts/claude-mem-heal.ps1"
    grep -qF 'thedotmack-claude-mem' "$heal"
    grep -qF -- '-ItemType Junction' "$heal"
}

# Junction creation must be guarded: only create when target dir missing AND
# source dir present. Mirrors the heal script's idempotence pattern.
@test "claude-mem-heal.ps1 junction creation is idempotent-guarded (BUG-012)" {
    local heal="$DOTFILES_DIR/scripts/claude-mem-heal.ps1"
    grep -B5 -A5 -- '-ItemType Junction' "$heal" | grep -qE 'Test-Path|-not'
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

# --- BUG-004: defense-in-depth around claude plugin install (truncate guard) ---
# Every `claude plugin install` call triggers upstream anthropics/claude-code#59870:
# the CLI's deserialize-modify-serialize cycle drops fields outside its internal
# struct (organizationType, organizationRateLimitTier, projects map, onboarding
# flags), shrinking ~/.claude/.claude.json from ~75 KB to ~1.5 KB and forcing
# re-authentication. The existing idempotence guard (`-match [regex]::Escape`
# against `claude plugin list` output) yields a false negative for
# claude-mem@thedotmack because it does not appear in that listing -- so every
# run installs it again, triggering the truncation. The fix is defense in depth:
# snapshot .claude.json before the call, restore if it shrinks >50% from a
# baseline of >=10 KB. Complementary to SDD-021's session-start canary at
# claude-session-start.ps1 (same 10240 threshold, same upstream issue).

@test "setup-windows.ps1 defines Backup-AndRestoreClaudeJson helper (BUG-004)" {
    grep -q 'function Backup-AndRestoreClaudeJson' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 cites upstream issue 59870 in the truncate guard (BUG-004)" {
    grep -qF '#59870' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 uses 10240-byte sanity floor in the truncate guard (BUG-004)" {
    grep -qF '10240' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 wraps claude plugin install with Backup-AndRestoreClaudeJson (BUG-004)" {
    # Structural check: the helper invocation appears within 5 lines preceding
    # the `claude plugin install` call in the foreach loop.
    grep -B5 'claude plugin install' "$PS1_SCRIPT" | grep -q 'Backup-AndRestoreClaudeJson'
}

@test "setup-windows.ps1 still preserves the upstream idempotence guard (BUG-004)" {
    # Defense in depth -- the wrapper does NOT replace the existing guard.
    grep -qF 'installedPlugins -match' "$PS1_SCRIPT"
    grep -qF 'claude plugin list' "$PS1_SCRIPT"
}

# --- BUG-011: extend the BUG-004 guard to every claude CLI call site ---
# BUG-004 covered only `claude plugin install`. The same upstream truncation
# (anthropics/claude-code#59870) fires on `claude mcp get`, `claude mcp add`, and
# `claude plugin list` because they go through the same deserialize-modify-
# serialize cycle. With ~9 MCP servers, each unwrapped setup run triggered
# ~18 chances of truncation. These asserts lock in the wrap on every call site.

@test "setup-windows.ps1 wraps claude mcp add with Backup-AndRestoreClaudeJson (BUG-011)" {
    grep -B15 'claude mcp add --transport' "$PS1_SCRIPT" | grep -q 'Backup-AndRestoreClaudeJson'
}

@test "setup-windows.ps1 wraps claude mcp get with Backup-AndRestoreClaudeJson (BUG-011)" {
    # Same scriptblock typically wraps both mcp get and mcp add inside one iteration.
    grep -B15 'claude mcp get' "$PS1_SCRIPT" | grep -q 'Backup-AndRestoreClaudeJson'
}

@test "setup-windows.ps1 wraps claude plugin list with Backup-AndRestoreClaudeJson (BUG-011)" {
    grep -B5 'claude plugin list' "$PS1_SCRIPT" | grep -q 'Backup-AndRestoreClaudeJson'
}

# --- BUG-014: register thedotmack marketplace before plugin install ---
# Windows side of the BUG-014 cross-OS fix. setup-windows.ps1 lists
# claude-mem@thedotmack in the plugin install loop but never registers the
# thedotmack marketplace first. The install fails silently (try/catch).
# Fix: invoke `claude plugin marketplace add thedotmack/claude-mem` BEFORE the
# install loop, wrapped with Backup-AndRestoreClaudeJson.

@test "setup-windows.ps1 registers thedotmack marketplace before plugin install (BUG-014)" {
    add_line=$(grep -n 'claude plugin marketplace add thedotmack/claude-mem' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    install_line=$(grep -n 'claude plugin install \$plugin' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    [ -n "$add_line" ] && [ -n "$install_line" ]
    [ "$add_line" -lt "$install_line" ]
}

@test "setup-windows.ps1 wraps claude plugin marketplace add with Backup-AndRestoreClaudeJson (BUG-014)" {
    grep -B5 'claude plugin marketplace add thedotmack' "$PS1_SCRIPT" | grep -q 'Backup-AndRestoreClaudeJson'
}

# --- BUG-013: install Obsidian CLI on Windows ---
# `@vorillaz/obsidian-cli` provides the `obsidian` binary used by
# obs-cli.ps1 and the vault-health workflow. setup-windows.ps1 must install
# it idempotently via npm global (user scope, no admin), gated on npm
# availability so machines without Node.js gracefully skip.

@test "setup-windows.ps1 installs Obsidian CLI via npm (BUG-013)" {
    grep -qF "npm install -g '@vorillaz/obsidian-cli'" "$PS1_SCRIPT"
}

@test "setup-windows.ps1 Obsidian CLI install is gated on npm + idempotent (BUG-013)" {
    # idempotence: skip if `obsidian` already on PATH
    grep -B10 "npm install -g '@vorillaz/obsidian-cli'" "$PS1_SCRIPT" | grep -q "Get-Command obsidian"
    # gating: only run if npm is available
    grep -B10 "npm install -g '@vorillaz/obsidian-cli'" "$PS1_SCRIPT" | grep -q "Get-Command npm"
}

# --- AI-014: OpenCode Windows bootstrap (mirror of AI-011 Linux side) ---
# setup-windows.ps1 must install OpenCode via winget SST.opencode (cleaner
# than the Linux curl-bash equivalent), deploy ai/opencode/opencode.jsonc
# reconcile-not-skip, and sync ai/opencode/commands/*.md to the user's
# OpenCode commands directory.

@test "setup-windows.ps1 installs OpenCode via winget SST.opencode (AI-014)" {
    grep -qF 'SST.opencode' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys opencode.jsonc via Deploy-File helper (SDD-007)" {
    # Post-SDD-007: the inline Copy-Item + Get-FileHash block was extracted
    # into Deploy-File in scripts/utils.ps1 (atomic + idempotent by SHA256).
    grep -qF 'opencode.jsonc' "$PS1_SCRIPT"
    grep -qE 'Deploy-File.*opencodeConfigSrc' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 syncs OpenCode commands with orphan removal (AI-014)" {
    # The commands sync writes to .config\opencode\commands and removes any
    # .md file present in the destination that does not exist in the source.
    grep -qF '.config\opencode\commands' "$PS1_SCRIPT"
    grep -qF 'cmds_removed' "$PS1_SCRIPT" || grep -qF '$cmdsRemoved' "$PS1_SCRIPT"
}

# --- BUG-005: Windows PowerShell 5.1 auto-reexec under pwsh ---
# SDD-002 (PR #51) introduced Merge-ClaudeSettings which uses
# `ConvertFrom-Json -AsHashtable` -- a parameter added in PowerShell 7.0 that
# does NOT exist in Windows PowerShell 5.1. When the user invokes
# `PowerShell -ExecutionPolicy Bypass -File .\setup-windows.ps1` (Windows
# default `PowerShell` resolves to 5.1.x), the inner try/catch swallows the
# ParameterBindingException as if it were a JSON parse error and the per-key
# merge from ai/claude/settings.json is silently skipped. Preamble at the top
# of the script detects PSVersion.Major < 7 and re-execs under pwsh, or fails
# loud with winget install hint if pwsh is missing.

@test "setup-windows.ps1 detects PowerShell version major (BUG-005)" {
    grep -qF 'PSVersionTable.PSVersion.Major' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 re-execs under pwsh when PS < 7 (BUG-005)" {
    grep -qF 'Get-Command pwsh' "$PS1_SCRIPT"
    grep -qF '$PSCommandPath' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 fails loud with winget hint when pwsh missing (BUG-005)" {
    grep -qF 'winget install Microsoft.PowerShell' "$PS1_SCRIPT"
    grep -qE 'exit\s+1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 cites BUG-005 in the preamble inline comment" {
    grep -qF 'BUG-005' "$PS1_SCRIPT"
}

@test "negative parity: setup-linux.sh does not reference PSVersion (BUG-005 is Windows-only)" {
    ! grep -qF 'PSVersion' "$DOTFILES_DIR/setup-linux.sh"
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

@test "parity: both scripts verify AGY.md deploy with AGENTS.md pointer marker (SDD-007)" {
    # Post-SDD-007: legacy GEMINI.md compat write removed; AGY.md is the
    # canonical pointer file for the agy CLI. Both setup scripts deploy it
    # and assert the AGENTS.md pointer marker on every run.
    grep -qF 'First, read `AGENTS.md`' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF "AGY.md" "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'First, read `AGENTS.md`' "$PS1_SCRIPT"
    grep -qF "AGY.md" "$PS1_SCRIPT"
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

# --- REFACTOR-003: diff-check.ps1 deploy + dch alias + sec 12 wiring ---
# Windows port of diff-check.sh closes the permanent SKIP in healthcheck.ps1
# section 12/12. Mirrors PR #10 (Linux side, 2026-05-12).

@test "REFACTOR-003: setup-windows.ps1 deploys diff-check.ps1 to ScriptsDir" {
    grep -qF 'diff-check.ps1' "$PS1_SCRIPT"
    grep -B5 'Deployed diff-check.ps1' "$PS1_SCRIPT" | grep -q 'Copy-Item'
}

@test "REFACTOR-003: powershell/profile.ps1 declares dch function" {
    grep -qF 'function dch' "$DOTFILES_DIR/powershell/profile.ps1"
    grep -B2 -A6 'function dch' "$DOTFILES_DIR/powershell/profile.ps1" | grep -qF 'diff-check.ps1'
}

# --- WIN-001: healthcheck.ps1 wiring as final step ---

@test "WIN-001: setup-windows.ps1 invokes healthcheck.ps1 after doctor" {
    grep -qF 'healthcheck.ps1' "$PS1_SCRIPT"
    grep -qF 'Running post-setup healthcheck' "$PS1_SCRIPT"
}

@test "WIN-001: setup-windows.ps1 healthcheck block is non-fatal (Write-Warn, no exit)" {
    # Capture the healthcheck block (between 8d header and section 9 SUMMARY)
    # and assert it warns instead of exiting.
    sed -n '/8d\. POST-SETUP HEALTHCHECK/,/9\. SUMMARY/p' "$PS1_SCRIPT" | grep -qF 'Write-Warn'
    ! sed -n '/8d\. POST-SETUP HEALTHCHECK/,/9\. SUMMARY/p' "$PS1_SCRIPT" | grep -qE '^\s*exit\s+[0-9]'
}

@test "WIN-001: setup-windows.ps1 healthcheck block runs after doctor block" {
    doctor_line=$(grep -n '8c\. POST-SETUP DOCTOR' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    health_line=$(grep -n '8d\. POST-SETUP HEALTHCHECK' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    [[ -n "$doctor_line" ]] && [[ -n "$health_line" ]] && [[ "$health_line" -gt "$doctor_line" ]]
}

# --- BUG-021: profile-section fail-fast (root cause of BUG-020 accumulation) ---
# Section 4 "DEPLOY POWERSHELL PROFILE" used to swallow OOM/-replace failures and
# print "[SUCCESS] Updated dotfiles section in PowerShell profile" regardless. On
# a corrupted 26 MB profile this allowed silent partial writes to compound until
# the deployed file hit 689K lines and every shell startup hit a parser error.
# The contract these tests pin: the profile-section block must (a) fail-fast on
# I/O error, (b) preflight known corruption signals (size, marker count), and
# (c) point users at the recovery script (profile-heal.ps1 from BUG-020).

@test "BUG-021: profile-section block is wrapped in try/catch with exit 1 on failure" {
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE '\btry\s*\{'
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE '\bcatch\s*\{'
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE 'exit\s+1'
}

@test "BUG-021: profile-section I/O uses -ErrorAction Stop (terminating errors only)" {
    # Without -ErrorAction Stop, Get-Content/Set-Content emit non-terminating
    # errors that bypass try/catch. Stop promotes them so the catch fires.
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE 'Get-Content[^|]*-ErrorAction Stop'
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE 'Set-Content[^|]*-ErrorAction Stop'
}

@test "BUG-021: profile preflight aborts when existing profile size > 1 MB" {
    # Healthy profile is ~7 KB. 1 MB is ~140x — a reliable corruption signal.
    # Abort BEFORE Get-Content -Raw OOMs on the 26 MB nightmare scenario.
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE '\.Length\s+-gt\s+1MB'
}

@test "BUG-021: profile preflight aborts when marker count > 2 (BUG-020 corruption signal)" {
    # A healthy profile has exactly 1 start + 1 end marker pair. Anything > 2
    # of either marker means a previous setup run left orphan blocks. Refuse to
    # touch it — defer to profile-heal.ps1 (BUG-020).
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE 'Select-String|-AllMatches|Matches\('
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE 'markerCount|MarkerCount'
}

@test "BUG-021: preflight error message points at profile-heal.ps1 (BUG-020 recovery handoff)" {
    # When preflight aborts, the user needs a one-line escape hatch. The
    # recovery script (BUG-020) is named profile-heal.ps1 in scripts/.
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qF 'profile-heal.ps1'
}

@test "parity: both setups exit non-zero on profile-write failure" {
    # Linux: setup-linux.sh:11 already has 'set -euo pipefail' which propagates
    # deploy_file's non-zero exits. Asserts the safety net is still in place.
    grep -qE '^set -euo pipefail' "$DOTFILES_DIR/setup-linux.sh"
    # Windows: try/catch + exit 1 in the profile-section block (covered structurally
    # by BUG-021 test #1; this parity test asserts both legs of the contract exist).
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE '\bcatch\s*\{'
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
