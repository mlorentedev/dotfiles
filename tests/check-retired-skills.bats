#!/usr/bin/env bats
# SKILL-001: the retired-skill check reads its targets from harness/manifest.json.
# Each test walks the manifest too and seeds every declared target in turn, so a
# harness added there is tested without editing this file, and a render type the
# script does not know fails here instead of passing unchecked.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SCRIPT="$REPO/scripts/check-retired-skills.sh"
    MANIFEST="$REPO/harness/manifest.json"
    FAKEHOME="$(mktemp -d)"
}

teardown() { rm -rf "$FAKEHOME"; }

@test "SKILL-001 check: a HOME with nothing retired passes" {
    run env HOME="$FAKEHOME" "$SCRIPT" audit enrich-us
    [ "$status" -eq 0 ]
}

@test "SKILL-001 check: every instruction file the manifest declares is read" {
    while IFS= read -r rel; do
        rm -rf "$FAKEHOME" && mkdir -p "$FAKEHOME/$(dirname "$rel")"
        printf 'MUST consume: [adversarial-review, audit]\n' > "$FAKEHOME/$rel"
        run env HOME="$FAKEHOME" "$SCRIPT" audit
        [ "$status" -eq 1 ] || { echo "not read: $rel"; return 1; }
    done < <(jq -r '[.agents.presence[].file, .skills.catalog.file, .doctrine.deploy[].file] | unique[]' "$MANIFEST")
}

@test "SKILL-001 check: every render target reports a copy we rendered and ignores a foreign one" {
    while IFS=$'\t' read -r render dir; do
        for kind in ours foreign; do
            rm -rf "$FAKEHOME"
            case "$render" in
                skill) f="$FAKEHOME/$dir/audit/SKILL.md" ;;
                *) f="$FAKEHOME/$dir/audit.md" ;;
            esac
            mkdir -p "$(dirname "$f")"
            if [ "$kind" = foreign ]; then
                printf -- '---\nname: audit\n---\nanother tool installed this\n' > "$f"
            elif [ "$render" = prompt ]; then
                printf '<!-- generated: true; from: 00_meta/skills/audit/SKILL.md; sha256:0 -->\n' > "$f"
            else
                printf -- '---\ngenerated: true\ngenerated_from: 00_meta/skills/audit/SKILL.md\n---\n' > "$f"
            fi
            want=1
            [ "$kind" = ours ] || want=0
            run env HOME="$FAKEHOME" "$SCRIPT" audit
            [ "$status" -eq "$want" ] || { echo "$render $dir $kind: exit $status, want $want"; return 1; }
        done
    done < <(jq -r '.skills.deploy[] | [.render, .dir] | @tsv' "$MANIFEST")
}
