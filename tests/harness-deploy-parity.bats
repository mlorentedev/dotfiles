#!/usr/bin/env bats
# Parity oracle for `dotf harness deploy` (CLI-026): the composed Go deploy
# must leave $HOME as scripts/compile-harness.sh --deploy leaves it, over the
# full committed manifest and records, before setup and converge are rewired
# onto it. Both engines deploy into their own throwaway HOME from one fixture
# checkout. The shell delegates instructions and presence to `dotf`, so the
# binary under test is on its PATH under that name; for those two surfaces
# this proves the ordering, and the per-surface oracles prove the rest.
#
# Two declared differences are left out of the byte diff and compared in the
# skills oracle instead: agy's prompts (the shell dropped the rules in their
# bodies) and the copilot catalog's bullet order.
#
# This file goes with compile-harness.sh: it has no oracle once the script is
# deleted.

load 'lib/dotf-bin'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    BIN_DIR="${BATS_FILE_TMPDIR:-/tmp/bats_deploy_parity_$$}/bin"
    dotf_bin_resolve "$BIN_DIR/dotf"
    command -v jq >/dev/null 2>&1 || skip "jq is required by the bash oracle"

    FX="$BATS_TEST_TMPDIR/fx"
    mkdir -p "$FX/scripts" "$BATS_TEST_TMPDIR/bash" "$BATS_TEST_TMPDIR/go" "$BATS_TEST_TMPDIR/path"
    cp -R "$REPO/harness" "$REPO/ai" "$FX/"
    cp "$REPO/AGENTS.md" "$FX/"
    cp "$REPO/scripts/compile-harness.sh" "$FX/scripts/"
    ln -s "$DOTF_BIN" "$BATS_TEST_TMPDIR/path/dotf"
}

deploy_both() {
    HOME="$BATS_TEST_TMPDIR/bash" HARNESS_REPO_ROOT="$FX" LC_ALL=C PATH="$BATS_TEST_TMPDIR/path:$PATH" \
        bash "$FX/scripts/compile-harness.sh" --deploy >/dev/null 2>&1
    PATH="$BATS_TEST_TMPDIR/path:$PATH" "$DOTF_BIN" harness deploy --repo-root "$FX" --home "$BATS_TEST_TMPDIR/go" >/dev/null 2>&1
}

@test "deploy parity: every deployed file but the declared differences is byte-identical" {
    deploy_both
    run diff -r -x prompts -x copilot-instructions.md "$BATS_TEST_TMPDIR/bash" "$BATS_TEST_TMPDIR/go"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" | head -40; false; }
    # Every surface reached the Go HOME: not two trees that both lack one.
    local f
    for f in .claude/CLAUDE.md .claude/agents/builder.md .claude/skills/handoff/SKILL.md \
             .gemini/GEMINI.md .codex/AGENTS.md; do
        [ -s "$BATS_TEST_TMPDIR/go/$f" ] || { printf 'missing: %s\n' "$f"; false; }
    done
    grep -q 'BEGIN HARNESS AGENT-PRESENCE' "$BATS_TEST_TMPDIR/go/.claude/CLAUDE.md"
}

@test "deploy parity: the copilot instructions differ only by the catalog's bullet order" {
    deploy_both
    local f=.copilot/copilot-instructions.md
    run diff <(sed 's/(sha256:[0-9a-f]*) — skill catalog/(sha) — skill catalog/' "$BATS_TEST_TMPDIR/bash/$f" | LC_ALL=C sort) \
             <(sed 's/(sha256:[0-9a-f]*) — skill catalog/(sha) — skill catalog/' "$BATS_TEST_TMPDIR/go/$f" | LC_ALL=C sort)
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" | head -20; false; }
}

@test "deploy parity: a second Go deploy changes nothing" {
    deploy_both
    run env PATH="$BATS_TEST_TMPDIR/path:$PATH" "$DOTF_BIN" harness deploy --repo-root "$FX" --home "$BATS_TEST_TMPDIR/go"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" | tail -20; false; }
    printf '%s\n' "$output" | grep -q '^\[deploy\] skills: 0 deployed, '
    printf '%s\n' "$output" | grep -q '^\[deploy\] agents: 0 deployed, '
    ! printf '%s\n' "$output" | grep -E '^\[deploy\] (instructions|catalog|doctrine|presence) ->'
}
