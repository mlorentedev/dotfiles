#!/usr/bin/env bats
# Tests for scripts/claude-mem-heal.sh (TEST-001 / #128)
#
# claude-mem-heal.sh idempotently repairs the thedotmack/claude-mem plugin
# cache. It is sourceable: with CLAUDE_CONFIG_DIR pointed at an empty temp
# dir the top-level heal loop is a no-op, so we can source the file and
# exercise its pure-ish heal_* functions against fixtures -- no network, no
# real ~/.claude mutation.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export HEAL_SCRIPT="$SCRIPTS_DIR/claude-mem-heal.sh"
    # Sandbox: isolate every run from the real ~/.claude.
    TMP="$(mktemp -d)"
    export CLAUDE_CONFIG_DIR="$TMP/claude"
    mkdir -p "$CLAUDE_CONFIG_DIR"
}

teardown() {
    [ -n "${TMP:-}" ] && rm -rf "$TMP"
}

# --- Syntax (1 & 2) ---

@test "claude-mem-heal.sh valid bash syntax" {
    bash -n "$HEAL_SCRIPT"
}

@test "claude-mem-heal.sh valid zsh syntax" {
    if command -v zsh >/dev/null 2>&1; then
        zsh -n "$HEAL_SCRIPT"
    else
        skip "zsh not available"
    fi
}

# --- Usage / invocation behavior (3) ---

@test "claude-mem-heal.sh runs silently and exits 0 on a clean (empty) install" {
    # No cache dir, no marketplace dirs => nothing to heal => silent, exit 0.
    run "$HEAL_SCRIPT"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "claude-mem-heal.sh --verbose logs what it checked and exits 0" {
    run "$HEAL_SCRIPT" --verbose
    [ "$status" -eq 0 ]
    [[ "$output" == *"[claude-mem-heal]"* ]]
    [[ "$output" == *"no cache dir"* ]]
}

@test "claude-mem-heal.sh documents both invocation forms in header" {
    grep -qF 'claude-mem-heal.sh           # silent unless something was healed' "$HEAL_SCRIPT"
    grep -qF 'claude-mem-heal.sh --verbose # always log what was checked' "$HEAL_SCRIPT"
}

# --- Helper: source ONLY the function definitions, not the top-level heal
# loop. The script ends in a bare `exit 0`, so sourcing it whole would kill
# the bats test process; instead we materialize a function-only copy by
# truncating at the top-level execution marker, then source that. The copy
# is byte-identical to the original up to the cut, so the functions under
# test are exactly the shipped ones.
_source_heal() {
    funcs="$TMP/heal-funcs.sh"
    awk '/^# Heal every cached version/ { exit } { print }' \
        "$HEAL_SCRIPT" > "$funcs"
    # shellcheck disable=SC1090
    . "$funcs"
}

# --- heal_mcp_json: idempotency + both broken-signature heals (4) ---

@test "heal_mcp_json is a no-op on a missing target" {
    _source_heal
    run heal_mcp_json "$TMP/does-not-exist.json"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "heal_mcp_json converges: re-healing produces byte-identical output" {
    # NOTE: heal_mcp_json is NOT skip-idempotent -- its own canonical output
    # contains the v13 'sh'/'-c' + 'while IFS= read' signatures, so it rewrites
    # on every pass. The property that actually holds (and matters) is
    # CONVERGENCE: a second heal yields exactly the same bytes as the first.
    _source_heal
    f="$TMP/converge.mcp.json"
    printf '%s\n' '{ "args": ["${_R%/}"] }' > "$f"
    heal_mcp_json "$f"
    first="$(cat "$f")"
    heal_mcp_json "$f"
    second="$(cat "$f")"
    [ "$first" = "$second" ]
    grep -qF 'head -n1' "$f"
}

@test "heal_mcp_json rewrites a v12.7.4 \${_R%/} broken file and logs the patch" {
    _source_heal
    broken="$TMP/v12.mcp.json"
    # The literal ${_R%/} expansion is the v12.7.4 breakage signature.
    before='{ "args": ["${_R%/}"] }'
    printf '%s\n' "$before" > "$broken"
    run heal_mcp_json "$broken"
    [ "$status" -eq 0 ]
    [[ "$output" == *"patched .mcp.json"* ]]
    [[ "$output" == *"v12.7.4"* ]]
    # The file was rewritten to the canonical race-free form: it changed, it
    # carries the head -n1 discovery, and it is now a full mcpServers block.
    [ "$(cat "$broken")" != "$before" ]
    grep -qF 'head -n1' "$broken"
    grep -qF '"mcpServers"' "$broken"
}

@test "heal_mcp_json rewrites a v13.x cascade file and logs the v13 patch" {
    _source_heal
    broken="$TMP/v13.mcp.json"
    printf '%s\n' '{ "command": "sh", "args": ["-c", "while IFS= read"] }' > "$broken"
    run heal_mcp_json "$broken"
    [ "$status" -eq 0 ]
    [[ "$output" == *"patched .mcp.json"* ]]
    [[ "$output" == *"v13.x"* ]]
    grep -qF 'head -n1' "$broken"
}

# --- heal_zod: guards before any npm side effect (4) ---

@test "heal_zod is a no-op when package.json is absent" {
    _source_heal
    run heal_zod "$TMP/empty-plugin"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "heal_zod is a no-op when package.json declares no zod dep" {
    _source_heal
    pdir="$TMP/nozod"
    mkdir -p "$pdir"
    printf '%s\n' '{ "dependencies": { "left-pad": "^1.0.0" } }' > "$pdir/package.json"
    run heal_zod "$pdir"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "heal_zod is a no-op when node_modules/zod already present (no npm call)" {
    _source_heal
    pdir="$TMP/haszod"
    mkdir -p "$pdir/node_modules/zod"
    printf '%s\n' '{ "dependencies": { "zod": "^4.3.6" } }' > "$pdir/package.json"
    run heal_zod "$pdir"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

# --- ensure_marketplace_compat_symlink: BUG-012 symlink seam (4) ---

@test "ensure_marketplace_compat_symlink no-ops when actual marketplace is absent" {
    _source_heal
    run ensure_marketplace_compat_symlink
    [ "$status" -eq 0 ]
    [ -z "$output" ]
    [ ! -e "$CLAUDE_CONFIG_DIR/plugins/marketplaces/thedotmack" ]
}

@test "ensure_marketplace_compat_symlink creates the legacy symlink when actual exists" {
    _source_heal
    actual="$CLAUDE_CONFIG_DIR/plugins/marketplaces/thedotmack-claude-mem"
    mkdir -p "$actual"
    run ensure_marketplace_compat_symlink
    [ "$status" -eq 0 ]
    [[ "$output" == *"created legacy marketplace symlink"* ]]
    [ -L "$CLAUDE_CONFIG_DIR/plugins/marketplaces/thedotmack" ]
}

@test "ensure_marketplace_compat_symlink is idempotent when legacy path already exists" {
    _source_heal
    actual="$CLAUDE_CONFIG_DIR/plugins/marketplaces/thedotmack-claude-mem"
    legacy="$CLAUDE_CONFIG_DIR/plugins/marketplaces/thedotmack"
    mkdir -p "$actual" "$legacy"
    run ensure_marketplace_compat_symlink
    [ "$status" -eq 0 ]
    [ -z "$output" ]
    # Pre-existing real dir must be left as-is, not replaced by a symlink.
    [ -d "$legacy" ] && [ ! -L "$legacy" ]
}

# --- heal_hooks_json: idempotency on a healthy file (4) ---

@test "heal_hooks_json no-ops on a missing target" {
    _source_heal
    run heal_hooks_json "$TMP/no-hooks.json"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "heal_hooks_json leaves a healthy hooks.json untouched" {
    _source_heal
    healthy="$TMP/healthy.hooks.json"
    printf '%s\n' '{ "hooks": { "Stop": [] } }' > "$healthy"
    before="$(cat "$healthy")"
    run heal_hooks_json "$healthy"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
    [ "$(cat "$healthy")" = "$before" ]
}
