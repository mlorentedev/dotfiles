#!/usr/bin/env bats
# Parity oracle for `dotf harness doctrine` (CLI-026): the Go doctrine deploy
# must write what scripts/compile-harness.sh --deploy writes into each capped
# rules file, over the committed enforced records and agent records, before the
# script is retired. Both engines deploy into their own throwaway HOME from one
# fixture checkout whose manifest carries the doctrine block and an agents
# block with no deploy targets and no presence, so the bash side writes the
# doctrine only. The bash side renders the roster by calling `dotf`, so the
# binary under test is on its PATH under that name.
#
# Three starting states, each seeded identically on both sides: no rules file,
# a file with a region and user content around it, and a file with no region.
#
# This file goes with compile-harness.sh: it has no oracle once the script is
# deleted.

load 'lib/dotf-bin'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    BIN_DIR="${BATS_FILE_TMPDIR:-/tmp/bats_doctrine_parity_$$}/bin"
    dotf_bin_resolve "$BIN_DIR/dotf"
    command -v jq >/dev/null 2>&1 || skip "jq is required by the bash oracle"

    FX="$BATS_TEST_TMPDIR/fx"
    mkdir -p "$FX/harness" "$FX/scripts" "$BATS_TEST_TMPDIR/bash" "$BATS_TEST_TMPDIR/go" "$BATS_TEST_TMPDIR/path"
    cp -R "$REPO/harness/agents" "$REPO/harness/enforced" "$FX/harness/"
    cp "$REPO/scripts/compile-harness.sh" "$FX/scripts/"
    jq '{vault_subpath, agents: {vault_subpath: .agents.vault_subpath, record_dir: .agents.record_dir, deploy: []}, doctrine}' \
        "$REPO/harness/manifest.json" > "$FX/harness/manifest.json"
    ln -s "$DOTF_BIN" "$BATS_TEST_TMPDIR/path/dotf"
}

# seed <relative file> <content>: the same starting file on both sides.
seed() {
    local side
    for side in bash go; do
        mkdir -p "$(dirname "$BATS_TEST_TMPDIR/$side/$1")"
        printf '%b' "$2" > "$BATS_TEST_TMPDIR/$side/$1"
    done
}

deploy_both() {
    HOME="$BATS_TEST_TMPDIR/bash" HARNESS_REPO_ROOT="$FX" LC_ALL=C PATH="$BATS_TEST_TMPDIR/path:$PATH" \
        bash "$FX/scripts/compile-harness.sh" --deploy >/dev/null 2>&1
    "$DOTF_BIN" harness doctrine --repo-root "$FX" --home "$BATS_TEST_TMPDIR/go" >/dev/null 2>&1
}

assert_same_trees() {
    run diff -r "$BATS_TEST_TMPDIR/bash" "$BATS_TEST_TMPDIR/go"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" | head -40; false; }
    # Not two empty or two roster-less payloads.
    grep -q '^## Non-negotiable rules (harness-enforced)$' "$BATS_TEST_TMPDIR/go/.gemini/GEMINI.md"
    grep -q 'MUST consume' "$BATS_TEST_TMPDIR/go/.gemini/GEMINI.md"
}

@test "doctrine parity: absent rules files are started and filled as the oracle does" {
    deploy_both
    assert_same_trees
}

@test "doctrine parity: a region is replaced in place and the user's lines are kept" {
    seed .gemini/GEMINI.md '# Mine\n\n> Cross-agent doctrine. The marked region is generated \xe2\x80\x94 edit the vault pattern and re-run setup.\n\n<!-- BEGIN HARNESS GENERATED (sha256:0) -->\nold\n<!-- END HARNESS GENERATED -->\n\nafter\n'
    seed .codex/AGENTS.md '# Codex\n\n<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n'
    deploy_both
    assert_same_trees
    grep -qx 'after' "$BATS_TEST_TMPDIR/go/.gemini/GEMINI.md"
}

@test "doctrine parity: a file with no region gets one appended" {
    seed .gemini/GEMINI.md '# Shared with another tool\nno trailing newline'
    seed .codex/AGENTS.md '# Codex\n'
    deploy_both
    assert_same_trees
}

@test "doctrine parity: a second Go run on the deployed tree changes nothing" {
    deploy_both
    run "$DOTF_BIN" harness doctrine --repo-root "$FX" --home "$BATS_TEST_TMPDIR/go"
    [ "$status" -eq 0 ]
    [ "$(printf '%s\n' "$output" | grep -c '^\[deploy\] doctrine current: ')" -eq 2 ]
}
