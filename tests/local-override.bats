#!/usr/bin/env bats
# IDEAS-001: machine-local override files (.zshrc.local / .bashrc.local).
# Verifies the mechanism's STRUCTURE (guarded source present + last) and the
# GUARD semantics (present -> sourced, absent -> graceful no-op) in both shells.
# We do not source the full rc (env-fragile in CI); the structural test proves
# the real rc carries the exact line, the functional test proves its logic.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    TMP="/tmp/bats_localoverride_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$TMP"
}

teardown() {
    rm -rf "$TMP"
}

@test ".zshrc sources ~/.zshrc.local (guarded) as the last non-blank line" {
    last="$(grep -vE '^[[:space:]]*$' "$REPO/.zshrc" | tail -1)"
    [[ "$last" == *'.zshrc.local'* ]]
    [[ "$last" == *'-r '* ]]
    [[ "$last" == *'-f '* ]]
}

@test ".bashrc sources ~/.bashrc.local (guarded) as the last non-blank line" {
    last="$(grep -vE '^[[:space:]]*$' "$REPO/.bashrc" | tail -1)"
    [[ "$last" == *'.bashrc.local'* ]]
    [[ "$last" == *'-r '* ]]
    [[ "$last" == *'-f '* ]]
}

@test "guard sources the local file when present (zsh + bash)" {
    printf 'export TEST_IDEAS001_LOCAL=42\n' > "$TMP/rc.local"
    for sh in zsh bash; do
        command -v "$sh" >/dev/null 2>&1 || continue
        out="$("$sh" -c '[ -r "$1" ] && [ -f "$1" ] && . "$1"; printf "%s" "${TEST_IDEAS001_LOCAL:-unset}"' _ "$TMP/rc.local")"
        [ "$out" = "42" ]
    done
}

@test "guard is a graceful no-op when the local file is absent (zsh + bash)" {
    for sh in zsh bash; do
        command -v "$sh" >/dev/null 2>&1 || continue
        run "$sh" -c '[ -r "$1" ] && [ -f "$1" ] && . "$1"; printf "ok"' _ "$TMP/missing.local"
        [ "$status" -eq 0 ]
        [ "$output" = "ok" ]
    done
}

@test "example files exist with commented use-cases + the secrets warning" {
    [ -f "$REPO/.zshrc.local.example" ]
    [ -f "$REPO/.bashrc.local.example" ]
    grep -q '^#' "$REPO/.zshrc.local.example"
    grep -q '^#' "$REPO/.bashrc.local.example"
    grep -qi 'age system' "$REPO/.zshrc.local.example"
    grep -qi 'age system' "$REPO/.bashrc.local.example"
}

@test ".gitignore excludes both local override files" {
    grep -qx '.zshrc.local' "$REPO/.gitignore"
    grep -qx '.bashrc.local' "$REPO/.gitignore"
}
