#!/usr/bin/env bats
# Tests for OpenCode bootstrap (spec AI-011-opencode-bootstrap)
# Relational tests per pattern-setup-script-idempotence — verify invariants,
# not just presence.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SETUP_SCRIPT="$DOTFILES_DIR/setup-linux.sh"
    export OPENCODE_CFG="$DOTFILES_DIR/ai/opencode/opencode.jsonc"
    export AGENTS_MD="$DOTFILES_DIR/AGENTS.md"
    export ALIASES_FILE="$DOTFILES_DIR/.zsh/aliases.zsh"
    export HEALTHCHECK="$DOTFILES_DIR/scripts/healthcheck.sh"
}

# --- setup-linux.sh: opencode install block ---

@test "setup-linux.sh has opencode install block" {
    grep -q "OpenCode (secondary AI coding agent" "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode install is idempotent (command -v gate before install)" {
    grep -q 'if ! command -v opencode' "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode config deploy uses reconcile-not-skip (cmp -s)" {
    # Post-SDD-009: cmp compares the staged-substituted tmp file (not the raw
    # source) against the deployed file, so substitution-driven content changes
    # still trigger a redeploy without breaking idempotence.
    grep -q 'cmp -s "\$OPENCODE_CONFIG_TMP" "\$OPENCODE_CONFIG_DST"' "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode deploy calls substitute_env_placeholders (SDD-009)" {
    grep -q 'substitute_env_placeholders "\$OPENCODE_CONFIG_TMP"' "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode block has post-deploy assertion" {
    grep -q "opencode binary not reachable" "$SETUP_SCRIPT"
}

@test "opencode PATH is baked into repo .zshrc and .bashrc (SSOT, no setup-time mutation)" {
    grep -q 'export PATH="\$HOME/.opencode/bin:\$PATH"' "$DOTFILES_DIR/.zshrc"
    grep -q 'export PATH="\$HOME/.opencode/bin:\$PATH"' "$DOTFILES_DIR/.bashrc"
    ! grep -q 'ensure_line_in_file.*OPENCODE_PATH_LINE' "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode install URL uses opencode.ai (not anomalyco fork)" {
    grep -q 'curl -fsSL https://opencode.ai/install' "$SETUP_SCRIPT"
    ! grep -q 'curl.*anomalyco' "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode block has no new silenced errors (2>/dev/null || true)" {
    # Count silenced errors in opencode block specifically (between OPENCODE marker and next blank-line section)
    [[ $(awk '/^# OpenCode/,/^# GitHub Copilot/' "$SETUP_SCRIPT" | grep -c '2>/dev/null || true') -eq 0 ]]
}

# --- Regression: aider sunset ---

@test "setup-linux.sh no longer installs aider" {
    ! grep -q "uv tool install --python 3.12 aider-chat" "$SETUP_SCRIPT"
    ! grep -q "# Aider (AI pair programming)" "$SETUP_SCRIPT"
}

@test "setup-windows.ps1 no longer installs aider" {
    ! grep -q "Installing aider-chat via uv" "$DOTFILES_DIR/setup-windows.ps1"
    ! grep -q "DEPLOY AIDER CONFIGURATION" "$DOTFILES_DIR/setup-windows.ps1"
}

@test "setup-linux.sh keeps uv install (used by hive MCP server)" {
    grep -q "Installing uv" "$SETUP_SCRIPT"
    grep -q "https://astral.sh/uv/install.sh" "$SETUP_SCRIPT"
}

@test "ai/aider/ directory removed from repo" {
    [[ ! -d "$DOTFILES_DIR/ai/aider" ]]
}

@test ".zsh/aliases.zsh no longer has aider tier aliases" {
    ! grep -qE '^alias (ai|aic|aia)=' "$ALIASES_FILE"
}

@test "powershell/profile.ps1 no longer has aider tier functions" {
    ! grep -qE '^function (ai|aic|aia) ' "$DOTFILES_DIR/powershell/profile.ps1"
}

@test "AGENTS.md no longer lists Aider in the agent matrix" {
    ! grep -qE 'and Aider all read this file' "$AGENTS_MD"
}

@test "README.md no longer mentions Aider" {
    ! grep -qi "aider" "$DOTFILES_DIR/README.md"
}

@test "env-contract.json uv purpose no longer mentions aider" {
    ! grep -q '"Python tool installs (aider' "$DOTFILES_DIR/env-contract.json"
}

# --- opencode.jsonc structure ---

@test "ai/opencode/opencode.jsonc exists" {
    [[ -f "$OPENCODE_CFG" ]]
}

@test "opencode.jsonc declares \$schema" {
    grep -q '"\$schema": "https://opencode.ai/config.json"' "$OPENCODE_CFG"
}

@test "opencode.jsonc has nan provider (NaN community, primary daily)" {
    grep -q '"nan":' "$OPENCODE_CFG"
    grep -q 'api.nan.builders/v1' "$OPENCODE_CFG"
}

@test "opencode.jsonc has openrouter provider (frontier fallback, PAYG)" {
    grep -q '"openrouter":' "$OPENCODE_CFG"
}

@test "opencode.jsonc has ollama provider (homelab via VPN)" {
    grep -q '"ollama":' "$OPENCODE_CFG"
    grep -q 'ollama.kubelab.live' "$OPENCODE_CFG"
}

@test "opencode.jsonc default model is nan/qwen3.6 (NaN, fast multimodal default)" {
    # Per opencode.jsonc comments: qwen3.6 chosen empirically (~0.8s wall, no
    # forced reasoning) over deepseek-v4-flash (~3s + 30+ reason tokens). Switch
    # to deepseek-v4-flash via /models for >256K context jobs.
    grep -qE '"model":\s*"nan/qwen3.6"' "$OPENCODE_CFG"
}

@test "opencode.jsonc exposes 3 chat NaN models (non-chat models intentionally excluded — opencode schema rejects 'embedding' modality)" {
    for m in deepseek-v4-flash qwen3.6 gemma4; do
        grep -qE "\"$m\":" "$OPENCODE_CFG" || { echo "missing chat model $m" >&2; false; }
    done
    # Non-chat models must NOT appear (would break config load)
    ! grep -qE '"qwen3-embedding":|"kokoro":|"whisper":' "$OPENCODE_CFG"
}

@test "opencode.jsonc no longer references opencode-go (Go subscription cancelled per SDD-007)" {
    ! grep -q '"opencode-go":' "$OPENCODE_CFG"
}

@test "opencode.jsonc mirrors the 3 active MCP servers (hive, context7, sequential-thinking)" {
    for srv in sequential-thinking context7 hive; do
        grep -qE "\"$srv\":" "$OPENCODE_CFG" || { echo "missing MCP $srv" >&2; false; }
    done
    # drawio + socket intentionally removed (see opencode.jsonc comment)
    ! grep -qE '^\s*"drawio":|^\s*"socket":' "$OPENCODE_CFG"
}

# --- Alias ---

@test ".zsh/aliases.zsh has oc alias for opencode (--pure: skip MCPs/plugins)" {
    # --pure bypasses MCP+plugins to avoid the tool-resolution hang on complex
    # queries (empirical 2026-05-25). Mirror in .bashrc and powershell/profile.ps1.
    # Hypothesis still under investigation: hang may be Ghostty-on-Linux specific.
    grep -q '^alias oc="opencode --pure"' "$ALIASES_FILE"
}

# --- AGENTS.md (SSOT cross-agent per ADR-009) ---

@test "AGENTS.md exists at repo root" {
    [[ -f "$AGENTS_MD" ]]
}

@test "setup-linux.sh deploys AGENTS.md to ~/.config/opencode/ (opencode global SSOT)" {
    # opencode reads ~/.config/opencode/AGENTS.md natively (per upstream docs).
    # Unlike claude/agy/copilot which use pointer files, opencode loads the
    # canonical filename so we copy the full SSOT verbatim.
    grep -qF '$HOME/.config/opencode/AGENTS.md' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'AGENTS.md source missing' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-windows.ps1 deploys AGENTS.md to ~/.config/opencode/ (cross-OS parity)" {
    grep -qF "'.config\\opencode\\AGENTS.md'" "$DOTFILES_DIR/setup-windows.ps1"
}

@test "AGENTS.md contains Standing Orders section" {
    grep -q "^## Standing Orders" "$AGENTS_MD"
}

@test "AGENTS.md contains Spec-Driven Development section" {
    grep -q "^## Spec-Driven Development" "$AGENTS_MD"
}

@test "AGENTS.md contains MCP Server Usage Rules" {
    grep -q "## MCP Server Usage Rules" "$AGENTS_MD"
}

# --- Per-agent pointer files (AI-013 fold-in) ---

@test "ai/claude/CLAUDE.md is a pointer to AGENTS.md (≤ 80 lines)" {
    # Threshold bumped 70→80 in AI-019 (model-tier overlay added ~8 lines).
    # Future per-agent extensions should justify each bump in the spec.
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/claude/CLAUDE.md"
    [[ $(wc -l < "$DOTFILES_DIR/ai/claude/CLAUDE.md") -le 80 ]]
}

@test "ai/agy/AGY.md is a pointer to AGENTS.md (≤ 50 lines)" {
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/agy/AGY.md"
    [[ $(wc -l < "$DOTFILES_DIR/ai/agy/AGY.md") -le 50 ]]
}

@test "ai/copilot/copilot-instructions.md is a pointer (no template-placeholder bug)" {
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/copilot/copilot-instructions.md"
    # Bug fix regression: must not contain the broken placeholder string
    ! grep -q "the knowledge base (\`~/Projects/knowledge/" "$DOTFILES_DIR/ai/copilot/copilot-instructions.md"
    # Bug fix regression: must not contain the wrong Apps\knowledge path
    ! grep -q "Apps\\\\knowledge" "$DOTFILES_DIR/ai/copilot/copilot-instructions.md"
}

@test ".github/copilot-instructions.md is a pointer (no template-placeholder bug)" {
    grep -q "First, read \[\`AGENTS.md\`\]" "$DOTFILES_DIR/.github/copilot-instructions.md"
    ! grep -q "the knowledge base (\`~/Projects/knowledge/" "$DOTFILES_DIR/.github/copilot-instructions.md"
    ! grep -q "Apps\\\\knowledge" "$DOTFILES_DIR/.github/copilot-instructions.md"
}

# --- healthcheck.sh integration ---

@test "healthcheck.sh has OpenCode section (10/13)" {
    grep -q 'section "10/13" "OpenCode"' "$HEALTHCHECK"
}

@test "healthcheck.sh OpenCode section checks binary + config + schema" {
    awk '/section "10\/13" "OpenCode"/,/section "11\/13"/' "$HEALTHCHECK" | grep -q 'opencode --version'
    awk '/section "10\/13" "OpenCode"/,/section "11\/13"/' "$HEALTHCHECK" | grep -q 'OPENCODE_CFG'
    awk '/section "10\/13" "OpenCode"/,/section "11\/13"/' "$HEALTHCHECK" | grep -q '\$schema'
}
