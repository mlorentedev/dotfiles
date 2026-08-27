#!/usr/bin/env bats
# Tests for scripts/vault-maintenance-weekly.sh (TEST-001 / #128)
#
# This script is mostly side-effectful: it runs `dotf vault crystallize --all`
# (CLI-050 / #1269 — was the sibling script knowledge-crystallize.sh) plus the
# sibling vault-health.sh, writes a log under $HOME/.local/share, and fires a
# best-effort desktop notification. The real
# maintenance run needs the Obsidian vault + every project, so we cannot unit
# test it directly. Instead we:
#   - assert syntax + structural guards on the real file, and
#   - drive the real log-writing + issue-counting path end-to-end against a
#     COPY of the script placed next to stub siblings, with HOME redirected to
#     a temp dir (no vault, no network, no notify-send dependency).
#
# The end-to-end run is exercised under BOTH zsh and bash. A pre-existing
# portability defect surfaced while writing this coverage: the section-header
# lines `printf '--- ... ---'` aborted under bash with "printf: --: invalid
# option" (the format string begins with `--`), and with `set -e` killed the
# whole log block — so the script only worked under zsh. Fixed in the same PR
# (incident->guard) to `printf '%s\n' '--- ... ---'`; the bash behavioral test
# below is the regression guard (it fails on the old script, passes on the fix).

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    export MAINT_SCRIPT="$SCRIPTS_DIR/vault-maintenance-weekly.sh"
    TMP="$(mktemp -d)"
}

teardown() {
    [ -z "${TMP:-}" ] || rm -rf "$TMP"
}

# --- Syntax (1 & 2) ---

@test "vault-maintenance-weekly.sh valid bash syntax" {
    bash -n "$MAINT_SCRIPT"
}

@test "vault-maintenance-weekly.sh valid zsh syntax" {
    if command -v zsh >/dev/null 2>&1; then
        zsh -n "$MAINT_SCRIPT"
    else
        skip "zsh not available"
    fi
}

# --- Structural guards (the script takes no args / has no usage seam) ---

@test "vault-maintenance-weekly.sh uses set -euo pipefail" {
    grep -q 'set -euo pipefail' "$MAINT_SCRIPT"
}

@test "vault-maintenance-weekly.sh derives SCRIPT_DIR with the zsh-safe BASH_SOURCE fallback" {
    grep -qF '${BASH_SOURCE[0]:-$0}' "$MAINT_SCRIPT"
}

@test "vault-maintenance-weekly.sh invokes both maintenance steps best-effort (|| true)" {
    grep -qE 'dotf vault crystallize --all .*\|\| true' "$MAINT_SCRIPT"
    grep -qE 'vault-health.sh" .*\|\| true' "$MAINT_SCRIPT"
}

@test "vault-maintenance-weekly.sh guards notify-send behind command -v (headless-safe)" {
    grep -qF 'command -v notify-send >/dev/null 2>&1' "$MAINT_SCRIPT"
}

@test "vault-maintenance-weekly.sh counts issues with grep -ciE and a 0 fallback" {
    grep -qE 'grep -ciE .*\|\| printf' "$MAINT_SCRIPT"
}

# --- Behavior: drive the real log/count logic against stub siblings ---

# Build a sandbox: a copy of the script next to stub siblings, HOME redirected,
# and a no-op notify-send shim first on PATH so the notification branch never
# touches the real desktop bus.
_prep_sandbox() {
    # $1 = body printed by the `dotf vault crystallize` stub (controls issue count)
    local crystallize_body="$1"
    cp "$MAINT_SCRIPT" "$TMP/vault-maintenance-weekly.sh"
    cat > "$TMP/dotf" <<EOF
#!/usr/bin/env bash
if [ "\$1" = "vault" ] && [ "\$2" = "crystallize" ]; then
    printf '%s\n' "$crystallize_body"
fi
EOF
    cat > "$TMP/vault-health.sh" <<'EOF'
#!/usr/bin/env bash
printf 'vault-health stub: ok\n'
EOF
    cat > "$TMP/notify-send" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "$TMP/dotf" "$TMP"/*.sh "$TMP/notify-send"
    export FAKE_HOME="$TMP/home"
    mkdir -p "$FAKE_HOME"
}

@test "vault-maintenance-weekly.sh writes a log and reports its path (zsh run)" {
    if ! command -v zsh >/dev/null 2>&1; then
        skip "zsh not available"
    fi
    _prep_sandbox "all clean"
    run env HOME="$FAKE_HOME" PATH="$TMP:$PATH" zsh "$TMP/vault-maintenance-weekly.sh"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Log written to"* ]]
    [ -f "$FAKE_HOME/.local/share/vault-maintenance/latest.log" ]
}

@test "vault-maintenance-weekly.sh log captures both maintenance sections (zsh run)" {
    if ! command -v zsh >/dev/null 2>&1; then
        skip "zsh not available"
    fi
    _prep_sandbox "all clean"
    run env HOME="$FAKE_HOME" PATH="$TMP:$PATH" zsh "$TMP/vault-maintenance-weekly.sh"
    [ "$status" -eq 0 ]
    log="$FAKE_HOME/.local/share/vault-maintenance/latest.log"
    grep -qF 'dotf vault crystallize --all' "$log"
    grep -qF 'vault-health' "$log"
    grep -qF '=== Done:' "$log"
}

@test "vault-maintenance-weekly.sh runs cleanly under bash - guards the printf '--' regression" {
    # Regression guard: section headers used `printf '--- ... ---'`, which under
    # bash abort with "printf: --: invalid option" and (set -e) kill the run before
    # the sections write. Fixed to `printf '%s\n' '--- ... ---'`. This FAILS on the
    # old script (sections + Done missing) and PASSES on the fix.
    _prep_sandbox "all clean"
    run env HOME="$FAKE_HOME" PATH="$TMP:$PATH" bash "$TMP/vault-maintenance-weekly.sh"
    [ "$status" -eq 0 ]
    [[ "$output" != *"invalid option"* ]]
    log="$FAKE_HOME/.local/share/vault-maintenance/latest.log"
    grep -qF 'dotf vault crystallize --all' "$log"
    grep -qF 'vault-health' "$log"
    grep -qF '=== Done:' "$log"
}

@test "vault-maintenance-weekly.sh tolerates a sibling that prints issue keywords (still exit 0, zsh run)" {
    if ! command -v zsh >/dev/null 2>&1; then
        skip "zsh not available"
    fi
    # 'warning'/'stale'/'action' in the crystallize output bumps the issue
    # counter; the script must still complete cleanly (the count only drives
    # the notification urgency, never the exit code).
    _prep_sandbox "WARNING: 3 stale memory files need action"
    run env HOME="$FAKE_HOME" PATH="$TMP:$PATH" zsh "$TMP/vault-maintenance-weekly.sh"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Log written to"* ]]
}
