#!/usr/bin/env bats
# Tests for OpenCode bootstrap (spec AI-011-opencode-bootstrap)
# Relational tests per pattern-setup-script-idempotence — verify invariants,
# not just presence.

load 'lib/refute'

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

@test "opencode is a packages.json catalog tool (npm) and setup-linux.sh carries no install block (AI-034, ADR-036)" {
    jq -e '.tools[] | select(.name=="opencode" and .source.type=="npm" and .source.package=="opencode-ai")' "$DOTFILES_DIR/packages.json" >/dev/null
    refute_grep_fixed 'opencode.ai/install' "$SETUP_SCRIPT"
    refute_grep_fixed 'OPENCODE_BIN_DIR=' "$SETUP_SCRIPT"
    refute_grep_fixed 'OPENCODE_VERSION' "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode config deploy is declarative always-overwrite (no cmp -s skip)" {
    # fix/linux-deploy-always-overwrite: the opencode block aligns with the
    # Windows always-overwrite strategy. The staged-substituted tmp file is
    # moved into place on every run (substitution may have changed content);
    # there is no "already in sync" skip path that could mask drift.
    refute_grep 'cmp -s "\$OPENCODE_CONFIG_TMP" "\$OPENCODE_CONFIG_DST"' "$SETUP_SCRIPT"
    grep -q 'mv "\$OPENCODE_CONFIG_TMP" "\$OPENCODE_CONFIG_DST"' "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode deploy renders via dotf secrets render (SDD-009/#587)" {
    grep -q 'dotf secrets render "\$OPENCODE_CONFIG_TMP"' "$SETUP_SCRIPT"
}

@test "setup-linux.sh opencode block has post-deploy assertion" {
    grep -q "opencode not reachable on PATH" "$SETUP_SCRIPT"
}

@test "the retired ~/.opencode/bin channel is off PATH in .zshrc and .bashrc (AI-034, ADR-036)" {
    # The curl-script copy would otherwise keep shadowing the npm one on the
    # Linux box; doctor names the leftover, the rc files stop reaching for it.
    refute_grep_fixed '.opencode/bin' "$DOTFILES_DIR/.zshrc"
    refute_grep_fixed '.opencode/bin' "$DOTFILES_DIR/.bashrc"
}

@test "packages.json is the only opencode pin (versions.conf carries none, ADR-036)" {
    refute_grep '^OPENCODE_VERSION=' "$DOTFILES_DIR/versions.conf"
    [ "$(jq -r '.tools[] | select(.name=="opencode") | .version' "$DOTFILES_DIR/packages.json")" != "" ]
}

# CLI-018: version-drift-is-WARN (yarn/opencode/pi) was asserted against
# healthcheck.ps1; it now lives in go test (TestCheckOptionalTools_DotfDrift,
# TestCheckVersionMatch) after the .ps1 was retired.

@test "setup-linux.sh opencode block has no new silenced errors (2>/dev/null || true)" {
    # Count silenced errors in opencode block specifically (between OPENCODE marker and next blank-line section)
    [[ $(awk '/^# OpenCode/,/^# GitHub Copilot/' "$SETUP_SCRIPT" | grep -c '2>/dev/null || true') -eq 0 ]]
}

# --- Regression: aider sunset ---

@test "setup-linux.sh no longer installs aider" {
    refute_grep_fixed "uv tool install --python 3.12 aider-chat" "$SETUP_SCRIPT"
    refute_grep_fixed "# Aider (AI pair programming)" "$SETUP_SCRIPT"
}

@test "setup-windows.ps1 no longer installs aider" {
    refute_grep_fixed "Installing aider-chat via uv" "$DOTFILES_DIR/setup-windows.ps1"
    refute_grep_fixed "DEPLOY AIDER CONFIGURATION" "$DOTFILES_DIR/setup-windows.ps1"
}

@test "setup-linux.sh keeps uv install (used by hive MCP server)" {
    grep -q "Installing uv" "$SETUP_SCRIPT"
    grep -q "https://astral.sh/uv/install.sh" "$SETUP_SCRIPT"
}

@test "ai/aider/ directory removed from repo" {
    [[ ! -d "$DOTFILES_DIR/ai/aider" ]]
}

@test ".zsh/aliases.zsh no longer has aider tier aliases" {
    refute_grep '^alias (ai|aic|aia)=' "$ALIASES_FILE"
}

@test "powershell/profile.ps1 no longer has aider tier functions" {
    refute_grep '^function (ai|aic|aia) ' "$DOTFILES_DIR/powershell/profile.ps1"
}

@test "AGENTS.md no longer lists Aider in the agent matrix" {
    refute_grep 'and Aider all read this file' "$AGENTS_MD"
}

@test "README.md no longer mentions Aider" {
    run grep -in 'aider' "$DOTFILES_DIR/README.md"
    [ "$status" -eq 1 ] || { printf 'README still mentions aider:\n%s\n' "$output" >&2; return 1; }
}

@test "env-contract.json uv purpose no longer mentions aider" {
    refute_grep_fixed '"Python tool installs (aider' "$DOTFILES_DIR/env-contract.json"
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
    refute_grep '"(openai|google|anthropic)/[^"]+":[[:space:]]*\{[[:space:]]*"name"' "$OPENCODE_CFG"
}

# The ollama provider is gone and must stay gone: the homelab endpoint no longer
# runs, and its slot was never inert. It named {env:OLLAMA_API_KEY}, the shell
# wrappers named OLLAMA_API_KEY in their `--only` list, and no registry entry ever
# existed for it -- so `dotf secrets run` failed loud and opencode did not start.
# Comment lines are excluded deliberately: the removal is documented in place,
# and a whole-file refute fails on the explanation of the thing it checks. What
# must not return is the declaration, not the record of why it went.
@test "opencode.jsonc declares no ollama provider" {
    run bash -c "grep -vE '^[[:space:]]*//' '$OPENCODE_CFG' | grep -nE '\"ollama\"[[:space:]]*:|ollama\.kubelab\.live|OLLAMA_API_KEY' || true"
    [ -z "$output" ] || { echo "ollama is still declared: $output"; false; }
}

@test "opencode.jsonc default model is nan/qwen3.8-flash" {
    grep -qE '"model":\s*"nan/qwen3.8-flash"' "$OPENCODE_CFG"
}

@test "opencode.jsonc exposes 6 chat NaN models (non-chat models intentionally excluded - opencode schema rejects 'embedding' modality)" {
    for m in deepseek-v4-flash qwen3.6 gemma4 mimo-v2.5 qwen3.8-flash glm5.3-flash; do
        grep -qE "\"$m\":" "$OPENCODE_CFG" || { echo "missing chat model $m" >&2; false; }
    done
    # Non-chat models must NOT appear (would break config load)
    refute_grep '"qwen3-embedding":|"kokoro":|"whisper":' "$OPENCODE_CFG"
}

@test "opencode.jsonc no longer references opencode-go (Go subscription cancelled per SDD-007)" {
    refute_grep_fixed '"opencode-go":' "$OPENCODE_CFG"
}

@test "opencode.jsonc mirrors the 3 active MCP servers (hive, context7, sequential-thinking)" {
    for srv in sequential-thinking context7 hive; do
        grep -qE "\"$srv\":" "$OPENCODE_CFG" || { echo "missing MCP $srv" >&2; false; }
    done
    # drawio + socket intentionally removed (see opencode.jsonc comment)
    refute_grep '^\s*"drawio":|^\s*"socket":' "$OPENCODE_CFG"
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

@test "ai/claude/CLAUDE.md is a pointer to AGENTS.md (<= 100 lines)" {
    # Threshold bumped 70→80 in AI-019 (model-tier overlay added ~8 lines).
    # Bumped 80→100 in ENGINE-001: the HARNESS engine injects the generated
    # "Overrides of Harness Defaults" block (no-attribution + english-only +
    # no-phase-refs) sourced from the vault, which needs the extra headroom.
    # Future per-agent extensions should justify each bump in the spec.
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/claude/CLAUDE.md"
    [[ $(wc -l < "$DOTFILES_DIR/ai/claude/CLAUDE.md") -le 100 ]]
}

@test "ai/agy/AGY.md is a pointer to AGENTS.md (<= 50 lines)" {
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/agy/AGY.md"
    [[ $(wc -l < "$DOTFILES_DIR/ai/agy/AGY.md") -le 50 ]]
}

@test "ai/copilot/copilot-instructions.md is a pointer (no template-placeholder bug)" {
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/copilot/copilot-instructions.md"
    # Bug fix regression: must not contain the broken placeholder string
    refute_grep_fixed "the knowledge base (\`~/Projects/knowledge/" "$DOTFILES_DIR/ai/copilot/copilot-instructions.md"
    # Bug fix regression: must not contain the wrong Apps\knowledge path
    refute_grep "Apps\\\\knowledge" "$DOTFILES_DIR/ai/copilot/copilot-instructions.md"
}

@test ".github/copilot-instructions.md is a pointer (no template-placeholder bug)" {
    grep -q "First, read \[\`AGENTS.md\`\]" "$DOTFILES_DIR/.github/copilot-instructions.md"
    refute_grep_fixed "the knowledge base (\`~/Projects/knowledge/" "$DOTFILES_DIR/.github/copilot-instructions.md"
    refute_grep "Apps\\\\knowledge" "$DOTFILES_DIR/.github/copilot-instructions.md"
}

# --- OpenCode diagnostics integration ---
# The healthcheck.sh OpenCode section (binary + config + $schema) was ported to
# cli/internal/doctor (checkOpenCode), covered by go test. The structural .sh
# greps that lived here are retired with the script.

# --- DX-004: reasoning visibility (interleaved) + TUI config (tui.json) ---

@test "opencode.jsonc maps NaN reasoning_content via interleaved on all 6 chat models (DX-004 AC1)" {
    # opencode only renders NaN's reasoning chain if reasoning_content is mapped
    # to a reasoning part. All six chat models must carry the mapping.
    count=$(grep -cE '"interleaved":[[:space:]]*\{[[:space:]]*"field":[[:space:]]*"reasoning_content"[[:space:]]*\}' "$OPENCODE_CFG")
    [ "$count" -eq 6 ] || { echo "expected 6 interleaved blocks, got $count" >&2; false; }
}

@test "opencode.jsonc reasoning comment updated off the stale 1.15.10 'renders neither' note (DX-004 AC5)" {
    refute_grep_fixed 'Opencode (1.15.10) renders neither' "$OPENCODE_CFG"
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
    # tui.json carries no secrets: it must NOT go through dotf secrets render
    refute_grep 'dotf secrets render "\$TUI_(SRC|DST)"' "$SETUP_SCRIPT"
}
