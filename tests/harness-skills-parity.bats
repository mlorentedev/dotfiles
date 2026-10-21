#!/usr/bin/env bats
# Parity oracle for `dotf harness skills` (CLI-026): the Go skill deploy must
# write what scripts/compile-harness.sh --deploy writes, over the committed
# records, before either shell twin is retired. Both engines deploy into their
# own throwaway HOME from one fixture checkout whose manifest carries only the
# skills block, so the bash side needs neither dotf nor a vault, and both see
# the same PATH, so requires_command gates the same targets on each side. Each
# HOME is seeded with the copilot instructions file and its empty GENERATED
# slot, as `dotf harness instructions` leaves it, so the catalog is compared too.
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
    jq '{skills: .skills}' "$REPO/harness/manifest.json" > "$FX/harness/manifest.json"
    CATALOG="$(jq -r '.skills.catalog.file' "$FX/harness/manifest.json")"
    local side
    for side in bash go; do
        mkdir -p "$(dirname "$BATS_TEST_TMPDIR/$side/$CATALOG")"
        printf '# Copilot\n\n<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n\nAfter the slot.\n' \
            > "$BATS_TEST_TMPDIR/$side/$CATALOG"
    done
}

deploy_both() {
    HOME="$BATS_TEST_TMPDIR/bash" HARNESS_REPO_ROOT="$FX" LC_ALL=C \
        bash "$FX/scripts/compile-harness.sh" --deploy >/dev/null
    "$DOTF_BIN" harness skills --repo-root "$FX" --home "$BATS_TEST_TMPDIR/go" >/dev/null
}

@test "skills parity: every output but the prompts and the catalog is byte-identical to the bash oracle" {
    deploy_both
    run diff -r -x prompts -x "$(basename "$CATALOG")" "$BATS_TEST_TMPDIR/bash" "$BATS_TEST_TMPDIR/go"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" | head -40; false; }
    # A vacuous pass on two empty trees proves nothing.
    [ -n "$(find "$BATS_TEST_TMPDIR/go/.claude/skills" -name SKILL.md | head -1)" ]
}

@test "skills parity: the catalog lists what the oracle lists, in byte order" {
    deploy_both
    # Declared divergence: the shell's `*/` glob compared `name/`, so
    # test-driven-development/ sorted before test/ even under LC_ALL=C; Go
    # walks the names in byte order. The bullets are compared as a set, and the
    # sha they hash to is left out with them.
    run diff <(sed 's/(sha256:[0-9a-f]*)//' "$BATS_TEST_TMPDIR/bash/$CATALOG" | LC_ALL=C sort) \
             <(sed 's/(sha256:[0-9a-f]*)//' "$BATS_TEST_TMPDIR/go/$CATALOG" | LC_ALL=C sort)
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
    # The region was filled, not left as the empty slot both sides started with.
    [ "$(grep -c '^- \*\*' "$BATS_TEST_TMPDIR/go/$CATALOG")" -gt 0 ]
    run diff <(grep '^- \*\*' "$BATS_TEST_TMPDIR/go/$CATALOG") <(grep '^- \*\*' "$BATS_TEST_TMPDIR/go/$CATALOG" | LC_ALL=C sort)
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
}

@test "skills parity: prompts differ from the oracle only by the body rules it dropped" {
    deploy_both
    local f name differ=0
    # The same prompt files on both sides, and some: the loop below walks the
    # bash side only, and the first test excludes prompts from its diff.
    run diff <(ls "$BATS_TEST_TMPDIR/bash/.gemini/prompts") <(ls "$BATS_TEST_TMPDIR/go/.gemini/prompts")
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
    [ -n "$(ls "$BATS_TEST_TMPDIR/bash/.gemini/prompts")" ]
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
