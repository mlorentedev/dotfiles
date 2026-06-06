#!/usr/bin/env bats
# Tests for scripts/claude-mem-heal.ps1::Repair-HooksJson
# (spec 2026-05-27-claude-mem-heal-consumer-epipe).
#
# Cross-OS parity for the consumer-EPIPE fix: the PowerShell heal must produce
# the same structural outcome as claude-mem-heal.sh::heal_hooks_json -- converge
# both the pristine `break; }; done` form and the BUG-017-era `}; done | head
# -n1` form onto the EPIPE-safe `}; done | sed -n 1p` drain, while preserving
# the BUG-018 continue directive. Driven via pwsh against fixtures; no real
# ~/.claude mutation.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export HEAL_PS1="$DOTFILES_DIR/scripts/claude-mem-heal.ps1"
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    TMP="$(mktemp -d)"
    # Point CLAUDE_CONFIG_DIR at the sandbox so dot-sourcing never touches the
    # real profile and never reads $env:USERPROFILE (StrictMode-safe on Linux).
    export CLAUDE_CONFIG_DIR="$TMP/claude"
    mkdir -p "$CLAUDE_CONFIG_DIR"
    # Function-only copy: truncate before the top-level heal loop, which ends in
    # `exit 0` and would kill the test pwsh. Marker matches the .sh convention.
    FUNCS="$TMP/heal-funcs.ps1"
    awk '/^# Heal every cached version/ { exit } { print }' "$HEAL_PS1" > "$FUNCS"
}

teardown() {
    [ -n "${TMP:-}" ] && rm -rf "$TMP"
}

# Convert an MSYS path to a Windows path for the native pwsh process. On Linux
# (no cygpath) pwsh understands the path as-is.
_winpath() {
    if command -v cygpath >/dev/null 2>&1; then cygpath -w "$1"; else printf '%s' "$1"; fi
}

# Drive Repair-HooksJson against a fixture path via pwsh.
_repair() {
    pwsh -NoProfile -Command ". '$(_winpath "$FUNCS")'; Repair-HooksJson -Target '$(_winpath "$1")'" 2>&1
}

# Drive Repair-McpJson against a fixture path via pwsh.
_repair_mcp() {
    pwsh -NoProfile -Command ". '$(_winpath "$FUNCS")'; Repair-McpJson -Target '$(_winpath "$1")'" 2>&1
}

@test "Repair-HooksJson converts the pristine break;}done form to sed -n 1p (AC3)" {
    f="$TMP/break.hooks.json"
    printf '%s\n' '{ "command": "X | while IFS= read -r _R; do [ -f Z ] && { printf Y; break; }; done); node B hook claude-code session-init" }' > "$f"
    _repair "$f"
    grep -qF 'sed -n 1p' "$f"
    ! grep -qF 'head -n1' "$f"
    grep -qF 'session-init 2>/dev/null' "$f"
}

@test "Repair-HooksJson converts the already-head-n1 form to sed -n 1p (AC3)" {
    f="$TMP/headn1.hooks.json"
    printf '%s\n' '{ "command": "X | while IFS= read -r _R; do [ -f Z ] && { printf Y; }; done | head -n1); node B hook claude-code observation 2>/dev/null; echo Z" }' > "$f"
    _repair "$f"
    grep -qF 'sed -n 1p' "$f"
    ! grep -qF 'head -n1' "$f"
}

@test "Repair-HooksJson is idempotent: re-healing the fixed form is a no-op (AC4)" {
    f="$TMP/idem.hooks.json"
    printf '%s\n' '{ "command": "X | while IFS= read -r _R; do [ -f Z ] && { printf Y; break; }; done); node B hook claude-code session-init" }' > "$f"
    _repair "$f" >/dev/null
    first="$(cat "$f")"
    out="$(_repair "$f")"
    [ "$(cat "$f")" = "$first" ]
    [[ "$out" != *"patched"* ]]
}

@test "Repair-HooksJson leaves a healthy hooks.json untouched" {
    f="$TMP/healthy.hooks.json"
    printf '%s\n' '{ "hooks": { "Stop": [] } }' > "$f"
    before="$(cat "$f")"
    _repair "$f" >/dev/null
    [ "$(cat "$f")" = "$before" ]
}

@test "Repair-HooksJson no-ops on a missing target" {
    out="$(_repair "$TMP/does-not-exist.json")"
    [ -z "$out" ]
}

# --- Repair-McpJson: .mcp.json consumer-EPIPE drain parity ---
# (task 2026-06-06-mcp-json-consumer-epipe-drain). The emitted template's
# path-resolution pipe must drain via `sed -n 1p` (reads to EOF), not the
# early-closing `head -n1` that races under SIGPIPE-ignore -- the same root
# cause Repair-HooksJson fixes for hooks.json, mirrored for .mcp.json.

@test "Repair-McpJson emits the sed -n 1p drain, not head -n1 (mcp consumer-EPIPE)" {
    f="$TMP/v13.mcp.json"
    printf '%s\n' '{ "command": "sh", "args": ["-c", "X | while IFS= read -r _R; do printf Y; done | head -n1"] }' > "$f"
    out="$(_repair_mcp "$f")"
    [[ "$out" == *"patched .mcp.json"* ]]
    grep -qF 'sed -n 1p' "$f"
    ! grep -qF 'head -n1' "$f"
    grep -qF '"mcpServers"' "$f"
}
