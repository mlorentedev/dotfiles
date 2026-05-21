#!/bin/sh
# claude-mem-heal.sh: idempotently repair the thedotmack/claude-mem plugin
# cache when the published artifact ships broken (v12.7.4, v13.0.0).
#
# Two known issues this script papers over:
#
#   1. .mcp.json embeds shell parameter expansion (${_R%/}) that Claude
#      Code's MCP loader misreads as an env-var name. See upstream
#      issue #2385. We rewrite the file to the simpler v10.6.3 form.
#
#   2. v13.0.0 declares "zod": "^4.3.6" in package.json but the published
#      bun.lock and node_modules/ omit it, so worker-service.cjs crashes
#      with "Cannot find module 'zod/v3'". We install zod in place.
#
# Both upstream errors revert on /plugin update, so this runs from
# claude-session-start.sh to self-heal at session start.
#
# Behaviour: silent on healthy installs (exit 0, no output). Prints one
# line per heal action taken so the SessionStart hook can surface it.
# Always exits 0 — never blocks session start on transient failures.
#
# Usage:
#   claude-mem-heal.sh           # silent unless something was healed
#   claude-mem-heal.sh --verbose # always log what was checked

set -eu

VERBOSE=0
[ "${1:-}" = "--verbose" ] && VERBOSE=1

CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
CACHE_ROOT="$CLAUDE_DIR/plugins/cache/thedotmack/claude-mem"
MARKETPLACE_DIR="$CLAUDE_DIR/plugins/marketplaces/thedotmack/plugin"
MARKETPLACE_DIR_ACTUAL="$CLAUDE_DIR/plugins/marketplaces/thedotmack-claude-mem/plugin"

log() { printf '[claude-mem-heal] %s\n' "$1"; }
verbose() { if [ "$VERBOSE" -eq 1 ]; then log "$1"; fi; }

# BUG-012: Claude Code clones the claude-mem marketplace under the GitHub repo
# name `thedotmack-claude-mem/`, but the plugin's bundled hooks.json hardcodes
# the legacy fallback `marketplaces/thedotmack/plugin/scripts/...`. Without a
# compatibility symlink, plugin hooks fail discovery when CLAUDE_PLUGIN_ROOT
# is unset (UserPromptSubmit blocked). Create the legacy path as a symlink to
# the actual install. Idempotent: only acts when source dir exists and target
# path is absent.
ensure_marketplace_compat_symlink() {
    legacy="$CLAUDE_DIR/plugins/marketplaces/thedotmack"
    actual="$CLAUDE_DIR/plugins/marketplaces/thedotmack-claude-mem"
    if [ ! -d "$actual" ]; then
        verbose "no thedotmack-claude-mem marketplace at $actual"
        return 0
    fi
    if [ -e "$legacy" ] || [ -L "$legacy" ]; then
        verbose "legacy marketplace path already present: $legacy"
        return 0
    fi
    if ln -s "$actual" "$legacy" 2>/dev/null; then
        log "created legacy marketplace symlink: $legacy -> thedotmack-claude-mem"
    else
        log "ERROR: failed to create $legacy symlink"
    fi
}

# Replace a broken .mcp.json with a healthy form. BUG-016 (2026-05-21):
# extended to detect v13.x cascading-printf pattern alongside v12.7.4's
# `${_R%/}` literal. v13.x triggers the upstream EPIPE race documented in
# thedotmack/claude-mem#2607 (causes `/mcp ... -32000` failures intermittently).
#
# The healthy form mirrors the v13.x cascade structure (so it works whether
# or not Claude Code sets CLAUDE_PLUGIN_ROOT) but pipes the consumer's
# matches through `head -n1` instead of breaking the inner `while` loop --
# this drains the entire producer pipe, eliminating the EPIPE writes that
# trigger the upstream bug.
#
# Idempotent: skips when neither v12.7.4 nor v13.x signature present.
heal_mcp_json() {
    target="$1"
    [ -f "$target" ] || { verbose "no .mcp.json at $target"; return 0; }

    # shellcheck disable=SC2016 # literal pattern, intentionally not expanded
    has_v12=$(grep -cF '${_R%/}' "$target" 2>/dev/null || echo 0)
    has_v13=$(grep -cE '"sh".*"-c"|while IFS= read' "$target" 2>/dev/null || echo 0)
    if [ "$has_v12" -eq 0 ] && [ "$has_v13" -eq 0 ]; then
        verbose ".mcp.json already healthy: $target"
        return 0
    fi

    cat > "$target" <<'EOF'
{
  "mcpServers": {
    "mcp-search": {
      "type": "stdio",
      "command": "sh",
      "args": [
        "-c",
        "_C=\"${CLAUDE_CONFIG_DIR:-$HOME/.claude}\"; _E=\"${CLAUDE_PLUGIN_ROOT:-${PLUGIN_ROOT:-}}\"; _P=$({ [ -n \"$_E\" ] && printf '%s\\n' \"$_E\"; ls -dt \"$_C/plugins/cache/thedotmack/claude-mem\"/[0-9]*/ 2>/dev/null; printf '%s\\n' \"$_C/plugins/marketplaces/thedotmack-claude-mem/plugin\" \"$_C/plugins/marketplaces/thedotmack/plugin\"; } | while IFS= read -r _R; do _R=\"${_R%/}\"; [ -d \"$_R/plugin/scripts\" ] && _Q=\"$_R/plugin\" || _Q=\"$_R\"; [ -f \"$_Q/scripts/mcp-server.cjs\" ] && printf '%s\\n' \"$_Q\"; done | head -n1); [ -n \"$_P\" ] || { echo 'claude-mem: mcp server not found' >&2; exit 1; }; exec node \"$_P/scripts/mcp-server.cjs\""
      ]
    }
  }
}
EOF
    if [ "$has_v13" -gt 0 ]; then
        log "patched .mcp.json (v13.x cascade -> head -n1 race-free form): $target"
    else
        log "patched .mcp.json (v12.7.4 \${_R%/} -> head -n1 race-free form): $target"
    fi
}

# Install the zod runtime dep if package.json declares it but it isn't
# installed. Idempotent: skips when node_modules/zod exists.
heal_zod() {
    plugin_dir="$1"
    pkg="$plugin_dir/package.json"
    [ -f "$pkg" ] || { verbose "no package.json at $plugin_dir"; return 0; }

    # Only act if package.json declares zod and node_modules/zod is missing.
    grep -q '"zod"' "$pkg" 2>/dev/null || { verbose "no zod dep in $pkg"; return 0; }
    [ -d "$plugin_dir/node_modules/zod" ] && { verbose "zod present in $plugin_dir"; return 0; }

    if ! command -v npm >/dev/null 2>&1; then
        log "ERROR: zod missing in $plugin_dir but npm not on PATH — skipping"
        return 0
    fi

    # Use a subshell so we don't leak the cd. Suppress output unless verbose.
    # --ignore-scripts is mandatory: other deps in the plugin (e.g.
    # tree-sitter) trigger node-gyp on postinstall and fail on machines
    # without a C++ build toolchain (Windows MSBuild, Linux build-essential).
    # We only need zod's pure-JS files; no postinstall is required for it.
    if [ "$VERBOSE" -eq 1 ]; then
        ( cd "$plugin_dir" && npm install --no-save --no-package-lock --no-audit --no-fund --ignore-scripts zod@^4.3.6 ) || {
            log "ERROR: npm install zod failed in $plugin_dir"
            return 0
        }
    else
        ( cd "$plugin_dir" && npm install --no-save --no-package-lock --no-audit --no-fund --ignore-scripts --silent zod@^4.3.6 >/dev/null 2>&1 ) || {
            log "ERROR: npm install zod failed in $plugin_dir"
            return 0
        }
    fi
    log "installed missing zod dep in $plugin_dir"
}

# BUG-017 (2026-05-21): patch hooks.json against the same EPIPE race that
# BUG-016 closed for .mcp.json. The 6 upstream hooks (Setup, SessionStart x2,
# UserPromptSubmit, PostToolUse, PreToolUse, Stop) all use the
# `{ printf; ls; printf; } | while ... break` pipe cascade. When the consumer
# breaks early, unconsumed producer writes EPIPE on Git Bash Windows.
# Minimal substitution: `break; }; done` -> `}; done | head -n1` keeps the
# loop running to completion, then head takes the first printed match.
#
# BUG-018 (2026-05-21): after BUG-017 closed the EPIPE race, the
# UserPromptSubmit hook STILL blocked because its command terminates with
# `node ... hook claude-code session-init` and does NOT emit Claude Code's
# `{"continue":true,"suppressOutput":true}` directive. bun-runner.js's
# empty-stdin diagnostic (upstream claude-mem#2188) goes to stdout, Claude
# Code reads it as non-continue, blocks the prompt. Heal appends the
# directive in the same pass.
heal_hooks_json() {
    target="$1"
    [ -f "$target" ] || { verbose "no hooks.json at $target"; return 0; }
    # Detect either BUG-017 signature (break; }; done) or BUG-018 signature
    # (UserPromptSubmit terminator without the continue directive).
    has_017=$(grep -cF 'break; }; done' "$target" 2>/dev/null || echo 0)
    has_018=$(grep -cE 'session-init"$|session-init"[^}]*$' "$target" 2>/dev/null | head -1)
    has_018=${has_018:-0}
    # Simpler: also check explicitly for the un-patched session-init terminator.
    if ! grep -qF 'break; }; done' "$target" && \
       ! grep -qF 'session-init"' "$target"; then
        verbose "hooks.json already healthy: $target"
        return 0
    fi
    # Use `#` as sed delimiter -- `|` appears literally in the replacement
    # (`done | head -n1`) and would confuse `s|...|...|g`.
    #
    # BUG-018 substitution is GENERIC across all 5 `hook claude-code <X>"`
    # terminators (UserPromptSubmit/session-init, SessionStart/context,
    # PostToolUse/observation, PreToolUse/file-context, Stop/summarize).
    # The user empirically hit UserPromptSubmit first (BUG-018 narrow scope)
    # then Stop minutes later (originally deferred as BUG-018b -- now folded
    # in via regex capture). Setup hook (`version-check.js`) is left untouched:
    # it only fires on plugin install/update, not user hot path.
    tmp="$target.tmp.$$"
    sed -e 's#break; }; done#}; done | head -n1#g' \
        -e 's#hook claude-code \([a-z][a-z-]*\)"#hook claude-code \1 2>/dev/null; echo '\''{\\"continue\\":true,\\"suppressOutput\\":true}'\''"#g' \
        "$target" > "$tmp" && mv "$tmp" "$target"
    log "patched hooks.json (BUG-017 head-n1 + BUG-018 continue-directive on all 5 hooks): $target"
}

heal_dir() {
    dir="$1"
    [ -d "$dir" ] || return 0
    heal_mcp_json "$dir/.mcp.json"
    heal_hooks_json "$dir/hooks/hooks.json"
    heal_hooks_json "$dir/plugin/hooks/hooks.json"
    heal_zod "$dir"
}

# Heal every cached version (so a rolled-back /plugin doesn't surprise us)
# plus the marketplace copy used as fallback by the discovery logic.
if [ -d "$CACHE_ROOT" ]; then
    for d in "$CACHE_ROOT"/*/; do
        heal_dir "${d%/}"
    done
else
    verbose "no cache dir at $CACHE_ROOT"
fi

ensure_marketplace_compat_symlink
heal_dir "$MARKETPLACE_DIR"
heal_dir "$MARKETPLACE_DIR_ACTUAL"

exit 0
