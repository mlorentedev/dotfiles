#!/usr/bin/env bats
# Tests for scripts/skills-to-opencode.sh (AI-012)
# Verifies the deterministic port of ai/skills/*/SKILL.md to
# ai/opencode/commands/*.md.

setup() {
    export REPO_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPT="$REPO_DIR/scripts/skills-to-opencode.sh"
    export SKILLS_DIR="$REPO_DIR/ai/skills"
    export COMMANDS_DIR="$REPO_DIR/ai/opencode/commands"
}

@test "script exists and is executable" {
    [ -x "$SCRIPT" ]
}

@test "script passes bash syntax check" {
    bash -n "$SCRIPT"
}

@test "script passes zsh syntax check" {
    zsh -n "$SCRIPT"
}

@test "script --check exits 0 when in sync" {
    "$SCRIPT" --check
}

@test "commands/ directory exists" {
    [ -d "$COMMANDS_DIR" ]
}

@test "12 portable commands generated (5 Claude-only skipped)" {
    local count
    count=$(find "$COMMANDS_DIR" -maxdepth 1 -name '*.md' | wc -l)
    [ "$count" -eq 12 ]
}

@test "skip-list applied: creating-skills NOT in commands/" {
    [ ! -f "$COMMANDS_DIR/creating-skills.md" ]
}

@test "skip-list applied: dispatching-parallel-agents NOT in commands/" {
    [ ! -f "$COMMANDS_DIR/dispatching-parallel-agents.md" ]
}

@test "skip-list applied: crystallize NOT in commands/" {
    [ ! -f "$COMMANDS_DIR/crystallize.md" ]
}

@test "skip-list applied: insights NOT in commands/" {
    [ ! -f "$COMMANDS_DIR/insights.md" ]
}

@test "skip-list applied: executing-plans NOT in commands/" {
    [ ! -f "$COMMANDS_DIR/executing-plans.md" ]
}

@test "portable command audit.md exists" {
    [ -f "$COMMANDS_DIR/audit.md" ]
}

@test "portable command writing-plans.md exists" {
    [ -f "$COMMANDS_DIR/writing-plans.md" ]
}

@test "portable command vault-doctor.md exists (Hive MCP works in OpenCode)" {
    [ -f "$COMMANDS_DIR/vault-doctor.md" ]
}

@test "generated frontmatter drops 'name:' line (filename is the identifier in OpenCode)" {
    ! grep -q '^name:' "$COMMANDS_DIR/audit.md"
}

@test "generated frontmatter keeps 'description:' line" {
    grep -q '^description:' "$COMMANDS_DIR/audit.md"
}

@test "generated command preserves frontmatter delimiters" {
    local first second
    first=$(grep -n '^---$' "$COMMANDS_DIR/audit.md" | head -1 | cut -d: -f1)
    second=$(grep -n '^---$' "$COMMANDS_DIR/audit.md" | sed -n '2p' | cut -d: -f1)
    [ "$first" = "1" ]
    [ -n "$second" ]
    [ "$second" -gt "$first" ]
}

@test "generated command preserves body content" {
    # audit.md body starts with '# Security Audit'
    grep -q '^# Security Audit' "$COMMANDS_DIR/audit.md"
}

@test "script --check detects drift when a command file is mutated" {
    local victim="$COMMANDS_DIR/audit.md"
    local backup
    backup=$(mktemp)
    cp "$victim" "$backup"
    echo "# rogue line" >> "$victim"
    run "$SCRIPT" --check
    cp "$backup" "$victim"
    rm -f "$backup"
    [ "$status" -ne 0 ]
}

@test "script --check detects orphan command (no matching skill)" {
    local orphan="$COMMANDS_DIR/nonexistent-skill.md"
    echo "---" > "$orphan"
    echo "description: stray" >> "$orphan"
    echo "---" >> "$orphan"
    run "$SCRIPT" --check
    rm -f "$orphan"
    [ "$status" -ne 0 ]
}

@test "script regeneration is idempotent (no diff on second run)" {
    "$SCRIPT" >/dev/null
    "$SCRIPT" --check
}

@test "script -h / --help prints usage" {
    run "$SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage:"* ]] || [[ "$output" == *"usage:"* ]]
}

@test "script rejects unknown arguments" {
    run "$SCRIPT" --bogus-flag
    [ "$status" -ne 0 ]
}
