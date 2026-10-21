#!/usr/bin/env bats
# Parity oracle for `dotf harness agents` (CLI-026): the Go agent deploy must
# write what scripts/compile-harness.sh --deploy writes for the agents block,
# over the committed records, before the script is retired. Both engines deploy
# into their own throwaway HOME from one fixture checkout whose manifest
# carries the agents block without presence, so the bash side renders agents
# only. The bash side resolves tiers and capabilities by calling `dotf`, so the
# binary under test is put on its PATH under that name: both sides then read
# the same maps through the same resolver, and the oracle compares the render.
#
# Declared differences, neither of which the committed records exercise: Go
# matches `targets:` by exact name, and renders a CRLF record as its LF bytes.
#
# This file goes with compile-harness.sh: it has no oracle once the script is
# deleted.

load 'lib/dotf-bin'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    BIN_DIR="${BATS_FILE_TMPDIR:-/tmp/bats_agents_parity_$$}/bin"
    dotf_bin_resolve "$BIN_DIR/dotf"
    command -v jq >/dev/null 2>&1 || skip "jq is required by the bash oracle"

    FX="$BATS_TEST_TMPDIR/fx"
    mkdir -p "$FX/harness" "$FX/scripts" "$BATS_TEST_TMPDIR/bash" "$BATS_TEST_TMPDIR/go" "$BATS_TEST_TMPDIR/path"
    cp -R "$REPO/harness/agents" "$FX/harness/"
    cp "$REPO/harness/model-map.json" "$REPO/harness/model-map.schema.json" \
       "$REPO/harness/capability-map.json" "$REPO/harness/capability-map.schema.json" "$FX/harness/"
    cp "$REPO/scripts/compile-harness.sh" "$FX/scripts/"
    jq '{agents: (.agents | del(.presence))}' "$REPO/harness/manifest.json" > "$FX/harness/manifest.json"
    ln -s "$DOTF_BIN" "$BATS_TEST_TMPDIR/path/dotf"
}

deploy_both() {
    HOME="$BATS_TEST_TMPDIR/bash" HARNESS_REPO_ROOT="$FX" LC_ALL=C PATH="$BATS_TEST_TMPDIR/path:$PATH" \
        bash "$FX/scripts/compile-harness.sh" --deploy >/dev/null
    "$DOTF_BIN" harness agents --repo-root "$FX" --home "$BATS_TEST_TMPDIR/go" >/dev/null
}

@test "agents parity: every agent file is byte-identical to the bash oracle" {
    deploy_both
    run diff -r "$BATS_TEST_TMPDIR/bash" "$BATS_TEST_TMPDIR/go"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" | head -40; false; }
    # A vacuous pass on two empty trees proves nothing, and neither does a pair
    # of renders that both lost the resolved lines.
    [ -n "$(find "$BATS_TEST_TMPDIR/go/.claude/agents" -name '*.md' | head -1)" ]
    grep -q '^model: ' "$BATS_TEST_TMPDIR/go/.claude/agents/builder.md"
    grep -q '^tools: ' "$BATS_TEST_TMPDIR/go/.claude/agents/builder.md"
}

@test "agents parity: a second Go run on the deployed tree changes nothing" {
    deploy_both
    run "$DOTF_BIN" harness agents --repo-root "$FX" --home "$BATS_TEST_TMPDIR/go"
    [ "$status" -eq 0 ]
    printf '%s\n' "$output" | grep -q '^\[deploy\] agents: 0 deployed, [1-9][0-9]* current$'
}
