#!/usr/bin/env bats
# Tests for scripts/check-md-escapes.sh — Hive vault_patch corruption detector.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    SCRATCH="$BATS_TMPDIR/sdd-006-$$"
    mkdir -p "$SCRATCH"
}

teardown() {
    rm -rf "$SCRATCH"
}

@test "check-md-escapes.sh exists and is executable" {
    [ -x "$SCRIPTS_DIR/check-md-escapes.sh" ]
}

@test "exits 2 with usage when no args given" {
    run "$SCRIPTS_DIR/check-md-escapes.sh"
    [ "$status" -eq 2 ]
    [[ "$output" == *"Usage"* ]]
}

@test "exits 2 when path does not exist" {
    run "$SCRIPTS_DIR/check-md-escapes.sh" "$SCRATCH/nonexistent.md"
    [ "$status" -eq 2 ]
    [[ "$output" == *"not found"* ]]
}

@test "detects '\\\\n-' corruption (backslash-n-hyphen, bullet merge)" {
    fixture="$SCRATCH/corrupt.md"
    # printf writes a literal `\n` sequence (two chars), NOT a newline.
    printf 'foo\\n- merged bullet\n' > "$fixture"
    run "$SCRIPTS_DIR/check-md-escapes.sh" "$fixture"
    [ "$status" -eq 1 ]
    [[ "$output" == *"CORRUPTED"* ]]
    [[ "$output" == *"corrupt.md"* ]]
}

@test "detects '\\\\n#' corruption (header merge)" {
    fixture="$SCRATCH/corrupt-header.md"
    printf 'paragraph end.\\n## Header\n' > "$fixture"
    run "$SCRIPTS_DIR/check-md-escapes.sh" "$fixture"
    [ "$status" -eq 1 ]
}

@test "stays silent on healthy markdown" {
    fixture="$SCRATCH/healthy.md"
    printf '# Title\n\n- item 1\n- item 2\n\n## Section\n\nNo escapes here.\n' > "$fixture"
    run "$SCRIPTS_DIR/check-md-escapes.sh" "$fixture"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "scans directory recursively, reports only corrupted" {
    mkdir -p "$SCRATCH/vault/sub"
    printf '## ok\n- bullet\n' > "$SCRATCH/vault/clean.md"
    printf 'bad\\n- merged\n' > "$SCRATCH/vault/sub/corrupt.md"
    run "$SCRIPTS_DIR/check-md-escapes.sh" "$SCRATCH/vault"
    [ "$status" -eq 1 ]
    [[ "$output" == *"corrupt.md"* ]]
    [[ "$output" != *"clean.md"* ]]
}

@test "ignores .obsidian/ and node_modules/ directories" {
    mkdir -p "$SCRATCH/vault/.obsidian" "$SCRATCH/vault/node_modules"
    printf 'corruption\\n- bad\n' > "$SCRATCH/vault/.obsidian/cache.md"
    printf 'corruption\\n- bad\n' > "$SCRATCH/vault/node_modules/dep.md"
    printf '# ok\n' > "$SCRATCH/vault/clean.md"
    run "$SCRIPTS_DIR/check-md-escapes.sh" "$SCRATCH/vault"
    [ "$status" -eq 0 ]
}

# Self-test: the dotfiles repo's own markdown files MUST stay clean.
# If a future Hive vault_patch (or any tool) corrupts a markdown file in this
# repo, this assertion fails CI loud — catches the bug class at PR time.
@test "self-test: dotfiles repo markdown is clean" {
    run "$SCRIPTS_DIR/check-md-escapes.sh" "$DOTFILES_DIR"
    [ "$status" -eq 0 ]
}
