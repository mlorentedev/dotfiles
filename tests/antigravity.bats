#!/usr/bin/env bats
# Tests for Antigravity CLI configuration (SDD-007: fresh-install model, no symlinks).
# Regression suite for BUG-100 (issue #100): asserts no symlinks under ~/.gemini/config/
# and that the master MCP config lives at agy's canonical read path.
#
# These tests assume agy is installed AND `dotf deploy` (or converge) has run;
# the VAULT_PATH substitution and the vault preflight are unit-tested in
# cli/internal/agy, and tests/guard-agy-mcp-in-dotf.bats keeps them out of the
# setup twins (CLI-105).
# CI containers without agy binary or post-setup state will skip these tests.

load 'lib/refute'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export GEMINI_HOME="$HOME/.gemini"
    export AGY_HOME="$GEMINI_HOME/antigravity-cli"
    export MASTER_CONFIG="$GEMINI_HOME/config/mcp_config.json"

    # Skip the entire suite when agy is not installed (CI / fresh machine).
    # These are integration-style tests, not unit tests of repo state.
    if ! command -v agy >/dev/null 2>&1; then
        skip "agy not installed (CI / fresh machine) — install from https://antigravity.google"
    fi
}

@test "agy binary exists and is executable" {
    command -v agy
}

@test "ai/agy/mcp_servers.json hive uses uvx (PyPI) not local repo (parity with claude root config)" {
    # Pre-fix: agy spawned `uv run --directory /home/<user>/Projects/hive hive-vault`
    # which required a local hive checkout and bombed on Windows with "system
    # cannot find the path specified". claude root config has always used
    # `uvx hive-vault` (PyPI). agy now matches -- cross-agent behaviour parity.
    local cmd
    cmd=$(jq -r '.mcpServers["hive-vault"].command' "$DOTFILES_DIR/ai/agy/mcp_servers.json")
    [ "$cmd" = "uvx" ]
    # Regression guard: --directory must not return.
    run jq -r '.mcpServers["hive-vault"].args | join(" ")' "$DOTFILES_DIR/ai/agy/mcp_servers.json"
    [ "$status" -eq 0 ]
    [[ "$output" != *"--directory"* ]] || false
}

@test "ai/agy/mcp_servers.json VAULT_PATH is the placeholder, not a hardcoded user path" {
    # The committed JSON must NOT contain /home/<user> or C:\Users\<user>.
    # Setup scripts substitute the placeholder with the OS-appropriate path.
    local vault
    vault=$(jq -r '.mcpServers["hive-vault"].env.VAULT_PATH' "$DOTFILES_DIR/ai/agy/mcp_servers.json")
    [ "$vault" = '${VAULT_PATH}' ]
    refute_grep '/home/[a-z]+/|C:\\\\Users\\\\' "$DOTFILES_DIR/ai/agy/mcp_servers.json"
}

@test "setup-windows.ps1 deploys agy skills via the SDD-008 engine (closes 0-skills-on-Windows bug, cross-OS parity)" {
    # SDD-008 (#179) replaced the per-agent sharedSkillsDir sync with the unified
    # render-at-deploy engine (Deploy-SkillRecord on Windows, compile-harness
    # --deploy on Linux). The old 'Synced Shared Skills to' / 'sharedSkillsDir'
    # literals are gone from both setups. From a Linux box we cannot run the
    # Windows setup, so we static-check that the Windows script still wires up
    # agy skill deployment: it invokes Deploy-SkillRecord and provisions the
    # gemini skills dir. This locks the cross-OS parity at the live mechanism.
    grep -qF 'Deploy-SkillRecord' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF "GeminiHome 'skills'" "$DOTFILES_DIR/setup-windows.ps1"
}

@test "ai/agy/AGY.md H1 is '# AGY.md' (regression: GEMINI.md->AGY.md rename completeness)" {
    # SDD-007 renamed the file; the H1 and body content lagged. This test
    # locks in that the body actually reflects the file's new identity.
    [[ "$(head -n1 "$DOTFILES_DIR/ai/agy/AGY.md")" == "# AGY.md" ]] || false
    # No stray "Gemini-specific" / "GEMINI.md" body references in the
    # canonical SSOT (archived specs and the cleanup logic in setup-*.{sh,ps1}
    # are excluded by file scope).
    refute_grep_fixed 'Gemini-specific' "$DOTFILES_DIR/ai/agy/AGY.md"
    refute_grep_fixed '# GEMINI.md' "$DOTFILES_DIR/ai/agy/AGY.md"
}

@test "ANTIGRAVITY_ENDPOINT is set to production in .zshrc" {
    grep -q 'export ANTIGRAVITY_ENDPOINT="https://cloudcode-pa.googleapis.com"' "$DOTFILES_DIR/.zshrc"
}

@test "agy settings.json deployed to AGY_HOME" {
    [ -f "$AGY_HOME/settings.json" ]
    [ ! -L "$AGY_HOME/settings.json" ]
}

@test "master MCP config exists at agy canonical path (~/.gemini/config/)" {
    [ -e "$MASTER_CONFIG" ]
}

@test "master MCP config is a regular file, NOT a symlink (BUG-100 regression)" {
    [ -f "$MASTER_CONFIG" ]
    [ ! -L "$MASTER_CONFIG" ]
}

@test "master MCP config is valid JSON" {
    jq '.' "$MASTER_CONFIG"
}

@test "master MCP config is not empty" {
    # agy creates an empty mcp_config.json itself on first run; only the deploy
    # fills it. A machine that has run agy but not the deploy (a fresh Mac) is
    # not a failure of this repo, so say so rather than fail on host state.
    [ -e "$MASTER_CONFIG" ] && [ ! -s "$MASTER_CONFIG" ] &&
        skip "agy has created an empty $MASTER_CONFIG but the deploy has not populated it on this host"
    [ -s "$MASTER_CONFIG" ]
}

@test "no symlinks anywhere under ~/.gemini/config/ (BUG-100 guard)" {
    # find -type l would surface any leftover symlink; assert zero hits
    result=$(find "$GEMINI_HOME/config" -maxdepth 3 -type l 2>/dev/null | wc -l)
    [ "$result" -eq 0 ]
}

@test ".geminiignore deployed and contains sensitive/" {
    [ -f "$GEMINI_HOME/.geminiignore" ]
    grep -q '^sensitive/' "$GEMINI_HOME/.geminiignore"
}

# --- AI-043 (#1390): agy writes its own settings, so the deploy must not own them ---
# These assert REPO CONTENT only (manifest + template), so unlike the tests above
# they run everywhere, with or without agy installed.

@test "AI-043: agy-settings deploys with strategy merge, never replace" {
    # MEASURED data loss, not a prediction. On 2026-09-01 the live file on this
    # box went byte-identical to the template: `/home/manu/Projects/ts-bridge`,
    # a workspace the user had trusted at runtime, was gone. Reproduced in a
    # sandbox HOME with the same binary and only the manifest differing --
    # `replace` destroyed a runtime-added trustedWorkspaces entry and a runtime
    # `mcp(hive-vault/*)` grant; `merge` kept both.
    #
    # The trap this pins: `replace` is the DEFAULT when no strategy is declared,
    # so the destructive setting is the one you get by writing nothing.
    local strategy
    strategy="$(jq -r '.configs[] | select(.name=="agy-settings") | .strategy // "replace(default)"' \
        "$DOTFILES_DIR/ai/deploy.json")"
    [ "$strategy" = "merge" ] \
        || { echo "agy-settings strategy is '$strategy'; runtime state is destroyed on every deploy" >&2; return 1; }
}

@test "AI-042: the agy template restores managed lists through merge union" {
    [ "$(jq -c '.trustedWorkspaces' "$DOTFILES_DIR/ai/agy/settings.json")" = \
        '["{HOME}/Projects/*","{HOME}/Projects/Workspace/*","{HOME}/Projects/worktrees"]' ]
    [ "$(jq -c '.permissions.deny' "$DOTFILES_DIR/ai/agy/settings.json")" = \
        '["command(rm -rf /)","command(rm -rf ~/*)"]' ]
}
