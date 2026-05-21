#!/usr/bin/env bats
# Tests for setup-linux.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "setup-linux.sh valid bash syntax" {
    bash -n "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh valid zsh syntax" {
    zsh -n "$DOTFILES_DIR/setup-linux.sh"
}

# --- Developer tools section ---

@test "setup-linux.sh creates ~/.local/bin directory" {
    grep -q 'ensure_directory.*\.local/bin' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh adds ~/.local/bin to PATH" {
    grep -q 'export PATH.*\.local/bin' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs uv if missing" {
    grep -q 'command -v uv' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'astral.sh/uv/install.sh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs poetry via uv" {
    grep -q 'command -v poetry' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'uv tool install poetry' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs age if missing" {
    grep -q 'command -v age' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'filippo.io/age' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs eza if missing" {
    grep -q 'command -v eza' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'eza.*linux.*tar.gz' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs jq if missing" {
    grep -q 'command -v jq' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'jq-linux-amd64' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs gh if missing" {
    grep -q 'command -v gh' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'cli/cli/releases' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs zoxide if missing" {
    grep -q 'command -v zoxide' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'zoxide.*install.sh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs direnv if missing" {
    grep -q 'command -v direnv' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'direnv.net/install.sh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs shellcheck if missing" {
    grep -q 'command -v shellcheck' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'ShellCheck/releases' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs bats if missing" {
    grep -q 'command -v bats' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'bats-core/bats-core' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh skips tools already installed" {
    grep -q 'uv already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'poetry already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'age already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'eza already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'jq already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'gh already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'zoxide already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'direnv already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'shellcheck already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'bats already installed' "$DOTFILES_DIR/setup-linux.sh"
}

# --- tmux integration ---

@test "setup-linux.sh copies tmux.conf into deploy dir" {
    grep -qE 'safe_copy "\$CURRENT_DIR/tmux\.conf" "\$DOTFILES_DIR/"' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh symlinks tmux.conf to ~/.tmux.conf" {
    grep -qE 'ln -sf "\$DOTFILES_DIR/tmux\.conf" "\$HOME/\.tmux\.conf"' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh checks for tmux presence" {
    grep -qE 'command -v tmux' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh tells user how to install tmux when missing" {
    grep -qE 'sudo apt install -y tmux' "$DOTFILES_DIR/setup-linux.sh"
}

# --- SessionStart hook registration (issue #20 prevention) ---

@test "setup-linux.sh registers SessionStart hook in Claude settings" {
    grep -q 'SessionStart' "$DOTFILES_DIR/setup-linux.sh"
}

# Hook command must point at the canonical deploy directory under ~/.dotfiles/scripts,
# matching where the install step actually lands claude-session-start.sh.
@test "setup-linux.sh SessionStart hook command points at dotfiles scripts dir" {
    grep -qE 'EXPECTED_HOOK_COMMAND="\$HOME/\.dotfiles/scripts/claude-session-start\.sh"' "$DOTFILES_DIR/setup-linux.sh"
}

# Hook registration must self-heal -- never trust "an entry exists" to mean
# "the entry is correct". Post-SDD-002 (PR #51): merge_claude_settings ALWAYS
# rewrites .hooks.SessionStart from the template (with __HOOK_COMMAND__
# substituted), which is a stronger guarantee than the previous compare-then-
# rewrite. Mirrors the Windows guard.
@test "setup-linux.sh SessionStart hook self-heals on path drift" {
    grep -qF 'EXPECTED_HOOK_COMMAND' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'merge_claude_settings "$CLAUDE_SETTINGS_TEMPLATE"' "$DOTFILES_DIR/setup-linux.sh"
}

# --- MCP server registration (SSOT + idempotence) ---

@test "mcp-servers.json exists and is valid JSON with at least one server" {
    [ -f "$DOTFILES_DIR/mcp-servers.json" ]
    if command -v jq >/dev/null 2>&1; then
        run jq -e '.servers | length > 0' "$DOTFILES_DIR/mcp-servers.json"
        [ "$status" -eq 0 ]
    fi
}

# MCP server list must live in mcp-servers.json, not hardcoded in setup-linux.sh.
@test "setup-linux.sh MCP registration reads from mcp-servers.json" {
    grep -q 'mcp-servers\.json' "$DOTFILES_DIR/setup-linux.sh"
    ! grep -qE 'claude mcp add --transport (stdio|http) (drawio|socket|context7|sequential-thinking|hive)' "$DOTFILES_DIR/setup-linux.sh"
}

# MCP registration must check existence before adding, not blindly retry.
@test "setup-linux.sh MCP registration checks existence with claude mcp get" {
    grep -q 'claude mcp get' "$DOTFILES_DIR/setup-linux.sh"
}

# Both setup scripts must read the same SSOT.
@test "parity: both setup scripts source mcp-servers.json" {
    grep -q 'mcp-servers\.json' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'mcp-servers\.json' "$DOTFILES_DIR/setup-windows.ps1"
}

# --- claude-mem-heal cross-OS parity ---

@test "claude-mem-heal.ps1 exists alongside the bash version" {
    [ -f "$DOTFILES_DIR/scripts/claude-mem-heal.sh" ]
    [ -f "$DOTFILES_DIR/scripts/claude-mem-heal.ps1" ]
}

# Both heal scripts must use --ignore-scripts to avoid triggering native
# postinstalls (tree-sitter -> node-gyp -> MSBuild on Windows, build-essential
# on Linux). Only zod's pure-JS files are needed.
@test "claude-mem-heal scripts both use --ignore-scripts on npm install" {
    grep -q -- '--ignore-scripts' "$DOTFILES_DIR/scripts/claude-mem-heal.sh"
    grep -q -- '--ignore-scripts' "$DOTFILES_DIR/scripts/claude-mem-heal.ps1"
}

# Both SessionStart hooks must invoke the heal at session start.
@test "parity: both SessionStart hooks invoke claude-mem-heal" {
    grep -q 'claude-mem-heal\.sh' "$DOTFILES_DIR/scripts/claude-session-start.sh"
    grep -q 'claude-mem-heal\.ps1' "$DOTFILES_DIR/scripts/claude-session-start.ps1"
}

# --- BUG-004: defense-in-depth around claude plugin install (truncate guard) ---
# Linux mirror of the Windows guard. Every `claude plugin install` call triggers
# upstream anthropics/claude-code#59870, dropping subscription fields out of
# ~/.claude/.claude.json. The bash idempotence guard (`grep -qF` against
# `claude plugin list` output) yields a false negative for claude-mem@thedotmack
# because it does not appear in that listing -- so every run installs it again,
# truncating .claude.json from ~75 KB to ~1.5 KB. Defense in depth: snapshot
# before the call, restore if shrinks >50% from a baseline of >=10 KB.

@test "setup-linux.sh defines snapshot_claude_json + restore_claude_json_if_truncated (BUG-004)" {
    grep -q 'snapshot_claude_json()' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'restore_claude_json_if_truncated()' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh cites upstream issue 59870 in the truncate guard (BUG-004)" {
    grep -qF '#59870' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh uses 10240-byte sanity floor in the truncate guard (BUG-004)" {
    grep -qF '10240' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh wraps claude plugin install with snapshot+restore (BUG-004)" {
    # snapshot called before, restore called after, both within the foreach loop body.
    grep -B5 'claude plugin install "\$plugin"' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'snapshot_claude_json'
    grep -A10 'claude plugin install "\$plugin"' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'restore_claude_json_if_truncated'
}

@test "setup-linux.sh still preserves the upstream idempotence guard (BUG-004)" {
    # Defense in depth -- the wrapper does NOT replace the existing guard.
    grep -qF 'grep -qF "$plugin"' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'claude plugin list' "$DOTFILES_DIR/setup-linux.sh"
}

@test "parity: both setup scripts cite upstream issue 59870 in the truncate guard (BUG-004)" {
    grep -qF '#59870' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF '#59870' "$DOTFILES_DIR/setup-windows.ps1"
}

@test "parity: both setup scripts wrap claude plugin install with a snapshot helper (BUG-004)" {
    # Windows uses the combined PS-idiomatic name; bash uses snapshot_/restore_ pair.
    grep -q 'Backup-AndRestoreClaudeJson' "$DOTFILES_DIR/setup-windows.ps1"
    grep -q 'snapshot_claude_json' "$DOTFILES_DIR/setup-linux.sh"
}

# --- BUG-011: extend the BUG-004 guard to every claude CLI call site ---
# BUG-004 covered only `claude plugin install`. The same upstream truncation
# (anthropics/claude-code#59870) fires on `claude mcp get`, `claude mcp add`, and
# `claude plugin list` because they go through the same deserialize-modify-
# serialize cycle. With ~9 MCP servers, each setup run unwrapped triggered
# ~18 chances of truncation. These asserts lock in the wrap on every call site.

@test "setup-linux.sh defines snapshot helpers BEFORE the MCP loop (BUG-011)" {
    # No forward references: helper definitions must precede first use.
    helper_line=$(grep -n '^snapshot_claude_json()' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    mcp_get_line=$(grep -n 'claude mcp get' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    [ -n "$helper_line" ] && [ -n "$mcp_get_line" ]
    [ "$helper_line" -lt "$mcp_get_line" ]
}

@test "setup-linux.sh wraps claude mcp add with snapshot+restore (BUG-011)" {
    # snapshot called within 15 lines before mcp add (header comments + mcp get
    # idempotence path live between); restore within 10 lines after.
    grep -B15 'claude mcp add --transport' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'snapshot_claude_json'
    grep -A10 'claude mcp add --transport' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'restore_claude_json_if_truncated'
}

@test "setup-linux.sh wraps claude plugin list with snapshot+restore (BUG-011)" {
    grep -B5 'claude plugin list 2>/dev/null' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'snapshot_claude_json'
    grep -A5 'claude plugin list 2>/dev/null' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'restore_claude_json_if_truncated'
}

@test "parity: both setup scripts wrap claude mcp add with the snapshot guard (BUG-011)" {
    grep -B15 'claude mcp add --transport' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'snapshot_claude_json'
    grep -B15 'claude mcp add --transport' "$DOTFILES_DIR/setup-windows.ps1" | grep -q 'Backup-AndRestoreClaudeJson'
}

@test "parity: both setup scripts wrap claude plugin list with the snapshot guard (BUG-011)" {
    grep -B5 'claude plugin list' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'snapshot_claude_json'
    grep -B5 'claude plugin list' "$DOTFILES_DIR/setup-windows.ps1" | grep -q 'Backup-AndRestoreClaudeJson'
}

# --- doctor + env-contract.json (cross-OS parity) ---

@test "env-contract.json exists and is valid JSON with required sections" {
    [ -f "$DOTFILES_DIR/env-contract.json" ]
    if command -v jq >/dev/null 2>&1; then
        run jq -e '.env_vars | length > 0' "$DOTFILES_DIR/env-contract.json"
        [ "$status" -eq 0 ]
        run jq -e '.required_path_entries.linux | length > 0' "$DOTFILES_DIR/env-contract.json"
        [ "$status" -eq 0 ]
        run jq -e '.required_path_entries.windows | length > 0' "$DOTFILES_DIR/env-contract.json"
        [ "$status" -eq 0 ]
        run jq -e '.required_binaries | length > 0' "$DOTFILES_DIR/env-contract.json"
        [ "$status" -eq 0 ]
    fi
}

@test "doctor.sh and doctor.ps1 both exist (cross-OS parity)" {
    [ -f "$DOTFILES_DIR/scripts/doctor.sh" ]
    [ -f "$DOTFILES_DIR/scripts/doctor.ps1" ]
}

@test "doctor.sh has valid bash syntax" {
    bash -n "$DOTFILES_DIR/scripts/doctor.sh"
}

@test "doctor.sh supports --check and --fix modes" {
    grep -q -- '--check' "$DOTFILES_DIR/scripts/doctor.sh"
    grep -q -- '--fix' "$DOTFILES_DIR/scripts/doctor.sh"
}

@test "doctor.ps1 supports -Fix switch" {
    grep -q '\[switch\]\$Fix' "$DOTFILES_DIR/scripts/doctor.ps1"
}

# Setup scripts must invoke the doctor before printing 'Setup Complete'.
@test "parity: both setup scripts invoke doctor post-setup" {
    grep -q 'doctor\.sh' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'doctor\.ps1' "$DOTFILES_DIR/setup-windows.ps1"
}

# SessionStart hooks must run a silent doctor and surface only drift.
@test "parity: both SessionStart hooks invoke silent doctor" {
    grep -q 'doctor\.sh' "$DOTFILES_DIR/scripts/claude-session-start.sh"
    grep -q 'doctor\.ps1' "$DOTFILES_DIR/scripts/claude-session-start.ps1"
}

# Required binaries in the contract must include min_version pins.
@test "env-contract.json pins min_version for required binaries" {
    if command -v jq >/dev/null 2>&1; then
        run jq -e '.required_binaries | all(has("min_version"))' "$DOTFILES_DIR/env-contract.json"
        [ "$status" -eq 0 ]
    fi
}

# Both doctors must implement the binary version-check logic.
@test "parity: both doctors check binary versions against min_version" {
    grep -q 'min_version' "$DOTFILES_DIR/scripts/doctor.sh"
    grep -q 'min_version' "$DOTFILES_DIR/scripts/doctor.ps1"
}

# Doctor sections must emit a one-line summary so non-verbose runs never
# show an empty header (regression guard for the post-#24 UX bug).
@test "parity: both doctors emit per-section summaries" {
    grep -q 'section_summary' "$DOTFILES_DIR/scripts/doctor.sh"
    grep -q 'Write-SectionSummary' "$DOTFILES_DIR/scripts/doctor.ps1"
}

# Profiles must export the structural env vars declared in env-contract.json,
# so a fresh shell silences doctor without needing --fix every session.
@test ".zshrc exports DOTFILES_DIR and CLAUDE_CONFIG_DIR" {
    grep -q 'export DOTFILES_DIR=' "$DOTFILES_DIR/.zshrc"
    grep -q 'export CLAUDE_CONFIG_DIR=' "$DOTFILES_DIR/.zshrc"
}

@test ".bashrc exports DOTFILES_DIR and CLAUDE_CONFIG_DIR" {
    grep -q 'export DOTFILES_DIR=' "$DOTFILES_DIR/.bashrc"
    grep -q 'export CLAUDE_CONFIG_DIR=' "$DOTFILES_DIR/.bashrc"
}

@test "powershell/profile.ps1 sets DOTFILES_DIR and CLAUDE_CONFIG_DIR" {
    grep -q '\$env:DOTFILES_DIR' "$DOTFILES_DIR/powershell/profile.ps1"
    grep -q '\$env:CLAUDE_CONFIG_DIR' "$DOTFILES_DIR/powershell/profile.ps1"
}
