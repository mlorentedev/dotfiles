#!/usr/bin/env bats
# Parity oracle for `dotf harness skills` (CLI-026): the Go skill deploy must
# write what scripts/compile-harness.sh --deploy writes, over the committed
# records, before either shell twin is retired. Both engines deploy into their
# own throwaway HOME from one fixture checkout whose manifest carries only the
# skills block, so the bash side needs neither dotf nor a vault, and both see
# the same PATH, so requires_command gates the same targets on each side.
#
# One declared difference: the shell's prompt render dropped every `---` line,
# so a rule in a prompt's body vanished; Go drops only the two frontmatter
# fences. Prompts are therefore compared with `---` lines removed, and Go's
# prompts are held to keeping the body's rules.
#
# This file goes with compile-harness.sh: it has no oracle once the script is
# deleted.

load 'lib/dotf-bin'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    BIN_DIR="${BATS_FILE_TMPDIR:-/tmp/bats_skills_parity_$$}/bin"
    dotf_bin_resolve "$BIN_DIR/dotf"
    command -v jq >/dev/null 2>&1 || skip "jq is required by the bash oracle"

    FX="$BATS_TEST_TMPDIR/fx"
    mkdir -p "$FX/harness" "$FX/scripts" "$BATS_TEST_TMPDIR/bash" "$BATS_TEST_TMPDIR/go"
    cp -R "$REPO/harness/skills" "$FX/harness/"
    cp "$REPO/scripts/compile-harness.sh" "$FX/scripts/"
    jq '{skills: (.skills | del(.catalog))}' "$REPO/harness/manifest.json" > "$FX/harness/manifest.json"
}

deploy_both() {
    HOME="$BATS_TEST_TMPDIR/bash" HARNESS_REPO_ROOT="$FX" LC_ALL=C \
        bash "$FX/scripts/compile-harness.sh" --deploy >/dev/null
    "$DOTF_BIN" harness skills --repo-root "$FX" --home "$BATS_TEST_TMPDIR/go" >/dev/null
}

@test "skills parity: every output but the prompts is byte-identical to the bash oracle" {
    deploy_both
    run diff -r -x prompts "$BATS_TEST_TMPDIR/bash" "$BATS_TEST_TMPDIR/go"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" | head -40; false; }
    # A vacuous pass on two empty trees proves nothing.
    [ -n "$(find "$BATS_TEST_TMPDIR/go/.claude/skills" -name SKILL.md | head -1)" ]
}

@test "skills parity: prompts differ from the oracle only by the body rules it dropped" {
    deploy_both
    local f name differ=0
    for f in "$BATS_TEST_TMPDIR/bash/.gemini/prompts/"*.md; do
        name="$(basename "$f")"
        if ! diff <(grep -vx -- '---' "$f") <(grep -vx -- '---' "$BATS_TEST_TMPDIR/go/.gemini/prompts/$name") >/dev/null; then
            printf 'prompt differs beyond its rules: %s\n' "$name"
            differ=1
        fi
    done
    [ "$differ" -eq 0 ]
}

@test "skills parity: a prompt keeps every rule its record's body carries" {
    deploy_both
    local rec body_rules prompt_rules
    rec="$FX/harness/skills/adversarial-review/SKILL.md"
    body_rules="$(awk '/^---[[:space:]]*$/{n++; if (n > 2) c++} END{print c+0}' "$rec")"
    prompt_rules="$(grep -cx -- '---' "$BATS_TEST_TMPDIR/go/.gemini/prompts/adversarial-review.md" || true)"
    [ "$body_rules" -gt 0 ]
    [ "$prompt_rules" -eq "$body_rules" ]
}

@test "skills parity: auxiliary files keep the executable bit on both sides" {
    deploy_both
    run diff <(cd "$BATS_TEST_TMPDIR/bash" && find . -type f -perm -u+x | LC_ALL=C sort) \
             <(cd "$BATS_TEST_TMPDIR/go" && find . -type f -perm -u+x | LC_ALL=C sort)
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
}

@test "skills parity: a second Go run on the deployed tree changes nothing" {
    deploy_both
    run "$DOTF_BIN" harness skills --repo-root "$FX" --home "$BATS_TEST_TMPDIR/go"
    [ "$status" -eq 0 ]
    printf '%s\n' "$output" | grep -q '^\[deploy\] skills: 0 deployed, [1-9][0-9]* current, 0 pruned$'
}
