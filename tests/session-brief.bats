#!/usr/bin/env bats
# Unit tests for scripts/session-brief.sh — the agent-agnostic session-brief
# core (ADR-023, HARNESS-026). Sources the lib (SESSION_BRIEF_LIB=1) to test the
# sb_* emitters in isolation, and runs the script standalone to exercise the
# --format runner. The Claude adapter's byte-equivalence is covered separately
# in session-start-config.bats.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    CORE="$DOTFILES_DIR/scripts/session-brief.sh"
    # Library mode: defines find_vault_root + sb_* WITHOUT running main.
    SESSION_BRIEF_LIB=1 . "$CORE"
    TMP="$BATS_TEST_TMPDIR"
}

@test "core: session-brief.sh exists and is executable" {
    [ -x "$CORE" ]
}

# --- sb_vault_detect ---------------------------------------------------------

@test "sb_vault_detect emits the headline for a vault root" {
    run sb_vault_detect "/some/myvault"
    [ "$status" -eq 0 ]
    [ "$output" = "Obsidian vault detected: myvault (/some/myvault)" ]
}

@test "sb_vault_detect emits nothing without a root" {
    run sb_vault_detect ""
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

# --- sb_specs ----------------------------------------------------------------

@test "sb_specs is silent when there is no specs/ dir" {
    run sb_specs "$TMP/empty"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "sb_specs counts active and archived specs" {
    mkdir -p "$TMP/r/specs/FOO-1" "$TMP/r/specs/BAR-2" "$TMP/r/specs/archive/OLD-1"
    run sb_specs "$TMP/r"
    [ "$status" -eq 0 ]
    [[ "$output" == *"[specs] 2 active, 1 archived"* ]]
}

@test "sb_specs flags specs carrying AGENT-DRAFT tags" {
    mkdir -p "$TMP/r/specs/FOO-1"
    printf '[AGENT-DRAFT] todo\n' > "$TMP/r/specs/FOO-1/proposal.md"
    run sb_specs "$TMP/r"
    [[ "$output" == *"unresolved [AGENT-DRAFT]"* ]]
    [[ "$output" == *"- FOO-1"* ]]
}

# --- sb_vault_baseline -------------------------------------------------------

@test "sb_vault_baseline is silent on a healthy vault" {
    mkdir -p "$TMP/v/00_meta/patterns" "$TMP/v/00_meta/skills/foo"
    : > "$TMP/v/00_meta/patterns/_index.md"
    : > "$TMP/v/00_meta/skills/README.md"
    : > "$TMP/v/README.md"
    printf 'content\n' > "$TMP/v/00_meta/skills/foo/SKILL.md"
    run sb_vault_baseline "$TMP/v"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "sb_vault_baseline flags a missing critical file" {
    mkdir -p "$TMP/v/00_meta/patterns"
    : > "$TMP/v/00_meta/patterns/_index.md"
    run sb_vault_baseline "$TMP/v"
    [[ "$output" == *"Vault baseline FAIL"* ]]
    [[ "$output" == *"MISSING: README.md"* ]]
}

@test "sb_vault_baseline flags an empty SKILL.md" {
    mkdir -p "$TMP/v/00_meta/patterns" "$TMP/v/00_meta/skills/foo"
    : > "$TMP/v/00_meta/patterns/_index.md"
    : > "$TMP/v/00_meta/skills/README.md"
    : > "$TMP/v/README.md"
    : > "$TMP/v/00_meta/skills/foo/SKILL.md"
    run sb_vault_baseline "$TMP/v"
    [[ "$output" == *"EMPTY: 00_meta/skills/foo/SKILL.md"* ]]
}

@test "sb_vault_baseline is silent when vault_root is empty" {
    run sb_vault_baseline ""
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

# --- sb_vault_health ---------------------------------------------------------

@test "sb_vault_health reports not-installed when vault-health.sh is absent" {
    run sb_vault_health "" "" "$TMP/nodir"
    [ "$status" -eq 0 ]
    [[ "$output" == *"vault-health.sh not found"* ]]
}

@test "sb_vault_health reports ALL CHECKS PASSED on a clean stub" {
    mkdir -p "$TMP/sd"
    printf '#!/bin/sh\nexit 0\n' > "$TMP/sd/vault-health.sh"
    chmod +x "$TMP/sd/vault-health.sh"
    run sb_vault_health "/v" "v" "$TMP/sd"
    [[ "$output" == *"Vault health: ALL CHECKS PASSED"* ]]
}

# --- standalone --format runner ---------------------------------------------

@test "standalone --format=stdout emits a non-fenced brief (format=stdout)" {
    # Copy into a dir WITHOUT vault-health.sh so the health line is deterministic.
    cp "$CORE" "$TMP/sb.sh"
    run env SESSION_BRIEF_CWD="$TMP" bash "$TMP/sb.sh" --format=stdout
    [ "$status" -eq 0 ]
    [[ "$output" == *"vault-health.sh not found"* ]]
    [ "${lines[0]:0:3}" != '```' ]
}

@test "standalone --format=markdown fences the brief (format=markdown)" {
    cp "$CORE" "$TMP/sb.sh"
    run env SESSION_BRIEF_CWD="$TMP" bash "$TMP/sb.sh" --format=markdown
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = '```text' ]
    [ "${lines[${#lines[@]}-1]}" = '```' ]
}

@test "standalone rejects an unknown format with usage (unknown format)" {
    run sh -c 'bash "$1" --format=bogus 2>&1' _ "$CORE"
    [ "$status" -eq 2 ]
    [[ "$output" == *"--format must be stdout or markdown"* ]]
}

@test "standalone rejects a missing format (unknown format)" {
    run sh -c 'bash "$1" 2>&1' _ "$CORE"
    [ "$status" -eq 2 ]
}
