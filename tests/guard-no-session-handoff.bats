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
    # A test that runs it or resolves its path would break; a removal list that
    # names it (setup-windows.bats, WIN-013) would not, so the guard measures
    # invocation and path references, not mention.
    run grep -rlE --exclude='guard-no-session-handoff.bats' '(run |bash |pwsh |scripts/)[^#]*session-handoff' "$REPO/tests"
    [ -z "$output" ]
}

@test "the SessionEnd hook is wired to dotf mem session-end, not a script (anti-regression)" {
    # HARNESS-045 AC1 moved WHERE this is declared, not WHAT it declares. The
    # command used to be a literal in each setup script; it is now one record in
    # harness/manifest.json that `dotf harness bind` emits on both OSes. The
    # invariant is unchanged -- SessionEnd runs the agnostic noun, never a shell
    # twin -- so the guard follows it to the manifest rather than being deleted.
    jq -e '.agents.bind[] | select(.agent == "claude") | .emit_hooks
           | map(select(.event == "SessionEnd" and .command == "mem session-end"))
           | length == 1' "$REPO/harness/manifest.json"
}

@test "the retired session-handoff scripts are not reachable from the manifest" {
    # The emitted command is a `dotf` subcommand suffix, so a resurrected shell
    # twin could only reappear as a path smuggled into one.
    run jq -r '.agents.bind[].emit_hooks[]?.command' "$REPO/harness/manifest.json"
    [ "$status" -eq 0 ]
    case "$output" in
        *session-handoff*|*.sh*|*.ps1*)
            echo "a bind command names a script: $output" >&2
            return 1 ;;
    esac
}
