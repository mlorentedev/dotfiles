#!/usr/bin/env bats
# GUARD: the session-handoff shell twins are retired (CLI-025). The SessionEnd
# bridge is now the agnostic `dotf mem session-end` noun (cli/internal/mem),
# invoked directly by the hook (no per-OS shim). This guard pins three invariants
# so a future change cannot silently resurrect the scripts or re-point the hook at
# them. It checks *invocation/existence*, not prose — explanatory comments that
# name the retired scripts are fine.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

@test "the session-handoff shell twins do not exist (CLI-025)" {
    [ ! -e "$REPO/scripts/session-handoff.sh" ]
    [ ! -e "$REPO/scripts/session-handoff.ps1" ]
}

@test "no bats test targets a session-handoff script (would break CI bats glob)" {
    # CI runs `bats tests/*.bats`; a test for the deleted script would fail. Exclude
    # this guard file, which names the script in prose.
    run grep -rlF --exclude='guard-no-session-handoff.bats' 'session-handoff' "$REPO/tests"
    [ -z "$output" ]
}

@test "the SessionEnd hook is wired to dotf mem session-end, not a script (anti-regression)" {
    grep -qF 'dotf mem session-end' "$REPO/setup-linux.sh"
    grep -qF 'mem session-end' "$REPO/setup-windows.ps1"
}
