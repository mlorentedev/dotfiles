#!/usr/bin/env bats
# Copilot's JSON is deployed by `dotf deploy` (ai/deploy.json, AI-039/#1322):
# settings.json and config.json by MERGE — the CLI writes both files itself, so
# a verbatim copy either gets rewritten (config.json, CLI-managed on 1.0.81:
# "User settings belong in settings.json") or wipes the box's own keys
# (settings.json: allowedUrls, effortLevel, contextTier, renderMarkdown).
# Audited against GitHub Copilot CLI 1.0.81 (`copilot help config`) on
# 2026-08-28 on the Windows work box; the documented key set is frozen below
# and re-audited when the pin in packages.json moves. Earlier audit (AI-036,
# #1296, 1.0.80): `defaultModel` was not a key, `$schema` 404'd, and
# `includeCoAuthoredBy` had been left at its default of TRUE.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SETTINGS="$DOTFILES_DIR/ai/copilot/settings.json"
    export CFG="$DOTFILES_DIR/ai/copilot/config.json"
    export MANIFEST="$DOTFILES_DIR/ai/deploy.json"
    # Top-level names only: dotted entries in the help (customAgents.defaultLocalOnly,
    # ide.autoConnect, subagents.agents.<name>) are nested under their first segment.
    export DOCUMENTED_KEYS="allowedUrls autoUpdate banner bannerStyle bashEnv beep beepOnSchedule commandHistoryMaxSize compactPaste companyAnnouncements contextTier continueOnAutoMode copyOnSelect customAgents defaultMode defaultPermissionMode deniedUrls disableAllHooks experimental hooks ide includeCoAuthoredBy inlineImageLiveWindow inlineImages keepAlive logLevel memory model mouse notifications powershellFlags proxyKerberosServicePrincipal proxyUrl renderMarkdown respectGitignore screenReader scrollbar showTipsOnStartup statusLine stayInAutopilot stream streamerMode subagents tabs terminalProgress theme trustedFolders updateTerminalTitle"
}

load 'lib/refute'

@test "settings.json and config.json are JSON objects" {
    [ "$(jq -r 'type' "$SETTINGS")" = "object" ]
    [ "$(jq -r 'type' "$CFG")" = "object" ]
}

@test "every top-level key in settings.json and config.json is one 'copilot help config' documents (1.0.81)" {
    # A key the CLI does not read is dead config that looks alive (the
    # defaultModel and telemetry class): assert membership, not just validity.
    for f in "$SETTINGS" "$CFG"; do
        while IFS= read -r key; do
            case " $DOCUMENTED_KEYS " in
                *" $key "*) ;;
                *) echo "undocumented key '$key' in $f"; return 1 ;;
            esac
        done < <(jq -r 'keys[]' "$f" | tr -d '\r') # MSYS jq emits CRLF; read keeps the \r
    done
}

@test "settings.json turns Co-authored-by trailers off (no AI attribution doctrine)" {
    [ "$(jq -r '.includeCoAuthoredBy' "$SETTINGS")" = "false" ]
}

@test "settings.json pins the model under the key the CLI reads, not defaultModel" {
    [ "$(jq -r '.defaultModel // "absent"' "$SETTINGS")" = "absent" ]
    model="$(jq -r '.model // ""' "$SETTINGS")"
    [ -n "$model" ]
}

@test "settings.json turns autoUpdate off: packages.json owns the version (ADR-036)" {
    [ "$(jq -r '.autoUpdate' "$SETTINGS")" = "false" ]
}

@test "settings.json does not set powershellFlags: tool calls stay -NoProfile (CLI-058 constraint, #1324)" {
    [ "$(jq -r '.powershellFlags // "absent"' "$SETTINGS")" = "absent" ]
}

@test "config.json carries only trustedFolders: the CLI manages the rest, telemetry is undocumented" {
    [ "$(jq -c 'keys' "$CFG")" = '["trustedFolders"]' ]
    [ "$(jq -r '.telemetry // "absent"' "$SETTINGS")" = "absent" ]
}

@test "neither file carries a \$schema URL (the published one 404s)" {
    [ "$(jq -r '.["$schema"] // "absent"' "$SETTINGS")" = "absent" ]
    [ "$(jq -r '.["$schema"] // "absent"' "$CFG")" = "absent" ]
}

@test "ai/deploy.json deploys settings.json and config.json by merge, mcp-config.json by replace, all gated on copilot" {
    [ "$(jq -r '.configs[] | select(.name=="copilot-settings") | "\(.src) \(.dst) \(.strategy) \(.requires)"' "$MANIFEST")" = "ai/copilot/settings.json {HOME}/.copilot/settings.json merge copilot" ]
    [ "$(jq -r '.configs[] | select(.name=="copilot-config") | "\(.src) \(.dst) \(.strategy) \(.requires) \(.paths)"' "$MANIFEST")" = "ai/copilot/config.json {HOME}/.copilot/config.json merge copilot native" ]
    [ "$(jq -r '.configs[] | select(.name=="copilot-mcp") | "\(.src) \(.dst) \(.strategy // "replace") \(.requires)"' "$MANIFEST")" = "ai/copilot/mcp-config.json {HOME}/.copilot/mcp-config.json replace copilot" ]
}

@test "neither setup copies ai/copilot by glob; both copy copilot-instructions.md explicitly (the JSON is dotf deploy's)" {
    # Measured on the invocation shape (the cp / Copy-Item line), not on any
    # mention: the comments name the old form on purpose.
    refute_grep 'cp -rf "\$CURRENT_DIR/ai/copilot/"\*' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'cp -f "$CURRENT_DIR/ai/copilot/copilot-instructions.md"' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep 'Copy-Item "\$copilotSource\\\*"' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF 'Copy-Item "$copilotSource\copilot-instructions.md"' "$DOTFILES_DIR/setup-windows.ps1"
}

@test "copilot is a packages.json catalog tool (npm) and neither setup carries an install block (AI-038, ADR-036)" {
    jq -e '.tools[] | select(.name == "copilot" and .source.type == "npm")' "$DOTFILES_DIR/packages.json" >/dev/null
    refute_grep 'Id = "GitHub\.Copilot"' "$DOTFILES_DIR/setup-windows.ps1"
    refute_grep '(apt|snap|curl)[^\n]*copilot' "$DOTFILES_DIR/setup-linux.sh"
}

# AI-042 (#1334): the trust lists carried two usernames and two home roots and
# matched nothing on any other machine. They carry {HOME} now and `dotf deploy`
# renders them per box, in the separator form each tool has been observed to
# accept (Copilot: the backslash form the CLI itself wrote on Windows; agy: the
# C:/... form it has read there daily). Guarded on every JSON under ai/, not
# on the one file the audit found first.
@test "no JSON under ai/ carries a user or home literal (AI-042, #1334)" {
    # refute_grep takes one file; every JSON under ai/<tool>/ is checked in turn.
    for f in "$DOTFILES_DIR"/ai/*/*.json; do
        refute_grep '/home/[a-z]+/|C:\\\\Users\\\\|C:/Users/' "$f"
    done
}

@test "the Copilot trust list is a {HOME} template rendered by dotf deploy (AI-042)" {
    # AI-043 (#1390) superseded the agy half of this test, and the reason is
    # worth keeping rather than silently deleting.
    #
    # This test used to assert two more things:
    #
    #   1. that ai/agy/settings.json carries `trustedWorkspaces` as a {HOME}
    #      template, and
    #   2. that the agy-settings manifest entry resolves to `replace`.
    #
    # Both had to go, and (2) is the one that matters: it PINNED the destructive
    # behaviour. agy rewrites `trustedWorkspaces` and `permissions.allow` at
    # runtime, so a replace deploy deleted whatever the user had trusted or
    # granted -- measured on this box, `/home/manu/Projects/ts-bridge` gone and
    # the live file byte-identical to the template. A suite that asserts
    # `replace` here is asserting the data loss.
    #
    # (1) is now vacuous by a STRONGER claim: the template ships no
    # `trustedWorkspaces` at all. "These paths must be {HOME} templates rather
    # than hardcoded" was AI-042's requirement for paths we ship; it does not
    # apply to a key we deliberately stopped shipping. `tests/antigravity.bats`
    # asserts the absence and the merge strategy, so neither claim is lost --
    # they moved to where the change lives.
    #
    # Copilot's half is untouched: `trustedFolders` is still ours to ship.
    [ "$(jq -c '.trustedFolders' "$CFG")" = '["{HOME}/Projects","{HOME}/Projects/*","{HOME}/Projects/Workspace","{HOME}/Projects/Workspace/*"]' ]
    [ "$(jq -r '.configs[] | select(.name=="agy-settings") | "\(.src) \(.dst) \(.paths)"' "$MANIFEST")" = "ai/agy/settings.json {HOME}/.gemini/antigravity-cli/settings.json slash" ]
    [ "$(jq -r '.version' "$MANIFEST")" = "3" ]
}

@test "neither setup copies ai/agy/settings.json any more: the manifest entry renders it (AI-042)" {
    # Measured on the invocation line, not on any mention: the comments name the old form.
    refute_grep '^[[:space:]]*deploy_file "\$CURRENT_DIR/ai/agy/settings\.json"' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep '^[[:space:]]*Copy-Item \$agySettingsSrc' "$DOTFILES_DIR/setup-windows.ps1"
}
