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

load 'lib/refute'

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

@test "vault-maintenance-weekly.sh hardens PATH with ~/.local/bin before calling bare dotf" {
    # cron runs with a minimal PATH that excludes ~/.local/bin (install-dotf.sh's
    # install target); without this dotf silently resolves to nothing under
    # `|| true` every Sunday. The behavioral regression guard is below.
    grep -qF 'export PATH="$HOME/.local/bin:$PATH"' "$MAINT_SCRIPT"
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

@test "vault-maintenance-weekly.sh resolves dotf under a cron-minimal PATH (no inherited ~/.local/bin)" {
    # Regression guard for the reviewed regression: every prior test prepends
    # $TMP (holding the dotf stub) onto PATH, which also happens to mask a
    # missing PATH hardening in the script. cron never does that -- it starts
    # from a minimal PATH with no ~/.local/bin -- so this places the stub where
    # the REAL installer puts dotf ($HOME/.local/bin) and runs with a PATH that
    # does not already contain it, so only the script's own PATH export can
    # make it resolve.
    _prep_sandbox "all clean"
    mkdir -p "$FAKE_HOME/.local/bin"
    mv "$TMP/dotf" "$FAKE_HOME/.local/bin/dotf"
    run env HOME="$FAKE_HOME" PATH="/usr/bin:/bin" bash "$TMP/vault-maintenance-weekly.sh"
    [ "$status" -eq 0 ]
    log="$FAKE_HOME/.local/share/vault-maintenance/latest.log"
    refute_grep_fixed 'command not found' "$log"
    grep -qF 'all clean' "$log"
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

# --- The Go path: `dotf vault maintain` (CLI-021 increment 3, #490) ---
#
# BUILT BESIDE the script above, which is still what cron runs
# (setup-linux.sh:1605) — the cutover is CLI-023 (#492). So both implementations
# are live and both are exercised here.
#
# Deliberately NOT a golden-parity suite, unlike increments 1 and 2. The twin's
# output is a timestamped log wrapping two subcommands whose byte-parity is
# already proven by knowledge-crystallize-go-parity.bats and
# vault-health-go-parity.bats; re-proving it through a third fixture scheme
# would measure the same thing a third time. What is left to characterize is
# the WRAPPER — log location, section framing, exit status — and those are
# behaviours. The unit-level seams (issue regex, notification threshold, the
# per-OS log path) are table-tested in cli/internal/vault/maintain_test.go.
#
# Skips (never fails) when the Go toolchain is absent, so a shell-only checkout
# still runs the rest of this file — same reasoning as the two parity suites
# (#807 / BUG-055).

_build_dotf_maintain() {
    command -v go >/dev/null 2>&1 || skip "go toolchain not installed"
    DOTF_BIN="${BATS_FILE_TMPDIR:-$TMP}/dotf-maintain"
    if [ ! -x "$DOTF_BIN" ]; then
        # A missing toolchain skips (above); a toolchain that FAILS to build is
        # a real defect and must fail, not read as harmless skips.
        ( cd "$BATS_TEST_DIRNAME/../cli" && go build -o "$DOTF_BIN" ./cmd/dotf ) || return 1
    fi
    export DOTF_BIN
    # An empty HOME: no ~/.claude/projects, so crystallize discovers nothing.
    # An empty VAULT_DIR and no `obsidian` on PATH: health degrades exactly as
    # it does on a headless box. No vault, no network, no desktop bus touched.
    export FAKE_HOME="$TMP/gohome"
    export FAKE_VAULT="$TMP/govault"
    mkdir -p "$FAKE_HOME" "$FAKE_VAULT"
    # A no-op notify-send FIRST on PATH, so the notification branch can never
    # reach the real desktop bus — the leak tests/golden/vault-health guards
    # against by replacing PATH rather than extending it.
    cat > "$TMP/notify-send" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "$TMP/notify-send"
}

_go_log() { printf '%s/.local/share/vault-maintenance/latest.log' "$FAKE_HOME"; }

@test "dotf vault maintain writes the log at the twin's location and reports its path" {
    _build_dotf_maintain
    run env HOME="$FAKE_HOME" VAULT_DIR="$FAKE_VAULT" PATH="$TMP:/usr/bin:/bin" \
        "$DOTF_BIN" vault maintain
    [ "$status" -eq 0 ]
    [[ "$output" == *"Log written to"* ]]
    # Same path the .sh has always written, because a human looks for the log by
    # path and the cutover must not move it.
    [ -f "$(_go_log)" ]
}

@test "dotf vault maintain log captures both maintenance sections in order" {
    _build_dotf_maintain
    run env HOME="$FAKE_HOME" VAULT_DIR="$FAKE_VAULT" PATH="$TMP:/usr/bin:/bin" \
        "$DOTF_BIN" vault maintain
    [ "$status" -eq 0 ]
    log="$(_go_log)"
    grep -qF 'dotf vault crystallize --all' "$log"
    grep -qF 'vault-health' "$log"
    grep -qF '=== Done:' "$log"
    # Order, not mere presence: health after crystallize, footer after both.
    cryst=$(grep -n 'dotf vault crystallize --all' "$log" | head -1 | cut -d: -f1)
    health=$(grep -n -- '--- vault-health ---' "$log" | head -1 | cut -d: -f1)
    done_line=$(grep -n '=== Done:' "$log" | head -1 | cut -d: -f1)
    [ "$cryst" -lt "$health" ]
    [ "$health" -lt "$done_line" ]
}

@test "dotf vault maintain exits 0 when health reports findings (a finding is not a failure)" {
    # No obsidian on PATH, so health cannot pass: it reports and the report
    # lands in the log. The RUN still did its job, so the status stays 0 —
    # otherwise cron mails the owner every week the GUI happened to be closed.
    # This is the guard for that decision (spec tasks.md §4), not an accident.
    _build_dotf_maintain
    run env HOME="$FAKE_HOME" VAULT_DIR="$FAKE_VAULT" PATH="$TMP:/usr/bin:/bin" \
        "$DOTF_BIN" vault maintain
    [ "$status" -eq 0 ]
    [[ "$output" == *"Vault health:"* ]]
    grep -qE 'warning|fail|action|stale' "$(_go_log)" || true
}

@test "dotf vault maintain needs no PATH hardening under a cron-minimal PATH" {
    # The .sh MUST export PATH="$HOME/.local/bin:$PATH" or its bare `dotf` call
    # silently no-ops under `|| true` every Sunday (its lines 12-16, guarded at
    # line 147 above). The Go port composes IN-PROCESS, so there is no
    # subprocess whose resolution can fail — this asserts that structurally, by
    # running with a PATH that contains neither ~/.local/bin nor the build dir.
    _build_dotf_maintain
    run env HOME="$FAKE_HOME" VAULT_DIR="$FAKE_VAULT" PATH="/usr/bin:/bin" \
        "$DOTF_BIN" vault maintain
    [ "$status" -eq 0 ]
    log="$(_go_log)"
    refute_grep_fixed 'command not found' "$log"
    grep -qF '=== Done:' "$log"
}
