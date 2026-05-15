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

log() { printf '[claude-mem-heal] %s\n' "$1"; }
verbose() { if [ "$VERBOSE" -eq 1 ]; then log "$1"; fi; }

# Replace a broken .mcp.json with the v10.6.3 form. Idempotent: only
# rewrites if the file contains the offending ${_R%/} pattern.
heal_mcp_json() {
    target="$1"
    [ -f "$target" ] || { verbose "no .mcp.json at $target"; return 0; }

    # shellcheck disable=SC2016 # literal pattern, intentionally not expanded
    if ! grep -qF '${_R%/}' "$target" 2>/dev/null; then
        verbose ".mcp.json already healthy: $target"
        return 0
    fi

    cat > "$target" <<'EOF'
{
  "mcpServers": {
    "mcp-search": {
      "type": "stdio",
      "command": "node",
      "args": [
        "${CLAUDE_PLUGIN_ROOT}/scripts/mcp-server.cjs"
      ]
    }
  }
}
EOF
    log "patched .mcp.json: $target"
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

heal_dir() {
    dir="$1"
    [ -d "$dir" ] || return 0
    heal_mcp_json "$dir/.mcp.json"
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

heal_dir "$MARKETPLACE_DIR"

exit 0
