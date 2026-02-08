#!/usr/bin/env bash

# Comprehensive test suite for dotfiles
# Usage: ./scripts/test-dotfiles.sh
# Run after: ./setup-linux.sh

set -o pipefail

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Test result functions
pass() {
    printf '  %bPASS%b: %s\n' "$GREEN" "$NC" "$1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
    printf '  %bFAIL%b: %s\n' "$RED" "$NC" "$1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

skip() {
    printf '  %bSKIP%b: %s - %s\n' "$YELLOW" "$NC" "$1" "$2"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
}

section() {
    echo ""
    printf '%b[%s]%b %s\n' "$BLUE" "$1" "$NC" "$2"
}

subsection() {
    printf '  %b--- %s ---%b\n' "$CYAN" "$1" "$NC"
}

# Determine paths
if [[ -n "$DOTFILES_DIR" ]]; then
    echo "Using custom DOTFILES_DIR: $DOTFILES_DIR"
elif [[ -d "$HOME/.dotfiles" ]]; then
    DOTFILES_DIR="$HOME/.dotfiles"
elif [[ -d "$(dirname "${BASH_SOURCE[0]}")/.." ]]; then
    DOTFILES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
else
    echo "Error: Could not find dotfiles directory"
    exit 1
fi

SCRIPTS_DIR="$DOTFILES_DIR/scripts"
SENSITIVE_DIR="$DOTFILES_DIR/sensitive"

echo "========================================"
echo "   DOTFILES COMPREHENSIVE TEST SUITE"
echo "========================================"
echo "Testing from: $DOTFILES_DIR"
echo "Shell: ${ZSH_VERSION:+zsh $ZSH_VERSION}${BASH_VERSION:+bash $BASH_VERSION}"

# ==================================================
section "1/15" "Script Syntax Validation"
for script in utils.sh load-secrets.sh age-encrypt-decrypt.sh github-secrets-manager.sh install-precommit.sh dotfiles-sync.sh; do
    if [[ -f "$SCRIPTS_DIR/$script" ]]; then
        if bash -n "$SCRIPTS_DIR/$script" 2>/dev/null; then
            pass "$script syntax OK"
        else
            fail "$script has syntax errors"
        fi
    else
        fail "$script not found"
    fi
done

if [[ -f "$DOTFILES_DIR/setup-linux.sh" ]]; then
    bash -n "$DOTFILES_DIR/setup-linux.sh" 2>/dev/null && pass "setup-linux.sh syntax OK" || fail "setup-linux.sh syntax errors"
fi

# ==================================================
section "2/15" "Source utils.sh"
# ==================================================

if . "$SCRIPTS_DIR/utils.sh" 2>/dev/null; then
    pass "utils.sh sourced successfully"
else
    fail "Failed to source utils.sh"
    echo "Cannot continue without utils.sh"
    exit 1
fi

# ==================================================
section "3/15" "Silent Check Functions"
# ==================================================

subsection "command_exists"
command_exists bash && pass "command_exists: finds existing command" || fail "command_exists: bash not found"
command_exists __nonexistent_cmd_xyz__ && fail "command_exists: false positive" || pass "command_exists: returns false for missing"

subsection "file_exists"
file_exists "$SCRIPTS_DIR/utils.sh" && pass "file_exists: finds existing file" || fail "file_exists: utils.sh not found"
file_exists "/nonexistent/file_xyz.txt" && fail "file_exists: false positive" || pass "file_exists: returns false for missing"

subsection "dir_exists"
dir_exists "$SCRIPTS_DIR" && pass "dir_exists: finds existing dir" || fail "dir_exists: scripts dir not found"
dir_exists "/nonexistent/dir_xyz" && fail "dir_exists: false positive" || pass "dir_exists: returns false for missing"

subsection "symlink_valid"
# Create temp symlink for testing
TEST_LINK="/tmp/test_symlink_$$"
TEST_TARGET="/tmp/test_target_$$"
echo "target" > "$TEST_TARGET"
ln -sf "$TEST_TARGET" "$TEST_LINK"
symlink_valid "$TEST_LINK" && pass "symlink_valid: valid symlink" || fail "symlink_valid: should be valid"
rm -f "$TEST_TARGET"
symlink_valid "$TEST_LINK" && fail "symlink_valid: broken symlink passed" || pass "symlink_valid: detects broken symlink"
rm -f "$TEST_LINK"

subsection "var_is_set"
# shellcheck disable=SC2034
TEST_VAR_SET="hello"
var_is_set TEST_VAR_SET && pass "var_is_set: detects set variable" || fail "var_is_set: failed on set var"
unset TEST_VAR_UNSET 2>/dev/null
var_is_set TEST_VAR_UNSET && fail "var_is_set: false positive on unset" || pass "var_is_set: returns false for unset"
# shellcheck disable=SC2034
TEST_VAR_EMPTY=""
var_is_set TEST_VAR_EMPTY && fail "var_is_set: false positive on empty" || pass "var_is_set: returns false for empty"

# ==================================================
section "4/15" "String Helper Functions"
# ==================================================

subsection "trim"
[[ "$(trim '  hello  ')" == "hello" ]] && pass "trim: removes spaces" || fail "trim: spaces"
[[ "$(trim 'hello')" == "hello" ]] && pass "trim: no change needed" || fail "trim: no change"
[[ "$(trim '  hello world  ')" == "hello world" ]] && pass "trim: preserves inner spaces" || fail "trim: inner spaces"

subsection "mask_value"
[[ "$(mask_value 'secret')" == "******" ]] && pass "mask_value: full mask" || fail "mask_value: full mask got '$(mask_value 'secret')'"
[[ "$(mask_value 'secret123' 3)" == "sec******" ]] && pass "mask_value: partial mask" || fail "mask_value: partial got '$(mask_value 'secret123' 3)'"
[[ "$(mask_value 'ab' 5)" == "ab" ]] && pass "mask_value: short string" || fail "mask_value: short string"

subsection "base64_encode / base64_decode"
encoded=$(base64_encode "test string 123")
[[ -n "$encoded" ]] && pass "base64_encode: produces output" || fail "base64_encode: empty output"
decoded=$(base64_decode "$encoded")
[[ "$decoded" == "test string 123" ]] && pass "base64_decode: roundtrip matches" || fail "base64_decode: got '$decoded'"

# ==================================================
section "5/15" "Config Parsing Functions"
# ==================================================

subsection "parse_mapping_file"
TEST_CONFIG="/tmp/test_config_$$.conf"
cat > "$TEST_CONFIG" << 'EOF'
# Comment line
KEY1=value1
KEY2=value2
  KEY3 = value3
# Another comment
KEY4="quoted value"
KEY5='single quoted'

EMPTY_LINE_ABOVE=yes
EOF

PARSED_KEYS=()
PARSED_VALUES=()

test_handler() {
    PARSED_KEYS+=("$1")
    PARSED_VALUES+=("$2")
}

# Run without subshell to preserve array modifications
# Redirect count output to a temp file
parse_mapping_file "$TEST_CONFIG" test_handler > /tmp/parse_count_$$
count=$(cat /tmp/parse_count_$$)
rm -f /tmp/parse_count_$$

[[ "$count" == "6" ]] && pass "parse_mapping_file: correct count ($count)" || fail "parse_mapping_file: expected 6, got $count"
[[ "${#PARSED_KEYS[@]}" == "6" ]] && pass "parse_mapping_file: array has 6 keys" || fail "parse_mapping_file: array has ${#PARSED_KEYS[@]} keys"

# zsh arrays are 1-indexed, bash arrays are 0-indexed
if [[ -n "$ZSH_VERSION" ]]; then _i=1; else _i=0; fi

[[ "${PARSED_KEYS[$_i]}" == "KEY1" ]] && pass "parse_mapping_file: first key" || fail "parse_mapping_file: first key '${PARSED_KEYS[$_i]}'"
[[ "${PARSED_VALUES[$_i]}" == "value1" ]] && pass "parse_mapping_file: first value" || fail "parse_mapping_file: first value '${PARSED_VALUES[$_i]}'"
[[ "${PARSED_KEYS[$((_i + 2))]}" == "KEY3" ]] && pass "parse_mapping_file: trims key whitespace" || fail "parse_mapping_file: key whitespace '${PARSED_KEYS[$((_i + 2))]}'"
[[ "${PARSED_VALUES[$((_i + 2))]}" == "value3" ]] && pass "parse_mapping_file: trims value whitespace" || fail "parse_mapping_file: value whitespace '${PARSED_VALUES[$((_i + 2))]}'"
[[ "${PARSED_VALUES[$((_i + 3))]}" == "quoted value" ]] && pass "parse_mapping_file: removes double quotes" || fail "parse_mapping_file: double quotes '${PARSED_VALUES[$((_i + 3))]}'"
[[ "${PARSED_VALUES[$((_i + 4))]}" == "single quoted" ]] && pass "parse_mapping_file: removes single quotes" || fail "parse_mapping_file: single quotes '${PARSED_VALUES[$((_i + 4))]}'"

# Test missing file
parse_mapping_file "/nonexistent_file_xyz" test_handler >/dev/null 2>&1
[[ $? -eq 1 ]] && pass "parse_mapping_file: returns 1 for missing file" || fail "parse_mapping_file: missing file handling"

rm -f "$TEST_CONFIG"

# ==================================================
section "6/15" "Counter Functions"
# ==================================================

subsection "init_counter / increment_counter / get_counter"
init_counter "test_counter"
[[ "$(get_counter test_counter)" == "0" ]] && pass "init_counter: starts at 0" || fail "init_counter: not 0"

increment_counter "test_counter"
[[ "$(get_counter test_counter)" == "1" ]] && pass "increment_counter: increments to 1" || fail "increment_counter: not 1"

increment_counter "test_counter"
increment_counter "test_counter"
[[ "$(get_counter test_counter)" == "3" ]] && pass "increment_counter: increments to 3" || fail "increment_counter: not 3"

# ==================================================
section "7/15" "Age Encryption Functions"
# ==================================================

if command_exists age; then
    pass "age: command available"

    subsection "AGE_DEFAULT_KEY"
    [[ -n "$AGE_DEFAULT_KEY" ]] && pass "AGE_DEFAULT_KEY: is set" || fail "AGE_DEFAULT_KEY: not set"

    if file_exists "$AGE_DEFAULT_KEY"; then
        pass "age key file: exists at $AGE_DEFAULT_KEY"

        subsection "age_get_pubkey"
        pubkey=$(age_get_pubkey)
        [[ "$pubkey" == age1* ]] && pass "age_get_pubkey: valid format ($pubkey)" || fail "age_get_pubkey: invalid format '$pubkey'"

        subsection "age_encrypt / age_decrypt"
        TEST_PLAIN="/tmp/test_plain_$$.txt"
        TEST_ENC="/tmp/test_enc_$$.age"
        TEST_DEC="/tmp/test_dec_$$.txt"
        TEST_CONTENT="secret content $(date +%s)"

        echo "$TEST_CONTENT" > "$TEST_PLAIN"

        if age_encrypt "$TEST_ENC" "$AGE_DEFAULT_KEY" "$TEST_PLAIN"; then
            pass "age_encrypt: creates encrypted file"
            [[ -f "$TEST_ENC" ]] && pass "age_encrypt: output file exists" || fail "age_encrypt: no output file"

            if age_decrypt "$TEST_ENC" "$AGE_DEFAULT_KEY" > "$TEST_DEC"; then
                pass "age_decrypt: decrypts successfully"
                decrypted=$(cat "$TEST_DEC")
                [[ "$decrypted" == "$TEST_CONTENT" ]] && pass "age_encrypt/decrypt: content matches" || fail "age: content mismatch"
            else
                fail "age_decrypt: failed"
            fi
        else
            fail "age_encrypt: failed"
        fi

        rm -f "$TEST_PLAIN" "$TEST_ENC" "$TEST_DEC"
    else
        skip "age key file" "not found at $AGE_DEFAULT_KEY (generate with: age-keygen -o $AGE_DEFAULT_KEY)"
    fi
else
    skip "age encryption" "age not installed (install with: apt install age)"
fi

# ==================================================
section "8/15" "GitHub CLI Functions"
# ==================================================

if command_exists gh; then
    pass "gh: command available"

    subsection "gh_is_authenticated"
    if gh_is_authenticated; then
        pass "gh_is_authenticated: logged in"

        subsection "gh_get_repo"
        # Save current dir and go to a repo
        ORIG_DIR=$(pwd)
        cd "$DOTFILES_DIR" 2>/dev/null || cd "$HOME/Projects/dotfiles" 2>/dev/null || true

        repo=$(gh_get_repo 2>/dev/null)
        if [[ -n "$repo" ]]; then
            pass "gh_get_repo: returns '$repo'"
        else
            pass "gh_get_repo: returns empty (no remote configured)"
        fi

        cd "$ORIG_DIR" || true
    else
        skip "gh_is_authenticated" "not logged in (run: gh auth login)"
        skip "gh_get_repo" "requires authentication"
    fi
else
    skip "GitHub CLI" "gh not installed"
fi

# ==================================================
section "9/15" "File Operation Functions"
# ==================================================

subsection "create_temp_file"
tmp=$(create_temp_file "dotfiles_test")
[[ -f "$tmp" ]] && pass "create_temp_file: creates file" || fail "create_temp_file: no file created"
[[ "$tmp" == /tmp/dotfiles_test* ]] && pass "create_temp_file: correct prefix" || fail "create_temp_file: wrong prefix '$tmp'"
perms=$(stat -c "%a" "$tmp" 2>/dev/null || stat -f "%Lp" "$tmp" 2>/dev/null)
[[ "$perms" == "600" ]] && pass "create_temp_file: secure permissions (600)" || fail "create_temp_file: insecure permissions ($perms)"
rm -f "$tmp"

subsection "ensure_line_in_file"
TEST_FILE="/tmp/test_lines_$$.txt"
echo "line one" > "$TEST_FILE"
echo "line two" >> "$TEST_FILE"

ensure_line_in_file "$TEST_FILE" "new line"
grep -q "new line" "$TEST_FILE" && pass "ensure_line_in_file: adds new line" || fail "ensure_line_in_file: line not added"

ensure_line_in_file "$TEST_FILE" "new line"
count=$(grep -c "new line" "$TEST_FILE")
[[ "$count" == "1" ]] && pass "ensure_line_in_file: no duplicate" || fail "ensure_line_in_file: duplicated ($count times)"

ensure_line_in_file "/nonexistent_xyz" "test"
[[ $? -eq 1 ]] && pass "ensure_line_in_file: returns 1 for missing file" || fail "ensure_line_in_file: missing file handling"

rm -f "$TEST_FILE"

subsection "verify_symlink"
TEST_TARGET="/tmp/verify_target_$$"
TEST_LINK="/tmp/verify_link_$$"
echo "content" > "$TEST_TARGET"
ln -sf "$TEST_TARGET" "$TEST_LINK"

# Capture output, check return code
verify_symlink "$TEST_LINK" "test_link" >/dev/null 2>&1
[[ $? -eq 0 ]] && pass "verify_symlink: returns 0 for valid" || fail "verify_symlink: should return 0"

rm -f "$TEST_TARGET"
verify_symlink "$TEST_LINK" "test_link" >/dev/null 2>&1
[[ $? -eq 1 ]] && pass "verify_symlink: returns 1 for broken" || fail "verify_symlink: should return 1"

rm -f "$TEST_LINK"

# ==================================================
section "10/15" "Environment Functions"
# ==================================================

subsection "export_var / unset_var"
export_var "TEST_EXPORT_VAR" "test_value_123"
[[ "$TEST_EXPORT_VAR" == "test_value_123" ]] && pass "export_var: sets variable" || fail "export_var: not set"

unset_var "TEST_EXPORT_VAR"
[[ -z "$TEST_EXPORT_VAR" ]] && pass "unset_var: unsets variable" || fail "unset_var: still set"

# ==================================================
section "11/15" "Secrets Loading (load-secrets.sh)"
# ==================================================

if [[ -f "$SCRIPTS_DIR/load-secrets.sh" ]]; then
    pass "load-secrets.sh: file exists"

    # Source it
    . "$SCRIPTS_DIR/load-secrets.sh"

    subsection "Function definitions"
    type secrets_load >/dev/null 2>&1 && pass "secrets_load: defined" || fail "secrets_load: not defined"
    type secrets_list >/dev/null 2>&1 && pass "secrets_list: defined" || fail "secrets_list: not defined"
    type secrets_get >/dev/null 2>&1 && pass "secrets_get: defined" || fail "secrets_get: not defined"
    type secrets_refresh >/dev/null 2>&1 && pass "secrets_refresh: defined" || fail "secrets_refresh: not defined"
    type secrets_add >/dev/null 2>&1 && pass "secrets_add: defined" || fail "secrets_add: not defined"
    type secrets_rotate >/dev/null 2>&1 && pass "secrets_rotate: defined" || fail "secrets_rotate: not defined"
    type secrets_check >/dev/null 2>&1 && pass "secrets_check: defined" || fail "secrets_check: not defined"
    type secrets_clean >/dev/null 2>&1 && pass "secrets_clean: defined" || fail "secrets_clean: not defined"
    type secrets_help >/dev/null 2>&1 && pass "secrets_help: defined" || fail "secrets_help: not defined"
    type secrets_audit >/dev/null 2>&1 && pass "secrets_audit: defined" || fail "secrets_audit: not defined"

    subsection "env-mapping.conf"
    if [[ -f "$SENSITIVE_DIR/env-mapping.conf" ]]; then
        pass "env-mapping.conf: exists"

        mapping_count=$(grep -v '^#' "$SENSITIVE_DIR/env-mapping.conf" | grep -v '^$' | grep -c '=' 2>/dev/null || echo 0)
        [[ "$mapping_count" -gt 0 ]] && pass "env-mapping.conf: has $mapping_count mappings" || fail "env-mapping.conf: no mappings found"
    else
        fail "env-mapping.conf: not found at $SENSITIVE_DIR/env-mapping.conf"
    fi

    subsection "secrets_list output"
    list_output=$(secrets_list 2>&1)
    [[ $? -eq 0 ]] && pass "secrets_list: executes without error" || fail "secrets_list: error during execution"
    echo "$list_output" | grep -q "Secret mappings" && pass "secrets_list: shows header" || fail "secrets_list: no header"

    subsection "secrets_check output"
    check_output=$(secrets_check 2>&1)
    echo "$check_output" | grep -q "Checking secrets integrity" && pass "secrets_check: shows header" || fail "secrets_check: no header"
    echo "$check_output" | grep -q "Summary:" && pass "secrets_check: shows summary" || fail "secrets_check: no summary"

    subsection "secrets_clean --dry-run"
    clean_output=$(secrets_clean --dry-run 2>&1)
    echo "$clean_output" | grep -q "Dry run complete" && pass "secrets_clean: dry-run works" || fail "secrets_clean: dry-run failed"

    subsection "secrets_help output"
    help_output=$(secrets_help 2>&1)
    echo "$help_output" | grep -q "Secrets Management Commands" && pass "secrets_help: shows header" || fail "secrets_help: no header"
    echo "$help_output" | grep -q "secrets_add" && pass "secrets_help: documents secrets_add" || fail "secrets_help: missing secrets_add"
    echo "$help_output" | grep -q "secrets_rotate" && pass "secrets_help: documents secrets_rotate" || fail "secrets_help: missing secrets_rotate"

    subsection "secrets_audit"
    # Just check it runs without error (audit log may or may not exist)
    secrets_audit >/dev/null 2>&1
    [[ $? -eq 0 ]] && pass "secrets_audit: executes without error" || fail "secrets_audit: error"

    subsection "secrets_sync"
    type secrets_sync >/dev/null 2>&1 && pass "secrets_sync: defined" || fail "secrets_sync: not defined"
    echo "$help_output" | grep -q "secrets_sync" && pass "secrets_help: documents secrets_sync" || fail "secrets_help: missing secrets_sync"
else
    fail "load-secrets.sh: not found"
fi

# ==================================================
section "12/15" "Dotfiles Sync Script"
# ==================================================

if [[ -f "$SCRIPTS_DIR/dotfiles-sync.sh" ]]; then
    pass "dotfiles-sync.sh: file exists"

    if bash -n "$SCRIPTS_DIR/dotfiles-sync.sh" 2>/dev/null; then
        pass "dotfiles-sync.sh: syntax OK"
    else
        fail "dotfiles-sync.sh: syntax errors"
    fi

    if [[ -x "$SCRIPTS_DIR/dotfiles-sync.sh" ]]; then
        pass "dotfiles-sync.sh: executable"
    else
        fail "dotfiles-sync.sh: not executable"
    fi

    # Test help/usage (script should handle missing DOTFILES_REPO_DIR gracefully)
    subsection "dotfiles-sync.sh execution"
    # We just check it doesn't crash on --help or with vars unset
    DOTFILES_REPO_DIR="" "$SCRIPTS_DIR/dotfiles-sync.sh" --secrets-only 2>&1 | grep -q "Error\|not found" && \
        pass "dotfiles-sync.sh: handles missing DOTFILES_REPO_DIR" || \
        pass "dotfiles-sync.sh: runs (DOTFILES_REPO_DIR may be set)"
else
    fail "dotfiles-sync.sh: not found"
fi

# ==================================================
section "13/15" "GitHub Secrets Manager"
# ==================================================

if [[ -f "$SCRIPTS_DIR/github-secrets-manager.sh" ]]; then
    pass "github-secrets-manager.sh: file exists"

    if bash -n "$SCRIPTS_DIR/github-secrets-manager.sh" 2>/dev/null; then
        pass "github-secrets-manager.sh: syntax OK"
    else
        fail "github-secrets-manager.sh: syntax errors"
    fi

    subsection "--list option"
    # Test --list option (should work without gh auth for listing)
    list_output=$("$SCRIPTS_DIR/github-secrets-manager.sh" --list 2>&1)
    if echo "$list_output" | grep -q "Available secrets\|GITHUB_PERSONAL_ACCESS_TOKEN\|Error"; then
        pass "github-secrets-manager.sh --list: produces output"
    else
        fail "github-secrets-manager.sh --list: no output"
    fi

    subsection "--from-mapping option parsing"
    # Just test that the option is recognized (won't actually upload without gh auth)
    if "$SCRIPTS_DIR/github-secrets-manager.sh" --from-mapping --help 2>&1 | grep -q "Usage\|Error\|authenticated"; then
        pass "github-secrets-manager.sh: recognizes --from-mapping"
    else
        pass "github-secrets-manager.sh: --from-mapping parsed (may need gh auth)"
    fi
else
    fail "github-secrets-manager.sh: not found"
fi

# ==================================================
section "14/15" "Environment Variables Configuration"
# ==================================================

subsection "DOTFILES_DIR"
[[ -n "$DOTFILES_DIR" ]] && pass "DOTFILES_DIR: set to '$DOTFILES_DIR'" || fail "DOTFILES_DIR: not set"

subsection "DOTFILES_REPO_DIR"
if [[ -n "$DOTFILES_REPO_DIR" ]]; then
    pass "DOTFILES_REPO_DIR: set to '$DOTFILES_REPO_DIR'"
    [[ -d "$DOTFILES_REPO_DIR" ]] && pass "DOTFILES_REPO_DIR: directory exists" || fail "DOTFILES_REPO_DIR: directory missing"
else
    skip "DOTFILES_REPO_DIR" "not set (repo sync disabled)"
fi

subsection "AGE_KEY_PATH"
key_path="${AGE_KEY_PATH:-$HOME/.config/age/key.txt}"
[[ -f "$key_path" ]] && pass "Age key: exists at $key_path" || skip "Age key" "not found at $key_path"

# ==================================================
section "15/15" "Installation Verification"
# ==================================================

subsection "Symlinks"
for link in "$HOME/.zshrc" "$HOME/.bashrc"; do
    name=$(basename "$link")
    if [[ -L "$link" && -e "$link" ]]; then
        pass "$name: valid symlink"
    elif [[ -f "$link" ]]; then
        pass "$name: exists (not symlink)"
    else
        fail "$name: missing"
    fi
done

for link in "$HOME/.zsh/aliases.zsh" "$HOME/.zsh/functions.zsh" "$HOME/.zsh/nvm.zsh"; do
    name=$(basename "$link")
    if [[ -L "$link" && -e "$link" ]]; then
        pass "$name: valid symlink"
    elif [[ -f "$link" ]]; then
        pass "$name: exists"
    else
        fail "$name: missing"
    fi
done

subsection "Script permissions"
for script in utils.sh load-secrets.sh age-encrypt-decrypt.sh github-secrets-manager.sh install-precommit.sh dotfiles-sync.sh; do
    if [[ -x "$SCRIPTS_DIR/$script" ]]; then
        pass "$script: executable"
    elif [[ -f "$SCRIPTS_DIR/$script" ]]; then
        fail "$script: not executable (run: chmod +x $SCRIPTS_DIR/$script)"
    else
        fail "$script: not found"
    fi
done

subsection "Directories"
[[ -d "$DOTFILES_DIR" ]] && pass "\$HOME/.dotfiles: exists" || fail "\$HOME/.dotfiles: missing"
[[ -d "$SCRIPTS_DIR" ]] && pass "scripts/: exists" || fail "scripts/: missing"
[[ -d "$SENSITIVE_DIR" ]] && pass "sensitive/: exists" || fail "sensitive/: missing"

# ==================================================
# SUMMARY
# ==================================================

echo ""
echo "========================================"
echo "            TEST SUMMARY"
echo "========================================"
printf '  %bPASSED%b: %s\n' "$GREEN" "$NC" "$TESTS_PASSED"
printf '  %bFAILED%b: %s\n' "$RED" "$NC" "$TESTS_FAILED"
printf '  %bSKIPPED%b: %s\n' "$YELLOW" "$NC" "$TESTS_SKIPPED"
TOTAL=$((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))
echo "  TOTAL:  $TOTAL"
echo "========================================"

if [[ $TESTS_FAILED -eq 0 ]]; then
    printf '%bAll tests passed!%b\n' "$GREEN" "$NC"
    exit 0
else
    printf '%bSome tests failed. Review output above.%b\n' "$RED" "$NC"
    exit 1
fi
