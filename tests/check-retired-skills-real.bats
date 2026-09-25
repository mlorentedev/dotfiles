#!/usr/bin/env bats
# SKILL-001: the retired-skill check reads its targets from harness/manifest.json.
# Every case here drives the real jq; check-retired-skills.bats holds the one
# case that wraps it (tests/stub-real-pairing.bats pairs the two).
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

# SKILL-001 review, round 3: a check that looked at nothing must not read as
# clean. Each case below leaves a copy of ours in place, so an exit 0 would be
# a false green.
seed_marked_audit() {
    mkdir -p "$FAKEHOME/.claude/skills/audit"
    printf -- '---\ngenerated: true\ngenerated_from: 00_meta/skills/audit/SKILL.md\n---\n' \
        > "$FAKEHOME/.claude/skills/audit/SKILL.md"
}

# copy_repo lays the script and a manifest out in a throwaway repo, so a case can
# break the manifest without touching the real one.
copy_repo() {
    COPY="$(mktemp -d)"
    mkdir -p "$COPY/scripts" "$COPY/harness"
    cp "$SCRIPT" "$COPY/scripts/"
    cp "$MANIFEST" "$COPY/harness/manifest.json"
}

@test "SKILL-001 check: a manifest with no deploy target fails instead of passing" {
    seed_marked_audit
    copy_repo
    jq '.skills.deploy = []' "$MANIFEST" > "$COPY/harness/manifest.json"
    run env HOME="$FAKEHOME" "$COPY/scripts/check-retired-skills.sh" audit
    rm -rf "$COPY"
    [ "$status" -eq 2 ]
}

@test "SKILL-001 check: an unreadable manifest fails instead of passing" {
    [ "$(id -u)" -ne 0 ] || skip "root reads a mode-000 file"
    seed_marked_audit
    copy_repo
    chmod 000 "$COPY/harness/manifest.json"
    run env HOME="$FAKEHOME" "$COPY/scripts/check-retired-skills.sh" audit
    chmod 600 "$COPY/harness/manifest.json"
    rm -rf "$COPY"
    [ "$status" -eq 2 ]
}

@test "SKILL-001 check: a rendered agent definition that names a retired skill is found" {
    dir="$(jq -r '.agents.deploy[]? | select(.render == "agent-md") | .dir' "$MANIFEST" | head -1)"
    [ -n "$dir" ] || skip "the manifest renders no agent definitions"
    mkdir -p "$FAKEHOME/$dir"
    printf 'MUST consume: [adversarial-review, audit]\n' > "$FAKEHOME/$dir/reviewer.md"
    run env HOME="$FAKEHOME" "$SCRIPT" audit
    [ "$status" -eq 1 ]
}
