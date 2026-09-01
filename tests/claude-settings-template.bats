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
        '{"model":"opus","effortLevel":"xhigh","permissions":{"allow":[]},"enabledPlugins":{},"env":{"CLAUDE_CODE_ENABLE_EXPERIMENTAL_ADVISOR_TOOL":"1"}}' \
        "$expr" 2>&1)"
    [ -n "$out" ]
    # template key arrives...
    [ "$(printf '%s' "$out" | jq -r '.env.CLAUDE_CODE_ENABLE_EXPERIMENTAL_ADVISOR_TOOL')" = "1" ]
    # ...and the machine-local one is not clobbered
    [ "$(printf '%s' "$out" | jq -r '.env.MACHINE_LOCAL')" = "keep" ]

    # a target with no env at all must still receive it
    out="$(printf '%s' '{"model":"sonnet"}' | jq --argjson tmpl \
        '{"model":"opus","effortLevel":"xhigh","permissions":{"allow":[]},"enabledPlugins":{},"env":{"A":"1"}}' \
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

@test "template crossSessionInbound is one of the values Claude Code accepts" {
    # The CLI validates against ["accept","hold","refuse"]. An invalid value is
    # not a hard error -- it warns and FALLS BACK to holding every message for
    # manual approval, so a typo here looks exactly like the default and the
    # only symptom is peer messages quietly expiring.
    local inbound
    inbound="$(jq -r '.crossSessionInbound // "unset"' "$SETTINGS_TEMPLATE")"
    case "$inbound" in
        accept|hold|refuse) ;;
        *) echo "crossSessionInbound '$inbound' is not an accepted value" >&2; return 1 ;;
    esac
}

@test "template attribution hides AI attribution in commits and PRs" {
    # Incident -> guard. The standing order is that no git or GitHub artifact
    # carries AI attribution, and Claude Code's DEFAULTS work against it:
    # attribution.commit/.pr default to the standard Co-Authored-By trailer and
    # sessionUrl defaults to true (appending a Claude-Session trailer and a
    # PR-body link). Until this template carried the key, the order was enforced
    # only by an instruction every agent had to remember -- and an instruction
    # that is obeyed 95% of the time still writes the trailer into permanent
    # history. Empty string hides attribution; false drops the session link.
    #
    # Asserted exactly, not merely "present": a non-empty commit/pr string or a
    # true sessionUrl re-enables the thing the order forbids, and would read as
    # configured while doing the opposite.
    [ "$(jq -r '.attribution.commit' "$SETTINGS_TEMPLATE")" = "" ]
    [ "$(jq -r '.attribution.pr' "$SETTINGS_TEMPLATE")" = "" ]
    [ "$(jq -r '.attribution.sessionUrl' "$SETTINGS_TEMPLATE")" = "false" ]
}

@test "template does NOT use the deprecated includeCoAuthoredBy key" {
    # Claude Code's schema marks it "Deprecated: Use attribution instead".
    # Carrying both is how the two drift apart; attribution is the one that
    # also covers the PR body and the session link.
    run jq -e 'has("includeCoAuthoredBy")' "$SETTINGS_TEMPLATE"
    [ "$status" -ne 0 ]
}

@test "template keeps auto-compaction on, with the summary precomputed" {
    # autoCompactEnabled is what stops a long session dying at the context
    # limit; precomputeCompactionEnabled builds the summary in the background
    # BEFORE it is needed and, per the schema, only applies when auto-compact is
    # on -- so the pair is asserted together. Setting the second without the
    # first is a no-op that reads as configured.
    [ "$(jq -r '.autoCompactEnabled' "$SETTINGS_TEMPLATE")" = "true" ]
    [ "$(jq -r '.precomputeCompactionEnabled' "$SETTINGS_TEMPLATE")" = "true" ]
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
        '{"model":"opus","effortLevel":"xhigh","permissions":{"allow":[]},"enabledPlugins":{}}' \
        "$expr" 2>&1)"
    [ -n "$out" ]
    [ "$(printf '%s' "$out" | jq -r '.model')" = "opus" ]
    [ "$(printf '%s' "$out" | jq -r '.userCustom')" = "preserve me" ]

    # and a template WITH it must carry it through
    out="$(printf '%s' "$existing" | jq --argjson tmpl \
        '{"model":"opus","effortLevel":"xhigh","outputStyle":"Concise","permissions":{"allow":[]},"enabledPlugins":{}}' \
        "$expr" 2>&1)"
    [ "$(printf '%s' "$out" | jq -r '.outputStyle')" = "Concise" ]
}

@test "template has outputStyle and both merge policies propagate it" {
    [[ "$(jq -r '.outputStyle' "$SETTINGS_TEMPLATE")" != "null" ]]
    grep -q 'outputStyle' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'outputStyle' "$DOTFILES_DIR/setup-windows.ps1"
}

# --- hooks: NOT this template's, and NOT either merge function's (HARNESS-045) ---

@test "template declares no hooks at all -- dotf harness bind owns them" {
    # Ownership moved wholly to `dotf harness bind`, which merges by marker from
    # harness/manifest.json. Leaving a hooks block here would resurrect the second
    # writer: the template is applied by an ALLOW-LIST merge, so a hooks key here
    # is either dead weight or, the day someone adds it back to the policy, the
    # positional assignment that deleted a live third-party group all over again.
    run jq -e '.hooks' "$SETTINGS_TEMPLATE"
    [ "$status" -ne 0 ]
}

@test "neither merge function writes hooks (single writer: dotf harness bind)" {
    # Scoped to the two merge BODIES, same extraction as the allow-list guard
    # above: a mention elsewhere in either script (the bind call itself, a
    # comment) is not a second writer.
    local linux_merge windows_merge
    linux_merge="$(sed -n '/^merge_claude_settings() {/,/^}/p' "$DOTFILES_DIR/setup-linux.sh")"
    windows_merge="$(sed -n '/^function Merge-ClaudeSettings {/,/^}/p' "$DOTFILES_DIR/setup-windows.ps1")"
    [ -n "$linux_merge" ] || { echo "could not extract merge_claude_settings from setup-linux.sh" >&2; return 1; }
    [ -n "$windows_merge" ] || { echo "could not extract Merge-ClaudeSettings from setup-windows.ps1" >&2; return 1; }

    printf '%s\n' "$linux_merge" | grep -qF '.hooks' \
        && { echo "merge_claude_settings still writes .hooks -- bind is the only writer" >&2; return 1; }
    printf '%s\n' "$windows_merge" | grep -qF "['hooks']" \
        && { echo "Merge-ClaudeSettings still writes hooks -- bind is the only writer" >&2; return 1; }
    return 0
}

@test "both setup scripts pass --repo-root to bind, never inferring it from the cwd" {
    # env.ResolveHarnessRoot walks up from the CWD for a .git, then falls back to
    # ~/.dotfiles. Measured under `env -i`: from a cwd outside any checkout, on a
    # machine with no ~/.dotfiles yet, bind exits 1 having emitted NO hooks --
    # a first run invoked by absolute path. Neither script may rely on that
    # inference; both know their own checkout. Windows is the likelier victim,
    # because a .ps1 does not change the cwd.
    grep -qF -- 'harness bind --repo-root "$CURRENT_DIR"' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF -- 'harness bind --repo-root $DotfilesDir' "$DOTFILES_DIR/setup-windows.ps1"
}

@test "both setup scripts call dotf harness bind behind a capability probe" {
    # The call is what makes AC1 real; the probe is what keeps a stale dotf from
    # silently emitting nothing (lesson 219 -- exit status cannot tell "I refuse"
    # from "I do not understand the question").
    grep -qF 'harness bind' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'harness bind' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF "grep -q '^[[:space:]]*bind[[:space:]]'" "$DOTFILES_DIR/setup-linux.sh"
    grep -qF "'(?m)^\s*bind\s'" "$DOTFILES_DIR/setup-windows.ps1"
}

@test "the linux probe captures before grepping, never pipes into grep -q" {
    # `cmd | grep -q` closes the pipe on first match; under pipefail the SIGPIPE'd
    # producer makes the pipeline exit 141 -- "too old" for a binary that just
    # proved it is current (measured, compile-harness.sh:369).
    refute_grep_fixed 'harness --help 2>/dev/null | grep' "$DOTFILES_DIR/setup-linux.sh"
}

@test "no __HOOK_COMMAND__ placeholder survives anywhere" {
    # The substitution is gone from both scripts; a leftover placeholder would be
    # a literal command string deployed into a hook.
    refute_grep_fixed '__HOOK_COMMAND__' "$SETTINGS_TEMPLATE"
    refute_grep_fixed '__HOOK_COMMAND__' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep_fixed '__HOOK_COMMAND__' "$DOTFILES_DIR/setup-windows.ps1"
    refute_grep_fixed '__SESSION_END_COMMAND__' "$SETTINGS_TEMPLATE"
    refute_grep_fixed '__SESSION_END_COMMAND__' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep_fixed '__SESSION_END_COMMAND__' "$DOTFILES_DIR/setup-windows.ps1"
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
