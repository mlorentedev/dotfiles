#!/usr/bin/env bats
# Parity oracle for `dotf harness check` (CLI-026): the Go drift gate must
# reach the verdict scripts/compile-harness.sh --check reaches, with the same
# lines, over the committed checkout and over each kind of drift, before CI
# and doctor move onto it. Both engines check one fixture checkout.
#
# The region diff itself is left out (indented lines): the shell prints a
# unified diff, Go names the first differing line. The lines are compared
# sorted, because the shell walks `*/` (lesson 505) and Go walks byte order.
#
# This file goes with compile-harness.sh: it has no oracle once the script is
# deleted.

load 'lib/dotf-bin'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    BIN_DIR="${BATS_FILE_TMPDIR:-/tmp/bats_check_parity_$$}/bin"
    dotf_bin_resolve "$BIN_DIR/dotf"
    command -v jq >/dev/null 2>&1 || skip "jq is required by the bash oracle"

    FX="$BATS_TEST_TMPDIR/fx"
    mkdir -p "$FX/scripts"
    cp -R "$REPO/harness" "$REPO/ai" "$FX/"
    cp "$REPO/AGENTS.md" "$FX/"
    cp "$REPO/scripts/compile-harness.sh" "$FX/scripts/"
}

# check_both runs both engines on the fixture and fails unless they agree on
# the exit status, the stdout lines and the unindented stderr lines. It leaves
# the shared exit status in $status.
check_both() {
    local out="$BATS_TEST_TMPDIR/out" bs gs
    mkdir -p "$out"
    HARNESS_REPO_ROOT="$FX" LC_ALL=C bash "$FX/scripts/compile-harness.sh" --check \
        >"$out/bash.1" 2>"$out/bash.2" && bs=0 || bs=$?
    "$DOTF_BIN" harness check --repo-root "$FX" >"$out/go.1" 2>"$out/go.2" && gs=0 || gs=$?
    if [ "$bs" -ne "$gs" ]; then
        printf 'exit: bash %s, go %s\n' "$bs" "$gs"
        cat "$out/bash.2" "$out/go.2"
        return 1
    fi
    local s
    for s in 1 2; do
        grep -v '^    ' "$out/bash.$s" | sed 's,//,/,g' | LC_ALL=C sort >"$out/b" || true
        grep -v '^    ' "$out/go.$s" | grep -v '^Error: ' | LC_ALL=C sort >"$out/g" || true
        diff "$out/b" "$out/g" || { printf 'stream %s differs (< bash, > go)\n' "$s"; return 1; }
    done
    status="$bs"
}

@test "check parity: the committed checkout passes on both" {
    check_both
    [ "$status" -eq 0 ]
}

@test "check parity: a drifted, a missing and an unmarked region fail alike" {
    printf 'an edit the targets never received\n' >>"$FX/harness/enforced/no-attribution.md"
    check_both
    [ "$status" -eq 1 ]
    rm "$FX/harness/enforced/pr-stewardship.md"
    check_both
    [ "$status" -eq 1 ]
    perl -0pi -e 's/^(<!-- BEGIN HARNESS GENERATED.*)$/$1\n$1/m' "$FX/ai/orca/ORCA.md"
    check_both
    [ "$status" -eq 1 ]
}

@test "check parity: an END line with a tail reads as drift on both" {
    perl -pi -e 's/^(<!-- END HARNESS GENERATED -->)$/$1 trailing/' "$FX/AGENTS.md"
    check_both
    [ "$status" -eq 1 ]
}

@test "check parity: an enforced id that reaches a surface neither injected nor opted out is a gap" {
    jq '.targets[1].inject -= ["english-only"]' "$FX/harness/manifest.json" >"$FX/m" && mv "$FX/m" "$FX/harness/manifest.json"
    check_both
    [ "$status" -eq 1 ]
}

@test "check parity: a skill or agent record without its required frontmatter fails alike" {
    perl -ni -e 'print unless /^owner:/' "$FX/harness/skills/handoff/SKILL.md"
    check_both
    [ "$status" -eq 1 ]
    printf 'no frontmatter\n' >"$FX/harness/skills/test/SKILL.md"
    printf -- '---\nname: x\n' >"$FX/harness/skills/spec/SKILL.md"
    perl -ni -e 'print unless /^kind:/' "$FX/harness/agents/builder/AGENT.md"
    check_both
    [ "$status" -eq 1 ]
}
