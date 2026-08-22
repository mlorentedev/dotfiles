#!/usr/bin/env bats
# Regression guard: compile-harness.sh must resolve its repo/deploy root from the
# script's own location (not `git rev-parse`), so --check / --deploy work from the
# NON-GIT deploy copy (~/.dotfiles, ADR-012 copy-deploy) — that is where setup and
# healthcheck (section 12) invoke them. Requiring a git repo there false-failed.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    TMP="$(mktemp -d)"
}

teardown() { rm -rf "$TMP"; }

@test "compile-harness --check works from a non-git copy (root resolved from SCRIPT_DIR)" {
    # Assemble a complete, NON-git copy of exactly what --check reads.
    mkdir -p "$TMP/scripts" "$TMP/ai/claude" "$TMP/ai/orca"
    cp "$REPO/scripts/compile-harness.sh" "$TMP/scripts/"
    cp "$REPO/AGENTS.md" "$TMP/"
    cp "$REPO/ai/claude/CLAUDE.md" "$TMP/ai/claude/"
    cp "$REPO/ai/orca/ORCA.md" "$TMP/ai/orca/"
    cp -r "$REPO/harness" "$TMP/"
    [ ! -d "$TMP/.git" ]   # genuinely not a git repo

    # Run from an unrelated CWD to prove root resolution is CWD-independent.
    run bash -c "cd / && '$TMP/scripts/compile-harness.sh' --check"
    [ "$status" -eq 0 ]
    echo "$output" | grep -q 'no harness drift'
}

@test "compile-harness without its harness/ dir fails gracefully (no crash)" {
    mkdir -p "$TMP/scripts"
    cp "$REPO/scripts/compile-harness.sh" "$TMP/scripts/"
    run bash -c "cd / && '$TMP/scripts/compile-harness.sh' --check"
    [ "$status" -ne 0 ]   # missing manifest -> clean non-zero, not a stack trace
}
