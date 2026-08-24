#!/usr/bin/env bash
# Suite-wide guard: a test can never launch a real GUI application.
#
# Measured 2026-08-23: a suite run left **18 Obsidian processes** alive, talking
# to the developer's real vault (`obsidian --no-sandbox dead-ends --vault
# knowledge`). Production scripts resolve these tools through PATH, so any test
# that runs one without isolating PATH finds the developer's actual install and
# opens windows against actual data.
#
# The fix is not "remember to stub it". `tests/golden/vault-health/lib.sh`
# already does this correctly for itself — it replaces PATH and then asserts no
# leak — and the rest of the suite had no such protection, which is precisely
# the shape that recurs. This makes the protection the default for every test,
# so a new test file inherits it without its author knowing it exists.
#
# The interceptors REFUSE rather than succeed quietly. A silent exit 0 would
# turn "opens a window" into "passes for the wrong reason", trading a visible
# failure for an invisible one. A test that legitimately needs one of these
# tools stubs it itself, and wins because its stub is earlier on PATH.
#
# KNOWN GAP, stated rather than papered over: bats runs `setup_suite` only when
# invoked over a directory or more than one file. `bats tests/one-file.bats`
# does not load this. CI and `bats tests/*.bats` both do.
# `tests/guard-no-gui.bats` reports the gap rather than skipping silently.

# GUI_BINARIES is the class, not just the one that bit. Every entry is a tool
# that opens a window or attaches to a running desktop session, and every one of
# them is installed on a developer machine here.
GUI_BINARIES=(obsidian orca code xdg-open)

# GUI_GUARD_LOG records any attempt, so the NEXT occurrence names its own
# culprit instead of needing a process-table archaeology session. Blocking alone
# would have stopped the windows; it would not have said which test opened them.
export GUI_GUARD_LOG="${BATS_SUITE_TMPDIR:-/tmp}/gui-guard-attempts.log"

setup_suite() {
    local bin_dir="${BATS_SUITE_TMPDIR:?bats did not provide a suite tmpdir}/gui-guard"
    mkdir -p "$bin_dir"

    local bin
    for bin in "${GUI_BINARIES[@]}"; do
        _gui_guard_write_interceptor "$bin_dir" "$bin"
    done

    export PATH="$bin_dir:$PATH"
    export GUI_GUARD_BIN_DIR="$bin_dir"
}

# _gui_guard_write_interceptor emits one refusing stand-in. The message names
# the remedy inline: a refusal whose fix has to be looked up is a refusal people
# route around by deleting the guard.
_gui_guard_write_interceptor() {
    local dir="$1" name="$2"
    cat > "$dir/$name" <<INTERCEPTOR
#!/usr/bin/env bash
{
    printf '%s\n' "\$(date -u +%Y-%m-%dT%H:%M:%SZ) $name \$*"
    printf '  file=%s\n' "\${BATS_TEST_FILENAME:-<unknown>}"
    printf '  test=%s\n' "\${BATS_TEST_NAME:-<unknown>}"
} >> "\${GUI_GUARD_LOG:-/dev/null}" 2>/dev/null

cat >&2 <<'MSG'
tests: refused to launch \`$name\` — a test reached a real GUI application.

Tests must never open a window or touch live desktop state. On this machine
\`$name\` is installed, so PATH resolution finds it and the test drives the real
application against real data.

Stub it in your own test, which wins because it is earlier on PATH:

    printf '#!/bin/sh\nexit 0\n' > "\$BATS_TEST_TMPDIR/$name"
    chmod +x "\$BATS_TEST_TMPDIR/$name"
    PATH="\$BATS_TEST_TMPDIR:\$PATH"

Or replace PATH outright, as tests/golden/vault-health/lib.sh does, when the
test must also cover the case where the tool is absent.

Guard: tests/setup_suite.bash
MSG
# 99, not 127. 127 means "command not found", and this command was found — it
# declined. bats says so out loud (BW01), and a wrong code here would teach
# every reader of a failure the wrong cause.
exit 99
INTERCEPTOR
    chmod +x "$dir/$name"
}

# teardown_suite closes the gap the interceptors cannot: PATH only governs a
# bare command name, so anything invoking an absolute path — `~/.local/bin/
# obsidian`, an AppImage — walks straight past them.
#
# The detection is exact rather than heuristic. Electron derives --user-data-dir
# from the environment, so a GUI launched from inside a test carries a bats
# tmpdir in that flag, and nothing a human opens ever does. That is the
# signature the 2026-08-23 incident left behind, and it is what distinguishes a
# test's stray process from the developer's own editor sitting open beside it.
teardown_suite() {
    local strays
    strays="$(_gui_guard_test_shaped_processes)"
    [ -n "$strays" ] || return 0

    {
        printf '\n'
        printf 'GUI GUARD: a test launched a real GUI application and left it running.\n'
        printf '\n'
        printf 'These carry a bats tmpdir in --user-data-dir, so they came from this suite\n'
        printf 'rather than from anything you opened:\n\n'
        printf '%s\n' "$strays"
        printf '\n'
        printf 'The PATH interceptors did not catch it, which means the call used an\n'
        printf 'absolute path rather than a bare command name. Find the caller and give it\n'
        printf 'a stub, or have it resolve the tool through PATH so the guard can see it.\n'
        printf '\n'
        printf 'Guard: tests/setup_suite.bash\n'
    } >&2

    # Leave nothing behind for the developer to hunt down and kill by hand.
    printf '%s\n' "$strays" | awk '{print $1}' | while read -r pid; do
        [ -n "$pid" ] && kill -TERM "$pid" 2>/dev/null
    done

    return 1
}

# _gui_guard_test_shaped_processes lists pid+command for GUI processes whose
# user-data-dir points inside a bats tmpdir. Matching on that flag rather than
# on the binary name is what keeps a developer's own open editor out of the
# result — the check must never kill something a human started.
_gui_guard_test_shaped_processes() {
    command -v pgrep >/dev/null 2>&1 || return 0

    local bin line
    for bin in "${GUI_BINARIES[@]}"; do
        # pgrep by NAME, then filter on the flag. Matching the flag with
        # `pgrep -f` would also match this very shell, whose own argv contains
        # the pattern — the false positive that made the first reading of this
        # incident report three strays that were the measurement itself.
        while IFS= read -r line; do
            case "$line" in
                *--user-data-dir=*bats-run*) printf '%s\n' "$line" ;;
            esac
        done < <(pgrep -a "$bin" 2>/dev/null || true)
    done
}
