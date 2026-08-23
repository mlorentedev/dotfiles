#!/usr/bin/env bats
# Tests for ai/claude/settings.json template (SDD-002)
# The template is the SSOT for the "dotfiles-owned" subset of ~/.claude/settings.json.
# Per-key merge policy is documented in specs/SDD-002-settings-portability/proposal.md.

load 'lib/refute'

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

@test "template has model = opus[1m]" {
    # The `[1m]` suffix selects the 1M-context variant. It matters that the
    # TEMPLATE carries it, not just the deployed file: the merge policy is
    # `template wins` for model, so every setup run resets whatever /model
    # last saved. With a bare "opus" here, a 1M default chosen interactively
    # survives exactly until the next deploy -- silently.
    [[ "$(jq -r '.model' "$SETTINGS_TEMPLATE")" == "opus[1m]" ]]
}

@test "template has effortLevel = xhigh" {
    [[ "$(jq -r '.effortLevel' "$SETTINGS_TEMPLATE")" == "xhigh" ]]
}

@test "every dotfiles-owned top-level key is named in both merge policies" {
    # The merge policy is an ALLOW-LIST in two places -- a jq expression in
    # setup-linux.sh and an if-chain in setup-windows.ps1. A key added to the
    # template and named in neither is a SILENT no-op on every existing
    # installation, reaching only machines bootstrapped from scratch.
    #
    # This test used to enumerate `model effortLevel outputStyle` by hand and
    # claim in a comment that the structured keys were "handled by name in both
    # scripts". That claim was FALSE for `env`, and nothing checked it: the
    # template carried CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS while the deployed
    # settings.json had no `env` key at all -- the same failure outputStyle had
    # already caused once. A hand-written list cannot catch the key nobody
    # thought to add to it, so the list now comes FROM THE TEMPLATE: any new
    # top-level key must be named in both policies or this fails.
    # Scoped to the two merge function BODIES, not to the whole scripts. Both
    # files say `ContainsKey(...)` and `$tmpl.` in unrelated places --
    # setup-windows.ps1 has `$tool.ContainsKey('Version')` in its winget loop --
    # so a whole-file grep would let a template key named `Version` satisfy this
    # guard while Merge-ClaudeSettings ignored it. A guard that passes on the
    # broken thing is the defect this whole test exists to prevent.
    local linux_merge windows_merge key
    linux_merge="$(sed -n '/^merge_claude_settings() {/,/^}/p' "$DOTFILES_DIR/setup-linux.sh")"
    windows_merge="$(sed -n '/^function Merge-ClaudeSettings {/,/^}/p' "$DOTFILES_DIR/setup-windows.ps1")"
    # An empty extract would fail every key below, which is the safe direction,
    # but say so explicitly rather than blaming the first key.
    [ -n "$linux_merge" ] || { echo "could not extract merge_claude_settings from setup-linux.sh" >&2; return 1; }
    [ -n "$windows_merge" ] || { echo "could not extract Merge-ClaudeSettings from setup-windows.ps1" >&2; return 1; }

    while IFS= read -r key; do
        # `$schema` is editor metadata for JSON language servers. It is never
        # deployed, so it is exempt BY NAME -- a stated decision, not a gap.
        if [ "$key" = "\$schema" ]; then continue; fi
        printf '%s\n' "$linux_merge" | grep -qF -- "\$tmpl.$key" \
            || { echo "merge_claude_settings never mentions '$key'" >&2; return 1; }
        printf '%s\n' "$windows_merge" | grep -qF -- "ContainsKey('$key')" \
            || { echo "Merge-ClaudeSettings never mentions '$key'" >&2; return 1; }
    done < <(jq -r 'keys[]' "$SETTINGS_TEMPLATE")
}

@test "the merge carries env through and preserves machine-local entries" {
    # `env` is how a feature flag reaches Claude Code's own process
    # environment: settings.env is merged into process.env at startup, which is
    # what makes CLAUDE_CODE_ENABLE_EXPERIMENTAL_ADVISOR_TOOL able to un-hide
    # /advisor. Asserting the SHIPPED jq expression, not a copy of it.
    local expr existing out
    expr="$(sed -n "/^    merged=\$(jq --argjson tmpl /,/^    ' \"\$target_path\"/p" \
        "$DOTFILES_DIR/setup-linux.sh" \
        | sed -e "1s/.*jq --argjson tmpl \"\$template_substituted\" '//" -e "\$d")"
    [ -n "$expr" ]

    existing='{"model":"sonnet","env":{"MACHINE_LOCAL":"keep"}}'
    out="$(printf '%s' "$existing" | jq --argjson tmpl \
        '{"model":"opus","effortLevel":"xhigh","permissions":{"allow":[]},"hooks":{},"enabledPlugins":{},"env":{"CLAUDE_CODE_ENABLE_EXPERIMENTAL_ADVISOR_TOOL":"1"}}' \
        "$expr" 2>&1)"
    [ -n "$out" ]
    # template key arrives...
    [ "$(printf '%s' "$out" | jq -r '.env.CLAUDE_CODE_ENABLE_EXPERIMENTAL_ADVISOR_TOOL')" = "1" ]
    # ...and the machine-local one is not clobbered
    [ "$(printf '%s' "$out" | jq -r '.env.MACHINE_LOCAL')" = "keep" ]

    # a target with no env at all must still receive it
    out="$(printf '%s' '{"model":"sonnet"}' | jq --argjson tmpl \
        '{"model":"opus","effortLevel":"xhigh","permissions":{"allow":[]},"hooks":{},"enabledPlugins":{},"env":{"A":"1"}}' \
        "$expr" 2>&1)"
    [ "$(printf '%s' "$out" | jq -r '.env.A')" = "1" ]
}

@test "template env enables the advisor tool" {
    # /advisor is hidden unless this flag is set: the gate reads
    # CLAUDE_CODE_ENABLE_EXPERIMENTAL_ADVISOR_TOOL from the process env, and
    # falls back to a server-side flag that is off for this account. Without
    # it, advisorModel is read but the tool never attaches -- silently.
    [ "$(jq -r '.env.CLAUDE_CODE_ENABLE_EXPERIMENTAL_ADVISOR_TOOL' "$SETTINGS_TEMPLATE")" = "1" ]
}

@test "template advisorModel is one of the values Claude Code accepts" {
    # The CLI validates against ["fable","opus","sonnet"] (plus "off"); anything
    # else is rejected at startup with "cannot be used as an advisor".
    local advisor
    advisor="$(jq -r '.advisorModel // "off"' "$SETTINGS_TEMPLATE")"
    case "$advisor" in
        fable|opus|sonnet|off) ;;
        *) echo "advisorModel '$advisor' is not an accepted value" >&2; return 1 ;;
    esac
}

@test "the merge expression survives a template that omits an optional key" {
    # jq: a condition that evaluates to `empty` makes the WHOLE if-expression
    # produce nothing, so `if ($tmpl.key // empty) then ... else . end` does not
    # mean "leave it alone when absent" -- it means the entire merge pipeline
    # yields an empty result. merge_claude_settings then logs "merge produced
    # empty output, skipping write" and NO key deploys: not model, not
    # effortLevel, not permissions, not hooks. `has()` is the guard that means
    # what the other one looks like it means.
    #
    # Extracted from setup-linux.sh so this tests the shipped expression rather
    # than a copy of it.
    local expr existing out
    expr="$(sed -n "/^    merged=\$(jq --argjson tmpl /,/^    ' \"\$target_path\"/p" \
        "$DOTFILES_DIR/setup-linux.sh" \
        | sed -e "1s/.*jq --argjson tmpl \"\$template_substituted\" '//" -e "\$d")"
    [ -n "$expr" ]
    existing='{"model":"sonnet","userCustom":"preserve me"}'

    # a template WITHOUT the optional key must still merge everything else
    out="$(printf '%s' "$existing" | jq --argjson tmpl \
        '{"model":"opus","effortLevel":"xhigh","permissions":{"allow":[]},"hooks":{},"enabledPlugins":{}}' \
        "$expr" 2>&1)"
    [ -n "$out" ]
    [ "$(printf '%s' "$out" | jq -r '.model')" = "opus" ]
    [ "$(printf '%s' "$out" | jq -r '.userCustom')" = "preserve me" ]

    # and a template WITH it must carry it through
    out="$(printf '%s' "$existing" | jq --argjson tmpl \
        '{"model":"opus","effortLevel":"xhigh","outputStyle":"Concise","permissions":{"allow":[]},"hooks":{},"enabledPlugins":{}}' \
        "$expr" 2>&1)"
    [ "$(printf '%s' "$out" | jq -r '.outputStyle')" = "Concise" ]
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
        refute_grep "\"${plugin}@claude-plugins-official\"" "$DOTFILES_DIR/setup-linux.sh"
    done
}

@test "setup-windows.ps1 plugin install loop must NOT include plugins removed for zero usage" {
    for plugin in github code-simplifier claude-md-management claude-code-setup \
        ralph-loop code-review commit-commands pr-review-toolkit feature-dev; do
        refute_grep "\"${plugin}@claude-plugins-official\"" "$DOTFILES_DIR/setup-windows.ps1"
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
