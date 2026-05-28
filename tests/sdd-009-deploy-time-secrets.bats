#!/usr/bin/env bats
# Tests for SDD-009: deploy-time secret substitution for opencode.jsonc.
#
# Helper under test: substitute_env_placeholders <file> (in scripts/utils.sh).
# Reads {env:NAME} tokens from <file>, looks up NAME in
# sensitive/env-mapping.conf, decrypts sensitive/<value>.secret.age with age,
# and rewrites <file> in place. Unresolved placeholders are left intact with
# a log_warning, allowing opencode's runtime env resolver to act as fallback.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"

    # Isolated fixture sandbox: fake age key + fake encrypted secret + fake mapping.
    export FIX_DIR="$BATS_TEST_TMPDIR/sdd009_$$"
    mkdir -p "$FIX_DIR"

    # Generate a real age keypair (these tests need a real age binary).
    if ! command -v age >/dev/null 2>&1 || ! command -v age-keygen >/dev/null 2>&1; then
        skip "age / age-keygen not installed"
    fi
    export AGE_KEY_PATH="$FIX_DIR/key.txt"
    age-keygen -o "$AGE_KEY_PATH" 2>/dev/null
    PUBKEY=$(grep -o 'age1[0-9a-z]*' "$AGE_KEY_PATH")

    # Fake secret: encrypted value for NAN_API_KEY.
    export SECRETS_DIR="$FIX_DIR/sensitive"
    mkdir -p "$SECRETS_DIR"
    printf 'FIXTURE-NAN-VALUE-ONE' | age -r "$PUBKEY" -o "$SECRETS_DIR/nan.api-key.secret.age"

    # Mapping file: NAN_API_KEY resolves, OLLAMA_API_KEY commented (unresolvable).
    export SECRETS_MAPPING_FILE="$SECRETS_DIR/env-mapping.conf"
    cat > "$SECRETS_MAPPING_FILE" <<EOF
NAN_API_KEY=nan.api-key
# OLLAMA_API_KEY=ollama.api-key
EOF

    # Sample target file mimicking opencode.jsonc layout.
    export TARGET_FILE="$FIX_DIR/opencode.jsonc"
    cat > "$TARGET_FILE" <<'EOF'
{
  "provider": {
    "nan":    { "options": { "apiKey": "{env:NAN_API_KEY}" } },
    "ollama": { "options": { "apiKey": "{env:OLLAMA_API_KEY}" } }
  }
}
EOF
}

teardown() {
    rm -rf "$FIX_DIR"
}

# --- Helper presence ---

@test "substitute_env_placeholders is defined in utils.sh" {
    run bash -c 'source "$1/utils.sh"; type substitute_env_placeholders >/dev/null 2>&1 && echo OK' -- "$SCRIPTS_DIR"
    [[ "$output" == "OK" ]]
}

# --- Core behavior ---

@test "substitutes resolvable {env:NAN_API_KEY} with decrypted value" {
    run bash -c '
        source "$1/utils.sh"
        substitute_env_placeholders "$2"
    ' -- "$SCRIPTS_DIR" "$TARGET_FILE"
    [[ $status -eq 0 ]]
    grep -q '"apiKey": "FIXTURE-NAN-VALUE-ONE"' "$TARGET_FILE"
}

@test "leaves unresolvable {env:OLLAMA_API_KEY} placeholder intact" {
    bash -c '
        source "$1/utils.sh"
        substitute_env_placeholders "$2"
    ' -- "$SCRIPTS_DIR" "$TARGET_FILE"
    # Resolved: NAN substituted.
    grep -q '"apiKey": "FIXTURE-NAN-VALUE-ONE"' "$TARGET_FILE"
    # Unresolved: OLLAMA placeholder preserved verbatim.
    grep -q '"apiKey": "{env:OLLAMA_API_KEY}"' "$TARGET_FILE"
}

@test "emits log_warning for unresolved placeholders" {
    run bash -c '
        source "$1/utils.sh"
        substitute_env_placeholders "$2" 2>&1
    ' -- "$SCRIPTS_DIR" "$TARGET_FILE"
    [[ "$output" == *"[WARNING]"* ]]
    [[ "$output" == *"OLLAMA_API_KEY"* ]]
}

@test "no warning when every placeholder resolves" {
    # Make OLLAMA_API_KEY resolvable: add mapping + secret.
    PUBKEY=$(grep -o 'age1[0-9a-z]*' "$AGE_KEY_PATH")
    printf 'FIXTURE-OLLAMA-VALUE' | age -r "$PUBKEY" -o "$SECRETS_DIR/ollama.api-key.secret.age"
    cat > "$SECRETS_MAPPING_FILE" <<EOF
NAN_API_KEY=nan.api-key
OLLAMA_API_KEY=ollama.api-key
EOF
    run bash -c '
        source "$1/utils.sh"
        substitute_env_placeholders "$2" 2>&1
    ' -- "$SCRIPTS_DIR" "$TARGET_FILE"
    [[ $status -eq 0 ]]
    [[ "$output" != *"[WARNING]"* ]]
    # AC5 strict: zero {env: tokens remain when everything resolves.
    ! grep -q '{env:' "$TARGET_FILE"
}

@test "idempotent: re-running with rotated secret re-substitutes" {
    # First run substitutes original value.
    bash -c '
        source "$1/utils.sh"
        substitute_env_placeholders "$2"
    ' -- "$SCRIPTS_DIR" "$TARGET_FILE"
    grep -q '"apiKey": "FIXTURE-NAN-VALUE-ONE"' "$TARGET_FILE"

    # Rotate the secret and re-deploy the source: target must update.
    PUBKEY=$(grep -o 'age1[0-9a-z]*' "$AGE_KEY_PATH")
    printf 'FIXTURE-NAN-VALUE-ROTATED' | age -r "$PUBKEY" -o "$SECRETS_DIR/nan.api-key.secret.age"
    # Restore source to original placeholder state (simulating re-deploy from
    # the canonical ai/opencode/opencode.jsonc before re-running the helper).
    cat > "$TARGET_FILE" <<'EOF'
{
  "provider": {
    "nan":    { "options": { "apiKey": "{env:NAN_API_KEY}" } },
    "ollama": { "options": { "apiKey": "{env:OLLAMA_API_KEY}" } }
  }
}
EOF
    bash -c '
        source "$1/utils.sh"
        substitute_env_placeholders "$2"
    ' -- "$SCRIPTS_DIR" "$TARGET_FILE"
    grep -q '"apiKey": "FIXTURE-NAN-VALUE-ROTATED"' "$TARGET_FILE"
    ! grep -q 'FIXTURE-NAN-VALUE-ONE' "$TARGET_FILE"
}

@test "deployed file has owner-only permissions (mode 600)" {
    bash -c '
        source "$1/utils.sh"
        substitute_env_placeholders "$2"
    ' -- "$SCRIPTS_DIR" "$TARGET_FILE"
    # stat -c is GNU; fall back to BSD -f when needed.
    if mode=$(stat -c '%a' "$TARGET_FILE" 2>/dev/null); then :; else mode=$(stat -f '%A' "$TARGET_FILE"); fi
    [[ "$mode" == "600" ]]
}

# --- Edge cases ---

@test "no-op when file contains zero {env:} tokens" {
    cat > "$TARGET_FILE" <<'EOF'
{ "model": "gpt-4", "temperature": 0.7 }
EOF
    original_md5=$(md5sum "$TARGET_FILE" | awk '{print $1}')
    run bash -c '
        source "$1/utils.sh"
        substitute_env_placeholders "$2"
    ' -- "$SCRIPTS_DIR" "$TARGET_FILE"
    [[ $status -eq 0 ]]
    new_md5=$(md5sum "$TARGET_FILE" | awk '{print $1}')
    [[ "$original_md5" == "$new_md5" ]]
}

@test "fails gracefully when target file does not exist" {
    run bash -c '
        source "$1/utils.sh"
        substitute_env_placeholders "/nonexistent/path/file.jsonc"
    ' -- "$SCRIPTS_DIR"
    [[ $status -ne 0 ]]
}

# --- Regression: must not abort under `set -euo pipefail` (caller setup-linux.sh) ---

@test "survives set -euo pipefail with unresolved placeholders (CI integration regression)" {
    # setup-linux.sh sources utils.sh under `set -euo pipefail`, so the helper
    # must tolerate grep "no match" returns (which arise legitimately when a
    # {env:VAR} has no mapping line, e.g. OLLAMA_API_KEY commented today).
    # A prior version of the helper exited 1 here, killing the entire setup.
    run bash -c '
        set -euo pipefail
        source "$1/utils.sh"
        export SECRETS_DIR="$2"
        export SECRETS_MAPPING_FILE="$2/env-mapping.conf"
        export AGE_KEY_PATH="$3"
        substitute_env_placeholders "$4"
        echo SURVIVED
    ' -- "$SCRIPTS_DIR" "$SECRETS_DIR" "$AGE_KEY_PATH" "$TARGET_FILE"
    [[ $status -eq 0 ]]
    [[ "$output" == *"SURVIVED"* ]]
}

@test "survives set -euo pipefail with file containing zero placeholders" {
    cat > "$TARGET_FILE" <<'EOF'
{ "model": "gpt-4" }
EOF
    run bash -c '
        set -euo pipefail
        source "$1/utils.sh"
        substitute_env_placeholders "$2"
        echo SURVIVED
    ' -- "$SCRIPTS_DIR" "$TARGET_FILE"
    [[ $status -eq 0 ]]
    [[ "$output" == *"SURVIVED"* ]]
}

# --- Cross-OS contract: utils.ps1 must export the parity function ---

@test "utils.ps1 declares Substitute-EnvPlaceholders function" {
    grep -q 'function Substitute-EnvPlaceholders' "$SCRIPTS_DIR/utils.ps1"
}
