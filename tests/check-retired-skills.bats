#!/usr/bin/env bats
# SKILL-001 review, round 3: the one case that shadows jq on PATH. The shadow
# runs the real jq and ends each line with CR, as the winget build of jq does on
# Windows, where the check used to read every path with a stray \r and find
# nothing. Every other case drives the real jq, in check-retired-skills-real.bats.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SCRIPT="$REPO/scripts/check-retired-skills.sh"
    MANIFEST="$REPO/harness/manifest.json"
    FAKEHOME="$(mktemp -d)"
}

teardown() { rm -rf "$FAKEHOME"; }

seed_marked_audit() {
    mkdir -p "$FAKEHOME/.claude/skills/audit"
    printf -- '---\ngenerated: true\ngenerated_from: 00_meta/skills/audit/SKILL.md\n---\n' \
        > "$FAKEHOME/.claude/skills/audit/SKILL.md"
}

@test "SKILL-001 check: a jq that ends its lines with CRLF still finds a copy we rendered" {
    seed_marked_audit
    real="$(type -P jq)"
    mkdir -p "$FAKEHOME/bin"
    cat > "$FAKEHOME/bin/jq" <<EOF
#!/usr/bin/env bash
"$real" "\$@" | sed 's/\$/\r/'
EOF
    chmod +x "$FAKEHOME/bin/jq"
    run env HOME="$FAKEHOME" PATH="$FAKEHOME/bin:$PATH" "$SCRIPT" audit
    [ "$status" -eq 1 ]
}
