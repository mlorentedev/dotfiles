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

# --- permissions.allow: only MCP entries, no Read paths ---

@test "template permissions.allow has exactly 3 entries" {
    [[ "$(jq '.permissions.allow | length' "$SETTINGS_TEMPLATE")" == "3" ]]
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

@test "template permissions.allow entries all start with mcp__ prefix" {
    jq -e '.permissions.allow | map(startswith("mcp__")) | all' "$SETTINGS_TEMPLATE"
}

@test "template does NOT define permissions.additionalDirectories (user-owned)" {
    run jq -e '.permissions.additionalDirectories' "$SETTINGS_TEMPLATE"
    [[ "$status" -ne 0 ]]
}

# --- enabledPlugins: 14 universal plugins, all true ---

@test "template enabledPlugins has exactly 14 universal plugins" {
    [[ "$(jq '.enabledPlugins | length' "$SETTINGS_TEMPLATE")" == "14" ]]
}

@test "template enabledPlugins values all set to true" {
    jq -e '.enabledPlugins | to_entries | map(.value == true) | all' "$SETTINGS_TEMPLATE"
}

@test "template enabledPlugins includes core plugins from the existing user setup" {
    # Sample the 5 most-used: code-review, commit-commands, github, pr-review-toolkit, claude-md-management.
    # Asserting a sample catches accidental removal while keeping the test list maintainable.
    jq -e '.enabledPlugins["code-review@claude-plugins-official"]' "$SETTINGS_TEMPLATE"
    jq -e '.enabledPlugins["commit-commands@claude-plugins-official"]' "$SETTINGS_TEMPLATE"
    jq -e '.enabledPlugins["github@claude-plugins-official"]' "$SETTINGS_TEMPLATE"
    jq -e '.enabledPlugins["pr-review-toolkit@claude-plugins-official"]' "$SETTINGS_TEMPLATE"
    jq -e '.enabledPlugins["claude-md-management@claude-plugins-official"]' "$SETTINGS_TEMPLATE"
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
