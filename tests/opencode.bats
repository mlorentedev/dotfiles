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

@test "setup-linux.sh converges a drifted opencode to the versions.conf pin (REFACTOR-011)" {
    # Presence is not convergence: an opencode older than OPENCODE_VERSION was
    # previously skipped as "already installed", leaving healthcheck flagging
    # version drift on every run.
    grep -q 'installed_opencode" != "$OPENCODE_VERSION' "$SETUP_SCRIPT"
    # Convergence is verified by re-query, not by installer exit code: a
    # shadowing install (npm global) would otherwise produce a false SUCCESS.
    grep -q 'shadows it in PATH' "$SETUP_SCRIPT"
}

# Version drift is advisory: the pinned tool still works, just outdated. Drift
# must WARN (exit-neutral), never FAIL. The .sh side is now go test
# (TestCheckOptionalTools_DotfDrift, TestCheckVersionMatch); healthcheck.ps1
# keeps the assertion until the Windows port.
@test "healthcheck.ps1: version drift reports WARN not FAIL" {
    grep -qE 'Write-Warn "(yarn|opencode|pi) version drift' "$DOTFILES_DIR/scripts/healthcheck.ps1"
    ! grep -qE 'Write-Fail "(yarn|opencode|pi) version drift' "$DOTFILES_DIR/scripts/healthcheck.ps1"
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

@test "opencode.jsonc openrouter picker is curated to the shared free+NaN set (#300, no big-3)" {
    # An explicit models map (not an empty {}) restricts the /models picker, and
    # must match pi's ai/pi/settings.json so the two agents are interchangeable.
    grep -q '"qwen/qwen3-coder:free":' "$OPENCODE_CFG"
    grep -q '"deepseek/deepseek-v4-pro":' "$OPENCODE_CFG"
    grep -q '"minimax/minimax-m3":' "$OPENCODE_CFG"
    # No OpenAI/Google/Anthropic OpenRouter model entries.
    ! grep -qE '"(openai|google|anthropic)/[^"]+":[[:space:]]*\{[[:space:]]*"name"' "$OPENCODE_CFG"
}

@test "opencode.jsonc has ollama provider (homelab via VPN)" {
    grep -q '"ollama":' "$OPENCODE_CFG"
    grep -q 'ollama.kubelab.live' "$OPENCODE_CFG"
}

@test "opencode.jsonc default model is nan/qwen3.6 (fast default; deepseek-v4-flash on-demand)" {
    # qwen3.6 is the default (256K, ~0.8s, fast) — matches the model-tier header
    # comment. deepseek-v4-flash (1M context, reasoning-heavy ~3s) is reached
    # on-demand via the `qf` wrapper, not as the always-on default; the tight
    # SSE chunkTimeout made it abort with "SSE timeout" as the default. Switch
    # the active model via /models per task.
    grep -qE '"model":\s*"nan/qwen3.6"' "$OPENCODE_CFG"
}

@test "opencode.jsonc exposes 4 chat NaN models (non-chat models intentionally excluded — opencode schema rejects 'embedding' modality)" {
    for m in deepseek-v4-flash qwen3.6 gemma4 mimo-v2.5; do
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

@test "opencode.jsonc DX keys present (share disabled, autoupdate notify, providers pruned, tool_output, read-only plan agent)" {
    grep -qE '"share":\s*"disabled"' "$OPENCODE_CFG"
    grep -qE '"autoupdate":\s*"notify"' "$OPENCODE_CFG"
    grep -qE '"disabled_providers":\s*\[' "$OPENCODE_CFG"
    grep -qE '"tool_output":\s*\{' "$OPENCODE_CFG"
    grep -qE '"max_lines":' "$OPENCODE_CFG"
    # plan agent must exist, pin the fast model, and hard-deny writes
    grep -qE '"agent":\s*\{' "$OPENCODE_CFG"
    grep -qE '"plan":\s*\{' "$OPENCODE_CFG"
}

# --- Alias ---

@test ".zsh/functions.sh defines oc for opencode (--pure: skip MCPs/plugins) [REFACTOR-010]" {
    # --pure bypasses MCP+plugins to avoid the tool-resolution hang on complex
    # queries (empirical 2026-05-25). Shared bash/zsh core in functions.sh;
    # Windows keeps plain opencode (see powershell-profile.bats). Moved out of
    # aliases.zsh per REFACTOR-010 to dedup the bash/zsh wrappers.
    grep -qE '^oc\(\) \{ opencode --pure' "$DOTFILES_DIR/.zsh/functions.sh"
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

@test "ai/claude/CLAUDE.md is a pointer to AGENTS.md (≤ 100 lines)" {
    # Threshold bumped 70→80 in AI-019 (model-tier overlay added ~8 lines).
    # Bumped 80→100 in ENGINE-001: the HARNESS engine injects the generated
    # "Overrides of Harness Defaults" block (no-attribution + english-only +
    # no-phase-refs) sourced from the vault, which needs the extra headroom.
    # Future per-agent extensions should justify each bump in the spec.
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/claude/CLAUDE.md"
    [[ $(wc -l < "$DOTFILES_DIR/ai/claude/CLAUDE.md") -le 100 ]]
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

# --- OpenCode diagnostics integration ---
# The healthcheck.sh OpenCode section (binary + config + $schema) was ported to
# cli/internal/doctor (checkOpenCode), covered by go test. The structural .sh
# greps that lived here are retired with the script.

# --- DX-004: reasoning visibility (interleaved) + TUI config (tui.json) ---

@test "opencode.jsonc maps NaN reasoning_content via interleaved on all 4 chat models (DX-004 AC1)" {
    # opencode only renders NaN's reasoning chain if reasoning_content is mapped
    # to a reasoning part. All four chat models must carry the mapping.
    count=$(grep -cE '"interleaved":[[:space:]]*\{[[:space:]]*"field":[[:space:]]*"reasoning_content"[[:space:]]*\}' "$OPENCODE_CFG")
    [ "$count" -eq 4 ] || { echo "expected 4 interleaved blocks, got $count" >&2; false; }
}

@test "opencode.jsonc reasoning comment updated off the stale 1.15.10 'renders neither' note (DX-004 AC5)" {
    ! grep -q 'Opencode (1.15.10) renders neither' "$OPENCODE_CFG"
    grep -q 'interleaved' "$OPENCODE_CFG"
}

@test "ai/opencode/tui.json exists with schema, theme=opencode, display_thinking=ctrl+o (DX-004 AC2)" {
    TUI_CFG="$DOTFILES_DIR/ai/opencode/tui.json"
    [ -f "$TUI_CFG" ]
    grep -qE '"\$schema":[[:space:]]*"https://opencode.ai/tui.json"' "$TUI_CFG"
    grep -qE '"theme":[[:space:]]*"opencode"' "$TUI_CFG"
    grep -qE '"display_thinking":[[:space:]]*"ctrl\+o"' "$TUI_CFG"
}

@test "setup-linux.sh deploys tui.json as a plain copy, no secret substitution (DX-004 AC3)" {
    grep -q 'TUI_SRC="\$CURRENT_DIR/ai/opencode/tui.json"' "$SETUP_SCRIPT"
    grep -q 'cmp -s "\$TUI_SRC" "\$TUI_DST"' "$SETUP_SCRIPT"
    # tui.json carries no secrets: it must NOT go through substitute_env_placeholders
    ! grep -qE 'substitute_env_placeholders "\$TUI_(SRC|DST)"' "$SETUP_SCRIPT"
}
