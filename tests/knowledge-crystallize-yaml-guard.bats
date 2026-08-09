#!/usr/bin/env bats
# BUG-062 (#857): knowledge-crystallize anchors every marker at column 0, so a
# MEMORY.md whose body lives inside a YAML `content: |` block scalar matches
# nothing, falls through to the bare append, and lands text OUTSIDE the block —
# breaking the handoff invariant, duplicating the date stamp, and making the
# file stop parsing as YAML while the script prints [SUCCESS] twice.
#
# Until dotf vault crystallize owns this path (#490), the script must REFUSE
# that shape rather than corrupt it. These tests pin the refusal.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
}

teardown() {
    [ -n "${FAKE_HOME:-}" ] && rm -rf "$FAKE_HOME"
    return 0
}

# $1 = indent width. The indent is NOT fixed across projects: pollex indents six
# spaces, hive four. Any guard keyed to a literal width passes on one and fails
# on the other, so both are exercised.
_setup_yaml_project() {
    local indent_width="$1" pad
    pad=$(printf '%*s' "$indent_width" '')
    FAKE_HOME="$(mktemp -d)"
    FAKE_PROJECT="$FAKE_HOME/Projects/demo"
    mkdir -p "$FAKE_PROJECT"
    local encoded
    encoded="$(printf '%s' "$FAKE_PROJECT" | tr '/' '-')"
    FAKE_MEM_DIR="$FAKE_HOME/.claude/projects/$encoded/memory"
    mkdir -p "$FAKE_MEM_DIR"
    {
        printf -- '---\n'
        printf 'id: "demo-memory"\n'
        printf 'type: memory\n'
        printf 'content: |\n'
        printf '%s# demo - Project Memory\n' "$pad"
        printf '%s\n' ""
        printf '%s## Last Crystallized: 2026-06-13\n' "$pad"
        printf '%s\n' ""
        printf '%s## Session Handoff\n' "$pad"
        printf '%s> Updated: 2026-06-13\n' "$pad"
    } > "$FAKE_MEM_DIR/MEMORY.md"
}

_setup_plain_project() {
    FAKE_HOME="$(mktemp -d)"
    FAKE_PROJECT="$FAKE_HOME/Projects/demo"
    mkdir -p "$FAKE_PROJECT"
    local encoded
    encoded="$(printf '%s' "$FAKE_PROJECT" | tr '/' '-')"
    FAKE_MEM_DIR="$FAKE_HOME/.claude/projects/$encoded/memory"
    mkdir -p "$FAKE_MEM_DIR"
    cat > "$FAKE_MEM_DIR/MEMORY.md" <<'EOF'
# Memory Index — demo

- [Some memory](some-memory.md) — hook

## Session Handoff

> Updated: 2026-01-01
**Last task:** something
EOF
}

@test "yaml-guard: refuses a block-scalar MEMORY.md indented four spaces (hive)" {
    _setup_yaml_project 4
    HOME="$FAKE_HOME" run bash "$SCRIPTS_DIR/knowledge-crystallize.sh" "$FAKE_PROJECT"
    [ "$status" -ne 0 ]
}

@test "yaml-guard: refuses a block-scalar MEMORY.md indented six spaces (pollex)" {
    # The indent width varies per project, so the guard must derive it, never
    # assume it. This case is what a literal-width guard would let through.
    _setup_yaml_project 6
    HOME="$FAKE_HOME" run bash "$SCRIPTS_DIR/knowledge-crystallize.sh" "$FAKE_PROJECT"
    [ "$status" -ne 0 ]
}

@test "yaml-guard: leaves the refused file byte-identical" {
    _setup_yaml_project 4
    local before after
    before=$(md5sum < "$FAKE_MEM_DIR/MEMORY.md")
    HOME="$FAKE_HOME" run bash "$SCRIPTS_DIR/knowledge-crystallize.sh" "$FAKE_PROJECT"
    after=$(md5sum < "$FAKE_MEM_DIR/MEMORY.md")
    [ "$before" = "$after" ]
}

@test "yaml-guard: the refused file still parses as YAML" {
    # Defect 3 of #857 — the damaging one. Appended text landed outside the
    # block scalar and the file stopped loading for any YAML consumer.
    if ! python3 -c "import yaml" 2>/dev/null; then
        skip "python3 yaml not available"
    fi
    _setup_yaml_project 4
    HOME="$FAKE_HOME" run bash "$SCRIPTS_DIR/knowledge-crystallize.sh" "$FAKE_PROJECT"
    python3 -c "import yaml,sys; yaml.safe_load(open(sys.argv[1]))" "$FAKE_MEM_DIR/MEMORY.md"
}

@test "yaml-guard: the refusal names the file and points at the tracking issue" {
    _setup_yaml_project 4
    HOME="$FAKE_HOME" run bash "$SCRIPTS_DIR/knowledge-crystallize.sh" "$FAKE_PROJECT"
    [[ "$output" == *"MEMORY.md"* ]]
    [[ "$output" == *"857"* || "$output" == *"490"* ]]
}

# --- PowerShell twin: structural only ---------------------------------------
# pwsh is not available on the Linux dev box, so the twin cannot be executed
# here; PSScriptAnalyzer and the Windows suite cover it in CI. These assertions
# exist so the twin cannot silently drift away from the bash guard, which is the
# divergence #850 explicitly refused to allow ("fixing only bash would leave
# Windows silently breaking the invariant").

@test "yaml-guard: the PowerShell twin carries the same guard" {
    grep -q 'function Test-YamlBlockScalar' "$SCRIPTS_DIR/knowledge-crystallize.ps1"
    grep -q 'Test-YamlBlockScalar -FilePath $MemoryFile' "$SCRIPTS_DIR/knowledge-crystallize.ps1"
}

# NOTE: a "neither twin keys on a literal indent width" grep-assertion was
# written here and deleted. Mutation showed it stayed green after a literal
# `^    ## Session Handoff` anchor was added to the script, so it could not fail
# and was pure decoration. The claim is already carried, behaviourally, by the
# four- and six-space cases above: a guard keyed to either width turns one of
# them red. A test that cannot fail is worse than no test, because it reads as
# coverage.

@test "yaml-guard: a refusal is counted as skipped, not processed, in both twins" {
    grep -q 'skipped=$((skipped + 1))' "$SCRIPTS_DIR/knowledge-crystallize.sh"
    grep -q '$skipped++' "$SCRIPTS_DIR/knowledge-crystallize.ps1"
}

@test "yaml-guard: a plain-markdown MEMORY.md is untouched by the guard" {
    # The guard must not cost the shape that already works: #851's fix stays
    # correct and the handoff invariant still holds.
    _setup_plain_project
    HOME="$FAKE_HOME" run bash "$SCRIPTS_DIR/knowledge-crystallize.sh" "$FAKE_PROJECT"
    [ "$status" -eq 0 ]
    local handoff_line date_line
    handoff_line=$(grep -n '^## Session Handoff' "$FAKE_MEM_DIR/MEMORY.md" | cut -d: -f1)
    date_line=$(grep -n '^# currentDate' "$FAKE_MEM_DIR/MEMORY.md" | cut -d: -f1)
    [ -n "$handoff_line" ]
    [ "$date_line" -lt "$handoff_line" ]
}
