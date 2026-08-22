#!/usr/bin/env bats
# Tests for ai/claude/settings.json template (SDD-002)
# The template is the SSOT for the "dotfiles-owned" subset of ~/.claude/settings.json.
# Per-key merge policy is documented in specs/SDD-002-settings-portability/proposal.md.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SETTINGS_TEMPLATE="$DOTFILES_DIR/ai/claude/settings.json"
}

@test "template file exists" {
    [[ -f "$SETTINGS_TEMPLATE" ]]
}

@test "template is valid JSON (jq parses without error)" {
    jq empty "$SETTINGS_TEMPLATE"
}

# --- Top-level keys (the "ours" subset) ---

@test "template has model = opus" {
    [[ "$(jq -r '.model' "$SETTINGS_TEMPLATE")" == "opus" ]]
}

@test "template has effortLevel = xhigh" {
    [[ "$(jq -r '.effortLevel' "$SETTINGS_TEMPLATE")" == "xhigh" ]]
}

@test "every dotfiles-owned top-level key is named in both merge policies" {
    # The merge policy is an ALLOW-LIST in two places -- a jq expression in
    # setup-linux.sh and an if-chain in setup-windows.ps1. A key added to the
    # template and named in neither is a SILENT no-op on every existing
    # installation, reaching only machines bootstrapped from scratch. Measured:
    # outputStyle sat in the template while the deployed settings.json had no
    # such key at all.
    #
    # Scoped to the scalar top-level keys. The structured ones (permissions,
    # hooks, enabledPlugins, env) have their own merge semantics asserted below
    # and are handled by name in both scripts.
    local key
    for key in model effortLevel outputStyle; do
        jq -e --arg k "$key" 'has($k)' "$SETTINGS_TEMPLATE" >/dev/null || continue
        grep -q "\.$key = \$tmpl\.$key\|\$tmpl\.$key" "$DOTFILES_DIR/setup-linux.sh" \
            || { echo "setup-linux.sh merge policy never mentions '$key'" >&2; return 1; }
        grep -q "ContainsKey('$key')" "$DOTFILES_DIR/setup-windows.ps1" \
            || { echo "setup-windows.ps1 merge policy never mentions '$key'" >&2; return 1; }
    done
}

@test "template has outputStyle and both merge policies propagate it" {
    [[ "$(jq -r '.outputStyle' "$SETTINGS_TEMPLATE")" != "null" ]]
    grep -q 'outputStyle' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'outputStyle' "$DOTFILES_DIR/setup-windows.ps1"
}

# --- hooks.SessionStart with placeholder ---

@test "template hooks.SessionStart contains __HOOK_COMMAND__ placeholder" {
    [[ "$(jq -r '.hooks.SessionStart[0].hooks[0].command' "$SETTINGS_TEMPLATE")" == "__HOOK_COMMAND__" ]]
}

@test "template hooks.SessionStart timeout = 30" {
    [[ "$(jq -r '.hooks.SessionStart[0].hooks[0].timeout' "$SETTINGS_TEMPLATE")" == "30" ]]
}

@test "template hooks.SessionStart[0].hooks[0].type = command" {
    [[ "$(jq -r '.hooks.SessionStart[0].hooks[0].type' "$SETTINGS_TEMPLATE")" == "command" ]]
}

# --- hooks.SessionEnd with placeholder (MEMORY-001 session bridge) ---

@test "template hooks.SessionEnd contains __SESSION_END_COMMAND__ placeholder" {
    [[ "$(jq -r '.hooks.SessionEnd[0].hooks[0].command' "$SETTINGS_TEMPLATE")" == "__SESSION_END_COMMAND__" ]]
}

@test "setup-linux merge wires hooks.SessionEnd from template" {
    grep -qF '.hooks.SessionEnd = $tmpl.hooks.SessionEnd' setup-linux.sh
}

@test "setup-windows merge wires hooks.SessionEnd from template" {
    grep -qF "\$existing['hooks']['SessionEnd'] = \$template['hooks']['SessionEnd']" setup-windows.ps1
}

# --- permissions.allow: MCP entries required, Bash/WebSearch/WebFetch/Skill allowed ---

@test "template permissions.allow has at least 3 entries" {
    [[ "$(jq '.permissions.allow | length' "$SETTINGS_TEMPLATE")" -ge 3 ]]
}

@test "template permissions.allow has all 3 expected MCP entries" {
    jq -e '.permissions.allow | index("mcp__hive__vault_query")' "$SETTINGS_TEMPLATE"
    jq -e '.permissions.allow | index("mcp__hive__vault_write")' "$SETTINGS_TEMPLATE"
    jq -e '.permissions.allow | index("mcp__sequential-thinking__sequentialthinking")' "$SETTINGS_TEMPLATE"
}

@test "template permissions.allow contains no Read entries (user-owned, machine-specific)" {
    # Negative assertion: no entry starts with "Read(" -- those are absolute paths
    # tied to a specific machine and stay user-owned via the merge policy.
    run jq -e '.permissions.allow | map(startswith("Read(")) | any' "$SETTINGS_TEMPLATE"
    [[ "$status" -ne 0 ]]
}

@test "template permissions.allow entries only use allowed prefixes" {
    # Allowed: mcp__ (MCP servers), Bash( (shell commands), WebSearch,
    # WebFetch(domain:...), Skill( (skills). No Read( paths (user-owned).
    jq -e '.permissions.allow | map(
        startswith("mcp__") or
        startswith("Bash(") or
        . == "WebSearch" or
        startswith("WebFetch(") or
        startswith("Skill(")
    ) | all' "$SETTINGS_TEMPLATE"
}

@test "template does NOT define permissions.additionalDirectories (user-owned)" {
    run jq -e '.permissions.additionalDirectories' "$SETTINGS_TEMPLATE"
    [[ "$status" -ne 0 ]]
}

# --- enabledPlugins: 5 universal plugins, all true (was 13 pre-usage-audit) ---

@test "template enabledPlugins has exactly 5 universal plugins" {
    [[ "$(jq '.enabledPlugins | length' "$SETTINGS_TEMPLATE")" == "5" ]]
}

@test "template enabledPlugins values all set to true" {
    jq -e '.enabledPlugins | to_entries | map(.value == true) | all' "$SETTINGS_TEMPLATE"
}

@test "template enabledPlugins includes core plugins from the existing user setup" {
    # Exactly 5 remain (not a sample) — each has recorded usage: security-guidance
    # and gopls-lsp have real saved findings/diagnostics; the output-style pair
    # drives the current session mode; frontend-design has no substitute yet.
    jq -e '.enabledPlugins["security-guidance@claude-plugins-official"]' "$SETTINGS_TEMPLATE"
    jq -e '.enabledPlugins["gopls-lsp@claude-plugins-official"]' "$SETTINGS_TEMPLATE"
    jq -e '.enabledPlugins["frontend-design@claude-plugins-official"]' "$SETTINGS_TEMPLATE"
    jq -e '.enabledPlugins["explanatory-output-style@claude-plugins-official"]' "$SETTINGS_TEMPLATE"
    jq -e '.enabledPlugins["learning-output-style@claude-plugins-official"]' "$SETTINGS_TEMPLATE"
}

# Inverse assertions (BUG-007, incident → guard pattern from SDD-006):
# these plugins were removed for zero recorded usage across every saved
# session transcript (2026-08-06 audit) or, for `github`, being broken
# (BUG-007). CI MUST fail if any returns to the template (accidental re-add,
# copy-paste from old docs, etc.). The setup scripts' plugin install loops
# are checked below as a cross-OS parity guard.
@test "template enabledPlugins must NOT include plugins removed for zero usage" {
    for plugin in github code-simplifier claude-md-management claude-code-setup \
        ralph-loop code-review commit-commands pr-review-toolkit feature-dev; do
        run jq -e ".enabledPlugins[\"${plugin}@claude-plugins-official\"]" "$SETTINGS_TEMPLATE"
        [ "$status" -ne 0 ]
    done
}

@test "setup-linux.sh plugin install loop must NOT include plugins removed for zero usage" {
    for plugin in github code-simplifier claude-md-management claude-code-setup \
        ralph-loop code-review commit-commands pr-review-toolkit feature-dev; do
        ! grep -qE "\"${plugin}@claude-plugins-official\"" "$DOTFILES_DIR/setup-linux.sh"
    done
}

@test "setup-windows.ps1 plugin install loop must NOT include plugins removed for zero usage" {
    for plugin in github code-simplifier claude-md-management claude-code-setup \
        ralph-loop code-review commit-commands pr-review-toolkit feature-dev; do
        ! grep -qE "\"${plugin}@claude-plugins-official\"" "$DOTFILES_DIR/setup-windows.ps1"
    done
}

# --- Negative assertions: user/machine-specific keys MUST NOT be in template ---

@test "template does NOT define hooks.PreToolUse (third-party tool surface)" {
    run jq -e '.hooks.PreToolUse' "$SETTINGS_TEMPLATE"
    [[ "$status" -ne 0 ]]
}

@test "template does NOT define hooks.PostToolUse (third-party tool surface)" {
    run jq -e '.hooks.PostToolUse' "$SETTINGS_TEMPLATE"
    [[ "$status" -ne 0 ]]
}

@test "template does NOT define hooks.Stop (third-party tool surface)" {
    run jq -e '.hooks.Stop' "$SETTINGS_TEMPLATE"
    [[ "$status" -ne 0 ]]
}
