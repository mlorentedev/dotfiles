#!/usr/bin/env bats
# Tests for ai/claude/settings.json template (SDD-002)
# The template is the SSOT for the "dotfiles-owned" subset of ~/.claude/settings.json.
# It is merged by the claude-settings entry of ai/deploy.json; the per-key policy is pinned by cli/internal/deploy/claude_settings_test.go.

load 'lib/refute'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SETTINGS_TEMPLATE="$DOTFILES_DIR/ai/claude/settings.json"
}

@test "template file exists" {
    [[ -f "$SETTINGS_TEMPLATE" ]] || false
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
    [[ "$(jq -r '.model' "$SETTINGS_TEMPLATE")" == "opus[1m]" ]] || false
}

@test "template has effortLevel = xhigh" {
    [[ "$(jq -r '.effortLevel' "$SETTINGS_TEMPLATE")" == "xhigh" ]] || false
}

@test "neither setup script merges settings.json -- dotf deploy is its one writer" {
    # The twins merged this template with jq and PowerShell next to the
    # claude-settings entry of ai/deploy.json, so every setup wrote the file twice
    # under two policies (#2000). The Go merge and its per-key policy are pinned by
    # cli/internal/deploy/claude_settings_test.go. A twin coming back is a second
    # writer whose policy nothing tests.
    local f
    for f in setup-linux.sh setup-windows.ps1; do
        refute_grep 'merge_claude_settings|Merge-ClaudeSettings|ai[/\\]claude[/\\]settings\.json' "$DOTFILES_DIR/$f" || return 1
    done
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

@test "template autoCompactWindow actually caps the model it is paired with" {
    # The window is in TOKENS, and the effective threshold is the MINIMUM of
    # this setting and the model's own window. So the value only does anything
    # if it is below the model's window -- pair it with a smaller model and it
    # silently becomes a no-op that still reads as configured.
    #
    # Asserted as a RELATIONSHIP, not a constant: the failure this guards is
    # someone dropping the `[1m]` suffix from `model` and leaving a 1M-sized
    # window behind, which then caps nothing. The documented range is 100000 to
    # 1000000 (code.claude.com/docs/en/settings-reference#autocompactwindow); a
    # value outside it is not what the file says it is.
    #
    # The value itself (300000, 2026-09-24) is an owner choice. It is just above
    # the 256K bucket where Anthropic's published long-context scores are near
    # ceiling and well below the 1M bucket where they drop, and CLAUDE.md is
    # re-injected from disk at every compaction. The evidence is in the PR that set it.
    local win model
    win="$(jq -r '.autoCompactWindow' "$SETTINGS_TEMPLATE")"
    model="$(jq -r '.model' "$SETTINGS_TEMPLATE")"
    case "$win" in
        ''|*[!0-9]*) echo "autoCompactWindow '$win' is not a positive integer" >&2; return 1 ;;
    esac
    [ "$win" -ge 100000 ] && [ "$win" -le 1000000 ] \
        || { echo "autoCompactWindow=$win is outside the documented 100000-1000000 range" >&2; return 1; }
    case "$model" in
        *'[1m]')
            [ "$win" -lt 1000000 ] \
                || { echo "model is $model (1M window) but autoCompactWindow=$win caps nothing" >&2; return 1; } ;;
    esac
}

@test "template retains transcripts for a bounded, explicit period" {
    # Default is 30 days. The value is a deliberate tradeoff and is asserted so
    # it cannot drift silently in either direction: transcripts are the fallback
    # continuity record when a handoff is thin, and they are also a durable
    # artifact that nothing scans for secrets. Bounded, not unbounded.
    local days
    days="$(jq -r '.cleanupPeriodDays' "$SETTINGS_TEMPLATE")"
    case "$days" in
        ''|*[!0-9]*) echo "cleanupPeriodDays '$days' is not a positive integer" >&2; return 1 ;;
    esac
    [ "$days" -ge 1 ]
}

@test "template sets language to spanish, which also sets the voice dictation language" {
    # Claude Code has no separate key for the dictation language: `language` is
    # described as "Preferred language for Claude responses and voice
    # dictation", and with it unset /voice dictates in English. The UI stays in
    # English either way. Git and GitHub artifacts stay English because
    # AGENTS.md says so, not because of this key.
    [[ "$(jq -r '.language' "$SETTINGS_TEMPLATE")" == "spanish" ]] || false
}

@test "template has outputStyle" {
    [[ "$(jq -r '.outputStyle' "$SETTINGS_TEMPLATE")" != "null" ]] || false
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
    [[ "$(jq '.permissions.allow | length' "$SETTINGS_TEMPLATE")" -ge 3 ]] || false
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
    [[ "$status" -ne 0 ]] || false
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
    [[ "$status" -ne 0 ]] || false
}

# --- enabledPlugins: 5 universal plugins, all true (was 13 pre-usage-audit) ---

@test "template enabledPlugins has exactly 5 universal plugins" {
    [[ "$(jq '.enabledPlugins | length' "$SETTINGS_TEMPLATE")" == "5" ]] || false
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
# copy-paste from old docs, etc.). ai/claude/plugins.json, the list
# `dotf deploy` installs on every OS, is checked below.
@test "template enabledPlugins must NOT include plugins removed for zero usage" {
    for plugin in github code-simplifier claude-md-management claude-code-setup \
        ralph-loop code-review commit-commands pr-review-toolkit feature-dev; do
        run jq -e ".enabledPlugins[\"${plugin}@claude-plugins-official\"]" "$SETTINGS_TEMPLATE"
        [ "$status" -ne 0 ]
    done
}

@test "plugins.json must NOT install plugins removed for zero usage" {
    for plugin in github code-simplifier claude-md-management claude-code-setup \
        ralph-loop code-review commit-commands pr-review-toolkit feature-dev; do
        run jq -e --arg id "${plugin}@claude-plugins-official" '.plugins | index($id)' "$DOTFILES_DIR/ai/claude/plugins.json"
        [ "$status" -ne 0 ]
    done
}

