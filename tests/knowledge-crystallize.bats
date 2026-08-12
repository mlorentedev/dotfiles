#!/usr/bin/env bats
# Tests for scripts/knowledge-crystallize.sh

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
}

@test "knowledge-crystallize.sh valid bash syntax" {
    bash -n "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh valid zsh syntax" {
    zsh -n "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh is executable" {
    [[ -x "$SCRIPTS_DIR/knowledge-crystallize.sh" ]]
}

@test "knowledge-crystallize.sh uses set -euo pipefail" {
    grep -q 'set -euo pipefail' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh sources utils.sh" {
    grep -q 'utils.sh' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh has --help or usage output" {
    grep -q 'usage\(\)' "$SCRIPTS_DIR/knowledge-crystallize.sh"
    grep -q '\-\-help' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh passes shellcheck" {
    if command -v shellcheck >/dev/null 2>&1; then
        shellcheck "$SCRIPTS_DIR/knowledge-crystallize.sh"
    elif [[ -x "$HOME/.local/bin/shellcheck" ]]; then
        "$HOME/.local/bin/shellcheck" "$SCRIPTS_DIR/knowledge-crystallize.sh"
    else
        skip "shellcheck not available"
    fi
}

@test "knowledge-crystallize.sh has encode_path function" {
    grep -q 'encode_path\(\)' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh has find_memory_file function" {
    grep -q 'find_memory_file\(\)' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh has stamp_last_crystallized function" {
    grep -q 'stamp_last_crystallized\(\)' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh prints checklist" {
    grep -q 'print_checklist' "$SCRIPTS_DIR/knowledge-crystallize.sh"
    grep -q 'crystallize\|crystallization' "$SCRIPTS_DIR/knowledge-crystallize.sh"
}

@test "knowledge-crystallize.sh --help exits 0" {
    run "$SCRIPTS_DIR/knowledge-crystallize.sh" --help
    [[ "$status" -eq 0 ]]
}

@test "knowledge-crystallize.sh --help shows usage" {
    run "$SCRIPTS_DIR/knowledge-crystallize.sh" --help
    [[ "$output" == *"USAGE"* ]]
}

# --- behavioural: HARNESS-029 invariant -------------------------------------
# The ## Session Handoff block must remain the LAST section of MEMORY.md.
# Regression: update_current_date/stamp_last_crystallized used to append to EOF
# when their marker was absent, displacing the block on the first crystallize.

setup_fake_project() {
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
**Next action:** something else
EOF
}

@test "knowledge-crystallize.sh keeps Session Handoff as the last section" {
    setup_fake_project
    HOME="$FAKE_HOME" run bash "$BATS_TEST_DIRNAME/../scripts/knowledge-crystallize.sh" "$FAKE_PROJECT"
    [ "$status" -eq 0 ]

    # Both stamps must land BEFORE the handoff block, never after it.
    local handoff_line date_line stamp_line
    handoff_line=$(grep -n '^## Session Handoff' "$FAKE_MEM_DIR/MEMORY.md" | cut -d: -f1)
    date_line=$(grep -n '^# currentDate' "$FAKE_MEM_DIR/MEMORY.md" | cut -d: -f1)
    stamp_line=$(grep -n '^## Last Crystallized:' "$FAKE_MEM_DIR/MEMORY.md" | cut -d: -f1)

    [ -n "$handoff_line" ]
    [ "$date_line" -lt "$handoff_line" ]
    [ "$stamp_line" -lt "$handoff_line" ]
    rm -rf "$FAKE_HOME"
}

@test "knowledge-crystallize.sh is idempotent on the handoff invariant" {
    setup_fake_project
    HOME="$FAKE_HOME" bash "$BATS_TEST_DIRNAME/../scripts/knowledge-crystallize.sh" "$FAKE_PROJECT"
    HOME="$FAKE_HOME" bash "$BATS_TEST_DIRNAME/../scripts/knowledge-crystallize.sh" "$FAKE_PROJECT"

    # A second run must not duplicate sections nor move the block.
    [ "$(grep -c '^# currentDate' "$FAKE_MEM_DIR/MEMORY.md")" -eq 1 ]
    [ "$(grep -c '^## Last Crystallized:' "$FAKE_MEM_DIR/MEMORY.md")" -eq 1 ]
    local handoff_line date_line
    handoff_line=$(grep -n '^## Session Handoff' "$FAKE_MEM_DIR/MEMORY.md" | cut -d: -f1)
    date_line=$(grep -n '^# currentDate' "$FAKE_MEM_DIR/MEMORY.md" | cut -d: -f1)
    [ "$date_line" -lt "$handoff_line" ]
    rm -rf "$FAKE_HOME"
}

@test "knowledge-crystallize.sh standalone (no utils.sh sibling) refuses cleanly on a BUG-062 block-scalar file, not with a bash command-not-found (BUG-065)" {
    # Runs a COPY of the real script alone in a scratch dir, with no utils.sh
    # beside it, so the "Minimal fallbacks when run standalone" branch is the
    # one that actually executes -- the scenario the fallback exists for.
    #
    # Empirically verified before fixing (not assumed from the issue text):
    # a missing log_error() does NOT crash the whole script under
    # set -euo pipefail here, because process_project() is called as an `if`
    # condition (`if ! process_project ...; then`), and bash suspends -e's
    # abort-on-error for every command inside a function invoked as an if
    # condition. The real, verified defect is milder but still real: the
    # refusal still exits 1, but its message degrades from the intended
    # `[ERROR] Refusing to stamp ...` lines to bash's own
    # "log_error: command not found", which is not actionable and defeats
    # the guard's whole point.
    local scratch standalone_script fake_home fake_project encoded mem_dir
    scratch="$(mktemp -d)"
    mkdir -p "$scratch/scripts"
    cp "$BATS_TEST_DIRNAME/../scripts/knowledge-crystallize.sh" "$scratch/scripts/"
    standalone_script="$scratch/scripts/knowledge-crystallize.sh"

    fake_home="$scratch/home"
    fake_project="$scratch/project"
    mkdir -p "$fake_project"
    encoded="$(printf '%s' "$fake_project" | tr '/' '-')"
    mem_dir="$fake_home/.claude/projects/$encoded/memory"
    mkdir -p "$mem_dir"
    # BUG-062 shape: a YAML block scalar, which is_yaml_block_scalar detects
    # and process_project() refuses to touch.
    printf -- '---\nbody: |\n  some content\n' > "$mem_dir/MEMORY.md"

    HOME="$fake_home" run "$standalone_script" "$fake_project"

    [ "$status" -eq 1 ]
    [[ "$output" == *"[ERROR]"*"Refusing to stamp"* ]]
    [[ "$output" != *"command not found"* ]]
    rm -rf "$scratch"
}
