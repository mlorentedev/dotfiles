#!/usr/bin/env bats
# Secret-resolution contract for the nan-* helper scripts (ADR-028 Phase 1,
# spec CLI-024-secrets-retire-loadsecrets / #587).
#
# The nan-* scripts must resolve NAN_API_KEY through the `dotf secrets` facade
# (`dotf secrets show nan-api-key`, or injected by `dotf secrets run`), never by
# sourcing the retired `load-secrets` twin. Structural (grep-based) like the
# other RC/script contract tests in this suite.

setup() {
    export SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    NAN_SCRIPTS=(nan-bench.sh nan-debug.sh nan-quality-bench.sh)
}

@test "nan-* scripts exist" {
    for s in "${NAN_SCRIPTS[@]}"; do
        [[ -f "$SCRIPTS_DIR/$s" ]] || { echo "missing: $s"; return 1; }
    done
}

@test "nan-* scripts do not source the retired load-secrets twin" {
    for s in "${NAN_SCRIPTS[@]}"; do
        ! grep -qE 'load-secrets' "$SCRIPTS_DIR/$s" || { echo "still references load-secrets: $s"; return 1; }
    done
}

@test "nan-* scripts do not call secrets_refresh (the old twin API)" {
    for s in "${NAN_SCRIPTS[@]}"; do
        ! grep -q 'secrets_refresh' "$SCRIPTS_DIR/$s" || { echo "still calls secrets_refresh: $s"; return 1; }
    done
}

@test "nan-* scripts self-fetch NAN_API_KEY via dotf secrets show" {
    for s in "${NAN_SCRIPTS[@]}"; do
        grep -qF 'dotf secrets show nan-api-key' "$SCRIPTS_DIR/$s" || { echo "no 'dotf secrets show': $s"; return 1; }
    done
}

@test "nan-* scripts guard the dotf fetch with command -v dotf" {
    for s in "${NAN_SCRIPTS[@]}"; do
        grep -qE 'command -v dotf' "$SCRIPTS_DIR/$s" || { echo "no 'command -v dotf' guard: $s"; return 1; }
    done
}

@test "nan-* scripts hint 'dotf secrets run' when the key is missing" {
    for s in "${NAN_SCRIPTS[@]}"; do
        grep -qF 'dotf secrets run' "$SCRIPTS_DIR/$s" || { echo "no run hint: $s"; return 1; }
    done
}

@test "nan-* scripts parse cleanly (bash -n)" {
    for s in "${NAN_SCRIPTS[@]}"; do
        bash -n "$SCRIPTS_DIR/$s" || { echo "syntax error: $s"; return 1; }
    done
}
