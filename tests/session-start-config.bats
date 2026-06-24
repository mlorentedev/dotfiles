#!/usr/bin/env bats
# Schema guard for session-start-config.json (SDD-004): the SSOT for the
# session-start thresholds + injector flags. The shell hooks that read it were
# retired in CLI-025 — the Go `dotf mem session-start` adapter reads it now and is
# tested in cli/internal/mem (TestLoadAdapterConfig). This file keeps the config
# file's shape pinned so a missing threshold/injector key is caught at CI time, not
# at session start.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    CONFIG_JSON="$DOTFILES_DIR/session-start-config.json"
}

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

@test "session-start-config.json has the 11 injector entries with enabled flags" {
    for key in sdd_reminder doctor_drift hive_project specs_summary vault_root_detection vault_health auto_memory_symlink knowledge_health vault_baseline memory_temperature claude_json_size; do
        val=$(jq -r ".injectors.$key.enabled // \"MISSING\"" "$CONFIG_JSON")
        [ "$val" != "MISSING" ] || { echo "missing key: injectors.$key.enabled"; return 1; }
    done
}
