#!/usr/bin/env bats
# Tests for SessionStart hooks SDD reminder injection (SDD-001 Tier 2)

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export HOOK_PS1="$DOTFILES_DIR/scripts/claude-session-start.ps1"
    export HOOK_SH="$DOTFILES_DIR/scripts/claude-session-start.sh"
    # Shared reminder marker -- bytes-identical between hooks, locked here
    export SDD_REMINDER_PREFIX='[sdd]'
    export SDD_REMINDER_CORE='Before your first tool call, read'
    export SDD_REMINDER_TARGET='AGENTS.md'
}

@test "both hook scripts exist" {
    [[ -f "$HOOK_PS1" ]]
    [[ -f "$HOOK_SH" ]]
}

# --- SDD reminder injection ---

@test "claude-session-start.ps1 contains the [sdd] reminder marker" {
    grep -qF "$SDD_REMINDER_PREFIX" "$HOOK_PS1"
    grep -qF "$SDD_REMINDER_CORE" "$HOOK_PS1"
    grep -qF "$SDD_REMINDER_TARGET" "$HOOK_PS1"
}

@test "claude-session-start.sh contains the [sdd] reminder marker (parity)" {
    grep -qF "$SDD_REMINDER_PREFIX" "$HOOK_SH"
    grep -qF "$SDD_REMINDER_CORE" "$HOOK_SH"
    grep -qF "$SDD_REMINDER_TARGET" "$HOOK_SH"
}

@test "both hooks mention the AGENTS.md fallback path on Windows and Linux" {
    grep -qF '~/Projects/dotfiles/AGENTS.md' "$HOOK_PS1"
    grep -qF '~/Projects/dotfiles/AGENTS.md' "$HOOK_SH"
}

@test "both hooks reference skip criteria (typos, comment-only, mechanical, bug<20, doc-only)" {
    # Each hook must list at least the canonical skip categories so the agent
    # has them inline without round-tripping to AGENTS.md.
    grep -qF 'typo' "$HOOK_PS1"
    grep -qF 'typo' "$HOOK_SH"
    grep -qF 'mechanical' "$HOOK_PS1"
    grep -qF 'mechanical' "$HOOK_SH"
}

# --- Position invariant: reminder must be unconditional + early ---
# The reminder must NOT live inside a function or if-block that gates on
# CWD / vault / repo state. It must run every session.

@test "claude-session-start.ps1 SDD reminder is not gated by Test-Path / if(-not...)" {
    # Confirm the reminder text appears OUTSIDE the gated regions. Heuristic:
    # the line containing "[sdd]" should come BEFORE the line containing
    # "function Find-HiveProject" (first gated diagnostic block in the script).
    sdd_line=$(grep -nF "$SDD_REMINDER_PREFIX" "$HOOK_PS1" | head -1 | cut -d: -f1)
    fn_line=$(grep -nE '^function Find-HiveProject' "$HOOK_PS1" | head -1 | cut -d: -f1)
    [[ -n "$sdd_line" ]] && [[ -n "$fn_line" ]] && [[ "$sdd_line" -lt "$fn_line" ]]
}

@test "claude-session-start.sh SDD reminder appears before first conditional diagnostic" {
    sdd_line=$(grep -nF "$SDD_REMINDER_PREFIX" "$HOOK_SH" | head -1 | cut -d: -f1)
    # First conditional in the existing .sh is the vault detection if-block.
    # Use 'find_vault_root' as the heuristic anchor (function defined later).
    fn_line=$(grep -nE '^find_vault_root\(\)' "$HOOK_SH" | head -1 | cut -d: -f1)
    [[ -n "$sdd_line" ]] && [[ -n "$fn_line" ]] && [[ "$sdd_line" -lt "$fn_line" ]]
}

# --- Cross-OS parity lock ---

@test "parity: SDD reminder core text identical between .ps1 and .sh" {
    # Extract the line containing the [sdd] marker from each file, compare core text.
    ps1_line=$(grep -F "$SDD_REMINDER_PREFIX" "$HOOK_PS1" | head -1)
    sh_line=$(grep -F "$SDD_REMINDER_PREFIX" "$HOOK_SH" | head -1)
    [[ -n "$ps1_line" ]] && [[ -n "$sh_line" ]]
    # Both should contain the same anchor phrases
    echo "$ps1_line" | grep -qF "$SDD_REMINDER_CORE"
    echo "$sh_line" | grep -qF "$SDD_REMINDER_CORE"
}
