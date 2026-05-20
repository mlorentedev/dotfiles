#!/usr/bin/env bats
# Tests for cross-file docs sync (SDD-005).
# Catches the drift class that hit BUG-001, BUG-002, and AI-019 → AUDIT-003.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    AI_COPILOT="$DOTFILES_DIR/ai/copilot/copilot-instructions.md"
    GH_COPILOT="$DOTFILES_DIR/.github/copilot-instructions.md"
    SCRATCH="$BATS_TMPDIR/sdd-005"
    mkdir -p "$SCRATCH"
}

teardown() {
    rm -rf "$SCRATCH"
}

@test "ai/copilot/copilot-instructions.md exists" {
    [ -f "$AI_COPILOT" ]
}

@test ".github/copilot-instructions.md exists" {
    [ -f "$GH_COPILOT" ]
}

# Parity rule (SDD-005): the two copilot-instructions files MUST contain the
# same content modulo the pointer banner. The pointer banner is a leading
# blockquote (lines beginning with `> `) that legitimately differs because
# .github/ uses a markdown link to ../AGENTS.md while ai/copilot/ uses a plain
# backticked reference (different deploy contexts). Everything outside the
# blockquote MUST be byte-identical.
@test "copilot-instructions: ai/copilot/ and .github/ match outside pointer banner" {
    sed -E '/^> /d' "$AI_COPILOT" > "$SCRATCH/ai.txt"
    sed -E '/^> /d' "$GH_COPILOT" > "$SCRATCH/gh.txt"
    if ! cmp -s "$SCRATCH/ai.txt" "$SCRATCH/gh.txt"; then
        echo "DRIFT: ai/copilot/ and .github/copilot-instructions.md differ outside the pointer banner."
        echo "Run 'diff' on the two files; sync ai/copilot/ -> .github/ (excluding the > blockquote)."
        diff "$SCRATCH/ai.txt" "$SCRATCH/gh.txt" >&2 || true
        return 1
    fi
}

@test "copilot-instructions: both files contain '## Model Tier' section header" {
    grep -q '^## Model Tier' "$AI_COPILOT"
    grep -q '^## Model Tier' "$GH_COPILOT"
}

@test "copilot-instructions: both reference AGENTS.md in their pointer banner" {
    grep -qE '^> .*AGENTS\.md' "$AI_COPILOT"
    grep -qE '^> .*AGENTS\.md' "$GH_COPILOT"
}
