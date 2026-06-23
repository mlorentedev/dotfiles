#!/usr/bin/env bats
# Regression guard for SessionStart hook false positives on vault repos.
#
# BUG-022: check_knowledge_health greps '^## Last Crystallized:' but the marker
#          is indented when MEMORY.md is embedded in a YAML `content: |` block
#          (the knowledge vault stores it that way) -> the date was never parsed
#          -> hook falsely reported "crystallization never run".
# BUG-023: check_vault_baseline listed AGENTS.md in critical_files, but per
#          ADR-010 AGENTS.md is single-SSOT in dotfiles (deployed, not duplicated
#          to the vault) -> hook falsely reported "Vault baseline FAIL".
#
# Incident note: these bugs shipped LIVE — PR #165 merged an empty commit, so
# this guard exists to keep the real fix (and its rationale) from regressing.
#
# Hermetic design: the hook runs everything top-to-bottom on exec with no main()
# guard, and its sibling vault-health.sh launches Obsidian (which hangs headless,
# see session-start-config.bats). We copy the hook AND its required session-brief
# core (ADR-023 — vault detection / spec / baseline signals live there now) into
# an isolated temp dir, but deliberately NOT the other siblings (vault-health.sh)
# nor the config file, so those lookups MISS and get skipped while the real
# check_knowledge_health / vault-baseline logic runs against a controlled HOME
# and CWD.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    SRC_SCRIPT="$DOTFILES_DIR/scripts/claude-session-start.sh"

    TEST_HOME=$(mktemp -d)
    # Place the copy one level below TEST_HOME so SCRIPT_DIR/../session-start-config.json
    # resolves to an absent path -> CFG_AVAILABLE=0 -> hardcoded defaults
    # (crystallize_max_days=14, memory_md_max_lines=150).
    mkdir -p "$TEST_HOME/bin"
    TMP_SCRIPT="$TEST_HOME/bin/claude-session-start.sh"
    cp "$SRC_SCRIPT" "$TMP_SCRIPT"
    # session-brief.sh is a REQUIRED dependency: the hook sources it for the
    # vault-detection / spec / baseline signals. Copy it alongside so those run;
    # vault-health.sh stays absent so sb_vault_health takes its "not found" path.
    cp "$DOTFILES_DIR/scripts/session-brief.sh" "$TEST_HOME/bin/session-brief.sh"
}

teardown() {
    [ -z "${TEST_HOME:-}" ] || rm -rf "$TEST_HOME"
}

# Run the hook hermetically: isolated HOME, no sibling scripts, given CWD.
run_hook() {
    local cwd="$1"
    printf '{"cwd":"%s","session_id":"bats"}' "$cwd" | HOME="$TEST_HOME" bash "$TMP_SCRIPT" 2>&1
}

# Write a MEMORY.md whose '## Last Crystallized:' marker is indented inside a
# YAML `content: |` block, mirroring how the knowledge vault stores it.
seed_memory() {
    local cwd="$1" date="$2" enc memdir
    enc=$(printf '%s' "$cwd" | tr '/' '-')
    memdir="$TEST_HOME/.claude/projects/$enc/memory"
    mkdir -p "$memdir"
    {
        printf '%s\n' '---'
        printf '%s\n' 'content: |'
        printf '      # Project Memory\n'
        printf '      ## Last Crystallized: %s\n' "$date"
    } > "$memdir/MEMORY.md"
}

# Build a minimal fake vault (has .obsidian so find_vault_root resolves it)
# with the canonical files, omitting AGENTS.md by default.
seed_vault() {
    local root="$1"
    mkdir -p "$root/.obsidian" "$root/00_meta/patterns" "$root/00_meta/skills"
    printf 'x\n' > "$root/00_meta/patterns/_index.md"
    printf 'x\n' > "$root/00_meta/skills/README.md"
    printf 'x\n' > "$root/README.md"
}

# --- BUG-022: indented Last Crystallized marker ---

@test "BUG-022: indented marker with a recent date is parsed (not flagged stale or missing)" {
    local cwd="$TEST_HOME/work/proj"
    mkdir -p "$cwd"
    seed_memory "$cwd" "$(date +%Y-%m-%d)"

    run run_hook "$cwd"
    [ "$status" -eq 0 ]
    [[ "$output" != *"Knowledge crystallization never run"* ]]
    [[ "$output" != *"CRYSTALLIZE NEEDED"* ]]
}

@test "BUG-022: indented marker with an old date IS parsed (-> CRYSTALLIZE NEEDED, not 'never run')" {
    # The old un-anchored grep would miss the indented line and emit "never run".
    # The fixed grep parses the (old) date and emits the stale warning instead.
    local cwd="$TEST_HOME/work/proj"
    mkdir -p "$cwd"
    seed_memory "$cwd" "2000-01-01"

    run run_hook "$cwd"
    [ "$status" -eq 0 ]
    [[ "$output" == *"CRYSTALLIZE NEEDED"* ]]
    [[ "$output" != *"Knowledge crystallization never run"* ]]
}

# --- BUG-023: AGENTS.md not a vault critical file ---

@test "BUG-023: absent AGENTS.md does NOT trigger a vault baseline failure" {
    local vault="$TEST_HOME/vault"
    seed_vault "$vault"   # deliberately no AGENTS.md

    run run_hook "$vault"
    [ "$status" -eq 0 ]
    [[ "$output" != *"MISSING: AGENTS.md"* ]]
    [[ "$output" != *"Vault baseline FAIL"* ]]
}

@test "BUG-023 control: a genuinely missing canonical file STILL fails the baseline" {
    # Proves the fix narrowed the check rather than disabling it.
    local vault="$TEST_HOME/vault2"
    seed_vault "$vault"
    rm -f "$vault/README.md"

    run run_hook "$vault"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Vault baseline FAIL"* ]]
    [[ "$output" == *"MISSING: README.md"* ]]
}
