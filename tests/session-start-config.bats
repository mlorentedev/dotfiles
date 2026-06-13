#!/usr/bin/env bats
# Tests for SDD-004 session-start-config SSOT.
# Verifies: (1) session-start-config.json is valid JSON with expected shape,
# (2) claude-session-start.sh reads thresholds from it (no hardcoded magic
# numbers), (3) claude-session-start.ps1 hardcoded thresholds still MATCH the
# JSON values — drift detector until SDD-004b mirrors the refactor to Windows.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export CONFIG_JSON="$DOTFILES_DIR/session-start-config.json"
    export SH_SCRIPT="$DOTFILES_DIR/scripts/claude-session-start.sh"
    export PS1_SCRIPT="$DOTFILES_DIR/scripts/claude-session-start.ps1"

    # Headless-safe (#167): the hook -> vault-health.sh probes the vault via the
    # `obsidian` CLI (PATH). On a machine where Obsidian is installed this LAUNCHES
    # the real GUI, which never returns headless (40+ min hang). Shadow `obsidian`
    # with a stub that reports GUI-down so vault-health exits 2 (handled) instead
    # of launching. Deterministic + hermetic for every test that runs the hook.
    STUB_BIN="$(mktemp -d)"
    printf '#!/usr/bin/env bash\nexit 1\n' > "$STUB_BIN/obsidian"
    chmod +x "$STUB_BIN/obsidian"
    export PATH="$STUB_BIN:$PATH"
}

teardown() {
    [ -z "${STUB_BIN:-}" ] || rm -rf "$STUB_BIN"
}

# --- Schema: session-start-config.json shape ---

@test "session-start-config.json exists at repo root" {
    [ -f "$CONFIG_JSON" ]
}

@test "session-start-config.json is valid JSON" {
    jq empty "$CONFIG_JSON"
}

@test "session-start-config.json has the 6 required thresholds" {
    for key in claude_json_min_bytes memory_md_max_lines crystallize_max_days memory_temp_hot_days memory_temp_warm_days memory_temp_cold_days; do
        val=$(jq -r ".thresholds.$key // \"MISSING\"" "$CONFIG_JSON")
        [ "$val" != "MISSING" ] || { echo "missing key: thresholds.$key"; return 1; }
    done
}

@test "session-start-config.json has the 12 injector entries with enabled flags" {
    for key in sdd_reminder claude_mem_heal doctor_drift hive_project specs_summary vault_root_detection vault_health auto_memory_symlink knowledge_health vault_baseline memory_temperature claude_json_size; do
        val=$(jq -r ".injectors.$key.enabled // \"MISSING\"" "$CONFIG_JSON")
        [ "$val" != "MISSING" ] || { echo "missing key: injectors.$key.enabled"; return 1; }
    done
}

# --- .sh refactor: thresholds via cfg_threshold helper ---

@test "claude-session-start.sh defines cfg_threshold helper" {
    grep -q '^cfg_threshold()' "$SH_SCRIPT"
}

@test "claude-session-start.sh defines cfg_injector_enabled helper" {
    grep -q '^cfg_injector_enabled()' "$SH_SCRIPT"
}

@test "claude-session-start.sh reads memory_md_max_lines via cfg_threshold" {
    grep -q 'cfg_threshold memory_md_max_lines' "$SH_SCRIPT"
}

@test "claude-session-start.sh reads crystallize_max_days via cfg_threshold" {
    grep -q 'cfg_threshold crystallize_max_days' "$SH_SCRIPT"
}

@test "claude-session-start.sh reads memory_temp_*_days via cfg_threshold" {
    grep -q 'cfg_threshold memory_temp_hot_days' "$SH_SCRIPT"
    grep -q 'cfg_threshold memory_temp_warm_days' "$SH_SCRIPT"
    grep -q 'cfg_threshold memory_temp_cold_days' "$SH_SCRIPT"
}

@test "claude-session-start.sh reads claude_json_min_bytes via cfg_threshold" {
    grep -q 'cfg_threshold claude_json_min_bytes' "$SH_SCRIPT"
}

# --- Cross-OS parity drift detector ---
# .ps1 still has hardcoded thresholds (SDD-004b mirror PR pending). Until that
# lands, this PR locks the values: any change to the JSON without a matching
# .ps1 edit (or vice-versa) fails CI.

@test "ps1: claude_json_min_bytes matches JSON" {
    json_val=$(jq -r '.thresholds.claude_json_min_bytes' "$CONFIG_JSON")
    grep -qE "\\\$threshold\s*=\s*$json_val\b" "$PS1_SCRIPT"
}

@test "ps1: memory_md_max_lines matches JSON" {
    json_val=$(jq -r '.thresholds.memory_md_max_lines' "$CONFIG_JSON")
    grep -qE "lineCount -gt $json_val\b" "$PS1_SCRIPT"
}

@test "ps1: crystallize_max_days matches JSON" {
    json_val=$(jq -r '.thresholds.crystallize_max_days' "$CONFIG_JSON")
    grep -qE "daysSince -gt $json_val\b" "$PS1_SCRIPT"
}

# --- Code-controlled byte-equivalence ---
# Compares HEAD/main copy of the script vs current refactored copy across 3
# representative CWDs (dotfiles repo / outside-vault tmp / inside-vault).
# IMPORTANT: PRE script must run from same dir as POST so SCRIPT_DIR resolves
# to the same scripts/ folder for both (otherwise sibling-script lookups —
# claude-mem-heal.sh, vault-health.sh — diverge artificially).

@test "byte-equivalence: refactor preserves output (3 CWD scenarios)" {
    if ! command -v jq >/dev/null 2>&1; then skip "jq required"; fi
    # Compare against origin/main when available (truer baseline than a possibly
    # stale local main — avoids false DIFFs when local main lags; #167 defect 2).
    local base_ref=""
    if git -C "$DOTFILES_DIR" rev-parse --verify origin/main >/dev/null 2>&1; then
        base_ref="origin/main"
    elif git -C "$DOTFILES_DIR" rev-parse --verify main >/dev/null 2>&1; then
        base_ref="main"
    else
        skip "no main / origin/main ref available"
    fi

    local pre="$DOTFILES_DIR/scripts/claude-session-start.sh.pre-refactor"
    git -C "$DOTFILES_DIR" show "$base_ref:scripts/claude-session-start.sh" > "$pre"
    chmod +x "$pre"

    local failures=0
    for cwd in "$DOTFILES_DIR" "/tmp" "$HOME/Projects/knowledge"; do
        [ -d "$cwd" ] || continue
        local input="{\"cwd\":\"$cwd\",\"session_id\":\"bats\"}"
        local pre_out post_out
        pre_out=$(echo "$input" | bash "$pre" 2>&1) || true
        post_out=$(echo "$input" | bash "$SH_SCRIPT" 2>&1) || true
        if [ "$pre_out" != "$post_out" ]; then
            echo "DIFF for cwd=$cwd"
            diff <(echo "$pre_out") <(echo "$post_out") | head -10
            failures=$((failures + 1))
        fi
    done

    rm -f "$pre"
    [ "$failures" -eq 0 ]
}
