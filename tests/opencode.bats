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
    grep -q 'cmp -s "\$OPENCODE_CONFIG_SRC" "\$OPENCODE_CONFIG_DST"' "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode block has post-deploy assertion" {
    grep -q "opencode binary not reachable" "$SETUP_SCRIPT"
}

@test "setup-linux.sh adds opencode PATH via ensure_line_in_file (no manual sed)" {
    grep -q 'ensure_line_in_file "\$HOME/.zshrc" "\$OPENCODE_PATH_LINE"' "$SETUP_SCRIPT"
    grep -q 'ensure_line_in_file "\$HOME/.bashrc" "\$OPENCODE_PATH_LINE"' "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode install URL uses opencode.ai (not anomalyco fork)" {
    grep -q 'curl -fsSL https://opencode.ai/install' "$SETUP_SCRIPT"
    ! grep -q 'curl.*anomalyco' "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode block has no new silenced errors (2>/dev/null || true)" {
    # Count silenced errors in opencode block specifically (between OPENCODE marker and next blank-line section)
    [[ $(awk '/^# OpenCode/,/^# GitHub Copilot/' "$SETUP_SCRIPT" | grep -c '2>/dev/null || true') -eq 0 ]]
}

# --- Regression: aider coexistence (PR1 keeps aider; sunset is PR2) ---

@test "setup-linux.sh STILL has aider install block (PR1 coexistence)" {
    grep -q "# Aider (AI pair programming)" "$SETUP_SCRIPT"
    grep -q "uv tool install --python 3.12 aider-chat" "$SETUP_SCRIPT"
}

@test ".zsh/aliases.zsh STILL has aider aliases (PR1 coexistence)" {
    grep -q '^alias ai="aider"' "$ALIASES_FILE"
    grep -q '^alias aic="aider' "$ALIASES_FILE"
    grep -q '^alias aia="aider' "$ALIASES_FILE"
}

# --- opencode.jsonc structure ---

@test "ai/opencode/opencode.jsonc exists" {
    [[ -f "$OPENCODE_CFG" ]]
}

@test "opencode.jsonc declares \$schema" {
    grep -q '"\$schema": "https://opencode.ai/config.json"' "$OPENCODE_CFG"
}

@test "opencode.jsonc has opencode-go provider (Go subscription, restricted catalog)" {
    grep -q '"opencode-go":' "$OPENCODE_CFG"
}

@test "opencode.jsonc has openrouter provider (env-detected OPENROUTER_API_KEY)" {
    grep -q '"openrouter":' "$OPENCODE_CFG"
}

@test "opencode.jsonc default model is in Go catalog (deepseek-v4-pro or kimi)" {
    grep -qE '"model":\s*"opencode-go/(deepseek-v4-pro|kimi-k2\.6)"' "$OPENCODE_CFG"
}

@test "opencode.jsonc does NOT expose Zen PAYG frontier models (guardrail layer 1)" {
    # Should not list Sonnet, GPT-5, Opus, Gemini Pro under opencode-go
    ! grep -qE 'claude-(opus|sonnet)|gpt-[45]|gemini-[23]\.[0-9]+-pro' "$OPENCODE_CFG"
}

@test "opencode.jsonc mirrors all 5 MCP servers from mcp-servers.json" {
    for srv in drawio socket sequential-thinking context7 hive; do
        grep -q "\"$srv\":" "$OPENCODE_CFG"
    done
}

# --- Alias ---

@test ".zsh/aliases.zsh has oc alias for opencode" {
    grep -q '^alias oc="opencode"' "$ALIASES_FILE"
}

# --- AGENTS.md (SSOT cross-agent per ADR-009) ---

@test "AGENTS.md exists at repo root" {
    [[ -f "$AGENTS_MD" ]]
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

@test "ai/claude/CLAUDE.md is a pointer to AGENTS.md (≤ 70 lines)" {
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/claude/CLAUDE.md"
    [[ $(wc -l < "$DOTFILES_DIR/ai/claude/CLAUDE.md") -le 70 ]]
}

@test "ai/gemini/GEMINI.md is a pointer to AGENTS.md (≤ 50 lines)" {
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/gemini/GEMINI.md"
    [[ $(wc -l < "$DOTFILES_DIR/ai/gemini/GEMINI.md") -le 50 ]]
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

@test "healthcheck.sh has OpenCode section (10/11)" {
    grep -q 'section "10/11" "OpenCode"' "$HEALTHCHECK"
}

@test "healthcheck.sh OpenCode section checks binary + config + schema" {
    awk '/section "10\/11" "OpenCode"/,/section "11\/11"/' "$HEALTHCHECK" | grep -q 'opencode --version'
    awk '/section "10\/11" "OpenCode"/,/section "11\/11"/' "$HEALTHCHECK" | grep -q 'OPENCODE_CFG'
    awk '/section "10\/11" "OpenCode"/,/section "11\/11"/' "$HEALTHCHECK" | grep -q '\$schema'
}
