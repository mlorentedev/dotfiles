#!/usr/bin/env bats
# The guard that keeps tests from launching GUI applications, tested.
#
# A guard nobody verifies is the failure it exists to catch, one level up
# (lessons 212 and 224). So this asserts the interceptors are actually on PATH,
# that they refuse rather than succeed, and that the detector works against a
# case built to trip it.

setup() {
    REPO_ROOT="$BATS_TEST_DIRNAME/.."
}

# Whether setup_suite ran at all. bats loads it only for a directory or more
# than one file, so a single-file run has no guard — reported, never hidden.
_guard_is_active() {
    [ -n "${GUI_GUARD_BIN_DIR:-}" ] && [ -d "${GUI_GUARD_BIN_DIR:-/nonexistent}" ]
}

@test "guard: setup_suite.bash declares the GUI binaries it intercepts" {
    grep -q '^GUI_BINARIES=(' "$REPO_ROOT/tests/setup_suite.bash"
    # obsidian is the one that bit: 18 live processes against the real vault.
    grep -qE '^GUI_BINARIES=\(.*obsidian' "$REPO_ROOT/tests/setup_suite.bash"
}

@test "guard: every declared binary is intercepted on PATH" {
    _guard_is_active || skip "setup_suite did not run (single-file invocation); the guard is inactive here"

    # shellcheck source=setup_suite.bash disable=SC1091
    . "$REPO_ROOT/tests/setup_suite.bash"
    local bin
    for bin in "${GUI_BINARIES[@]}"; do
        local resolved
        resolved="$(command -v "$bin" || true)"
        [ "$resolved" = "$GUI_GUARD_BIN_DIR/$bin" ] || {
            printf 'guard leak: %s resolves to %s, not the interceptor\n' "$bin" "$resolved" >&2
            return 1
        }
    done
}

@test "guard: an intercepted binary refuses instead of succeeding quietly" {
    _guard_is_active || skip "setup_suite did not run (single-file invocation); the guard is inactive here"

    run obsidian --no-sandbox dead-ends --vault knowledge
    # 99 rather than 127: the interceptor was FOUND and refused. 127 would say
    # "not installed", which is a different fact and the wrong remedy.
    # Exit 0 would be the worst outcome of all: the window never opens AND the
    # test that tried passes, so the reach goes unnoticed until someone reads ps.
    [ "$status" -eq 99 ]
    printf '%s' "$output" | grep -q 'refused to launch'
}

@test "guard: the refusal names the remedy, not just the rule" {
    _guard_is_active || skip "setup_suite did not run (single-file invocation); the guard is inactive here"

    run obsidian vault
    # A guard whose fix must be looked up elsewhere is one people remove.
    printf '%s' "$output" | grep -q 'BATS_TEST_TMPDIR'
    printf '%s' "$output" | grep -q 'tests/setup_suite.bash'
}

@test "guard: an attempt is logged with the test that made it" {
    _guard_is_active || skip "setup_suite did not run (single-file invocation); the guard is inactive here"

    run obsidian unresolved --vault knowledge
    [ -f "$GUI_GUARD_LOG" ]
    grep -q 'unresolved --vault knowledge' "$GUI_GUARD_LOG"
    # Blocking alone would stop the windows without ever saying which test
    # opened them — which is exactly the position the 18-process incident left
    # us in, reading a process table after the fact.
    grep -q "$(basename "$BATS_TEST_FILENAME")" "$GUI_GUARD_LOG"
}

# The detector tested against a case built to trip it: a script that reaches for
# obsidian the way a production script does. If the interceptor ever stopped
# matching, this goes red rather than the suite going quietly permissive.
@test "guard: a script that shells out to obsidian is stopped" {
    _guard_is_active || skip "setup_suite did not run (single-file invocation); the guard is inactive here"

    local probe="$BATS_TEST_TMPDIR/reaches-for-obsidian.sh"
    printf '#!/usr/bin/env bash\nobsidian --no-sandbox vault\n' > "$probe"
    chmod +x "$probe"

    run "$probe"
    [ "$status" -eq 99 ]
    printf '%s' "$output" | grep -q 'refused to launch'
}

# A test that stubs the tool itself must still win — the guard bounds the
# default, it does not take the decision away.
@test "guard: a test's own stub takes precedence over the interceptor" {
    _guard_is_active || skip "setup_suite did not run (single-file invocation); the guard is inactive here"

    printf '#!/bin/sh\nprintf "stubbed"\n' > "$BATS_TEST_TMPDIR/obsidian"
    chmod +x "$BATS_TEST_TMPDIR/obsidian"

    run env PATH="$BATS_TEST_TMPDIR:$PATH" obsidian vault
    [ "$status" -eq 0 ]
    [ "$output" = "stubbed" ]
}

@test "guard: setup_suite.bash passes shellcheck" {
    # From PATH: mise installs shellcheck (#2013 W2), and a fixed
    # ~/.local/bin path skipped this test on every machine set up since.
    command -v shellcheck >/dev/null 2>&1 || skip "shellcheck not installed"
    run shellcheck -s bash "$REPO_ROOT/tests/setup_suite.bash"
    [ "$status" -eq 0 ]
}

# Launch a fake GUI in the background. `kill -9` of the fake leaves its own
# child (`sleep`) running as an orphan, and an orphan that still holds bats'
# fd 3 makes bats wait for it to exit: that is how this file took 30.9 s to run
# 0.3 s of tests (CI-004, #1739).
#
# The descriptors are closed rather than the fake `exec`ing `sleep`: the
# detector matches on the process name `obsidian`, and exec would rename it.
_launch_fake() {
    "$@" >/dev/null 2>&1 3>&- &
}

# The leak itself, asserted rather than timed: no descendant of a fake launched
# through _launch_fake holds the descriptor bats waits on.
@test "guard: a fake GUI launched by this file does not hold bats' fd 3" {
    [ -e /proc/self/fd/3 ] || skip "no /proc fd view on this platform"
    command -v pgrep >/dev/null 2>&1 || skip "pgrep not installed"

    local fake="$BATS_TEST_TMPDIR/obsidian"
    printf '#!/bin/sh\nsleep 30\n' > "$fake"
    chmod +x "$fake"
    _launch_fake "$fake" --user-data-dir="$BATS_TEST_TMPDIR/nowhere"
    local pid=$!
    sleep 0.3

    local ours children c leaked=""
    ours="$(readlink /proc/$$/fd/3)"
    children="$(pgrep -P "$pid")"
    kill -9 "$pid" 2>/dev/null || true
    for c in $children; do
        [ "$(readlink "/proc/$c/fd/3" 2>/dev/null)" = "$ours" ] && leaked="$leaked $c"
    done
    # shellcheck disable=SC2086
    kill -9 $children 2>/dev/null || true

    [ -n "$children" ] || { echo "the fake spawned no child; the test proves nothing" >&2; return 1; }
    [ -z "$leaked" ] || { echo "orphans holding bats' fd 3 ($ours):$leaked" >&2; return 1; }
}

# The post-suite detector, tested against a process built to look exactly like
# the strays the incident left behind. A detector never seen firing is a claim,
# not a check — and this one's job is to catch what the PATH interceptors
# structurally cannot: an invocation by absolute path.
# bats test_tags=os-sensitive
@test "guard: the stray detector matches a test-shaped GUI process and not a human's" {
    _guard_is_active || skip "setup_suite did not run (single-file invocation); the guard is inactive here"
    command -v pgrep >/dev/null 2>&1 || skip "pgrep not installed"

    # shellcheck source=setup_suite.bash disable=SC1091
    . "$REPO_ROOT/tests/setup_suite.bash"

    # A fake `obsidian` that sleeps, carrying the signature of a test-launched
    # GUI: a bats tmpdir in --user-data-dir.
    local fake="$BATS_TEST_TMPDIR/obsidian"
    printf '#!/bin/sh\nsleep 30\n' > "$fake"
    chmod +x "$fake"
    _launch_fake "$fake" --user-data-dir="$BATS_RUN_TMPDIR/test/1/config/obsidian"
    local stray=$!
    # And one that looks like a human's: the real config dir, no bats tmpdir.
    _launch_fake "$fake" --user-data-dir="$HOME/.config/obsidian"
    local human=$!
    sleep 0.3

    local found
    found="$(_gui_guard_test_shaped_processes)"

    kill -9 "$stray" "$human" 2>/dev/null || true

    printf '%s' "$found" | grep -q "$stray" || {
        printf 'detector missed a test-shaped process (pid %s):\n%s\n' "$stray" "$found" >&2
        return 1
    }
    # The half that matters more: it must never sweep up something a human
    # opened. Killing the developer's editor would be a worse bug than the one
    # this guard exists to fix.
    if printf '%s' "$found" | grep -q "$human"; then
        printf 'detector matched a human-shaped process (pid %s) — it would kill a real editor\n' "$human" >&2
        return 1
    fi
}

# Under `bats --jobs`, and on a box where several agent sessions run suites at
# once, another run's fixtures are alive while this one tears down. The detector
# reports and kills what it matches, so it must match only this run's.
# bats test_tags=os-sensitive
@test "guard: the stray detector ignores a test-shaped process from another bats run" {
    _guard_is_active || skip "setup_suite did not run (single-file invocation); the guard is inactive here"
    command -v pgrep >/dev/null 2>&1 || skip "pgrep not installed"

    # shellcheck source=setup_suite.bash disable=SC1091
    . "$REPO_ROOT/tests/setup_suite.bash"

    local fake="$BATS_TEST_TMPDIR/obsidian"
    printf '#!/bin/sh\nsleep 30\n' > "$fake"
    chmod +x "$fake"
    _launch_fake "$fake" --user-data-dir="$BATS_RUN_TMPDIR/test/1/config/obsidian"
    local ours=$!
    _launch_fake "$fake" --user-data-dir="/tmp/bats-run-OTHER$$/test/1/config/obsidian"
    local theirs=$!
    sleep 0.3

    local found unscoped
    found="$(_gui_guard_test_shaped_processes)"
    # With no run tmpdir to scope to, it must match nothing rather than
    # everything: an empty prefix would sweep up a human's editor.
    unscoped="$(BATS_RUN_TMPDIR='' _gui_guard_test_shaped_processes)"

    kill -9 "$ours" "$theirs" 2>/dev/null || true

    printf '%s' "$found" | grep -q "^$ours " || { printf 'missed this run'"'"'s process (pid %s):\n%s\n' "$ours" "$found" >&2; return 1; }
    if printf '%s' "$found" | grep -q "^$theirs "; then
        printf 'matched another run'"'"'s process (pid %s); teardown would kill it\n' "$theirs" >&2
        return 1
    fi
    [ -z "$unscoped" ] || { printf 'matched with BATS_RUN_TMPDIR empty:\n%s\n' "$unscoped" >&2; return 1; }
}

