#!/usr/bin/env bats
# Tests for scripts/utils.sh - foundation utility library

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    source "$SCRIPTS_DIR/utils.sh"
}

# --- Logging functions ---

@test "log_info outputs INFO tag" {
    run log_info "test message"
    [[ "$output" == *"[INFO]"* ]]
    [[ "$output" == *"test message"* ]]
}

@test "log_success outputs SUCCESS tag" {
    run log_success "done"
    [[ "$output" == *"[SUCCESS]"* ]]
    [[ "$output" == *"done"* ]]
}

@test "log_warning outputs WARNING tag" {
    run log_warning "caution"
    [[ "$output" == *"[WARNING]"* ]]
    [[ "$output" == *"caution"* ]]
}

@test "log_error outputs ERROR tag" {
    run log_error "bad"
    [[ "$output" == *"[ERROR]"* ]]
    [[ "$output" == *"bad"* ]]
}

@test "exit_error exits with code 1 by default" {
    run bash -c 'source "$1/utils.sh"; exit_error "fail"' -- "$SCRIPTS_DIR"
    [[ $status -eq 1 ]]
}

@test "exit_error exits with custom code" {
    run bash -c 'source "$1/utils.sh"; exit_error "fail" 42' -- "$SCRIPTS_DIR"
    [[ $status -eq 42 ]]
}

# --- Silent check functions ---

@test "command_exists finds bash" {
    command_exists bash
}

@test "command_exists returns false for missing command" {
    ! command_exists __nonexistent_cmd_xyz__
}

@test "file_exists finds real file" {
    file_exists "$SCRIPTS_DIR/utils.sh"
}

@test "file_exists returns false for missing file" {
    ! file_exists "/nonexistent/file_xyz.txt"
}

@test "dir_exists finds real directory" {
    dir_exists "$SCRIPTS_DIR"
}

@test "dir_exists returns false for missing directory" {
    ! dir_exists "/nonexistent/dir_xyz"
}

@test "symlink_valid detects valid symlink" {
    local target="/tmp/bats_target_$$"
    local link="/tmp/bats_link_$$"
    echo "content" > "$target"
    ln -sf "$target" "$link"
    symlink_valid "$link"
    rm -f "$target" "$link"
}

@test "symlink_valid detects broken symlink" {
    local target="/tmp/bats_target_$$"
    local link="/tmp/bats_link_$$"
    echo "content" > "$target"
    ln -sf "$target" "$link"
    rm -f "$target"
    ! symlink_valid "$link"
    rm -f "$link"
}

@test "var_is_set detects set variable" {
    TEST_SET_VAR="hello"
    var_is_set TEST_SET_VAR
}

@test "var_is_set returns false for unset variable" {
    unset TEST_UNSET_VAR 2>/dev/null
    ! var_is_set TEST_UNSET_VAR
}

@test "var_is_set returns false for empty variable" {
    TEST_EMPTY_VAR=""
    ! var_is_set TEST_EMPTY_VAR
}

# --- String helpers ---

@test "trim removes leading/trailing spaces" {
    result=$(trim "  hello  ")
    [[ "$result" == "hello" ]]
}

@test "trim preserves inner spaces" {
    result=$(trim "  hello world  ")
    [[ "$result" == "hello world" ]]
}

@test "mask_value full mask" {
    result=$(mask_value "secret")
    [[ "$result" == "******" ]]
}

@test "mask_value partial mask" {
    result=$(mask_value "secret123" 3)
    [[ "$result" == "sec******" ]]
}

@test "base64 roundtrip" {
    encoded=$(base64_encode "test string 123")
    decoded=$(base64_decode "$encoded")
    [[ "$decoded" == "test string 123" ]]
}

# --- Config parsing ---

@test "parse_mapping_file parses KEY=VALUE correctly" {
    local conf="/tmp/bats_conf_$$.conf"
    cat > "$conf" << 'EOF'
# Comment
KEY1=value1
KEY2=value2
EOF
    PARSED_KEYS=()
    test_handler() { PARSED_KEYS+=("$1"); }
    count=$(parse_mapping_file "$conf" test_handler)
    [[ "$count" == "2" ]]
    rm -f "$conf"
}

@test "parse_mapping_file returns 1 for missing file" {
    run parse_mapping_file "/nonexistent_xyz" echo
    [[ $status -eq 1 ]]
}

# --- Counter functions ---

@test "init_counter starts at 0" {
    init_counter "test_ctr"
    [[ "$(get_counter test_ctr)" == "0" ]]
}

@test "increment_counter increments" {
    init_counter "test_ctr"
    increment_counter "test_ctr"
    [[ "$(get_counter test_ctr)" == "1" ]]
    increment_counter "test_ctr"
    increment_counter "test_ctr"
    [[ "$(get_counter test_ctr)" == "3" ]]
}

# --- File operations ---

@test "create_temp_file creates file with correct prefix" {
    tmp=$(create_temp_file "bats_test")
    [[ -f "$tmp" ]]
    [[ "$tmp" == /tmp/bats_test* ]]
    perms=$(stat -c "%a" "$tmp" 2>/dev/null || stat -f "%Lp" "$tmp" 2>/dev/null)
    [[ "$perms" == "600" ]]
    rm -f "$tmp"
}

@test "ensure_line_in_file adds line" {
    local f="/tmp/bats_lines_$$.txt"
    echo "line one" > "$f"
    ensure_line_in_file "$f" "new line"
    grep -q "new line" "$f"
    rm -f "$f"
}

@test "ensure_line_in_file prevents duplicates" {
    local f="/tmp/bats_lines_$$.txt"
    echo "line one" > "$f"
    ensure_line_in_file "$f" "new line"
    # Second call returns 1 (line already exists) - that's correct behavior
    ensure_line_in_file "$f" "new line" || true
    count=$(grep -c "new line" "$f")
    [[ "$count" == "1" ]]
    rm -f "$f"
}

# --- Environment functions ---

@test "export_var sets variable" {
    export_var "BATS_TEST_VAR" "val123"
    [[ "$BATS_TEST_VAR" == "val123" ]]
}

@test "unset_var removes variable" {
    export BATS_TEST_VAR2="val"
    unset_var "BATS_TEST_VAR2"
    [[ -z "$BATS_TEST_VAR2" ]]
}

# --- Zsh compatibility ---

@test "utils.sh sources under zsh without error" {
    run zsh -c ". '$SCRIPTS_DIR/utils.sh'"
    [[ $status -eq 0 ]]
}

@test "log functions work under zsh" {
    run zsh -c ". '$SCRIPTS_DIR/utils.sh'; log_info 'zsh test'"
    [[ "$output" == *"[INFO]"* ]]
    [[ "$output" == *"zsh test"* ]]
}

@test "command_exists works under zsh" {
    run zsh -c ". '$SCRIPTS_DIR/utils.sh'; command_exists zsh && echo yes || echo no"
    [[ "$output" == *"yes"* ]]
}

@test "var_is_set works under zsh" {
    run zsh -c ". '$SCRIPTS_DIR/utils.sh'; MY_VAR=hello; var_is_set MY_VAR && echo yes || echo no"
    [[ "$output" == *"yes"* ]]
}

@test "counter functions work under zsh" {
    run zsh -c ". '$SCRIPTS_DIR/utils.sh'; init_counter cnt; increment_counter cnt; echo \$(get_counter cnt)"
    [[ "$output" == *"1"* ]]
}

@test "trim works under zsh" {
    run zsh -c ". '$SCRIPTS_DIR/utils.sh'; echo \$(trim '  hello  ')"
    [[ "$output" == *"hello"* ]]
}

@test "mask_value works under zsh" {
    run zsh -c ". '$SCRIPTS_DIR/utils.sh'; echo \$(mask_value secret)"
    [[ "$output" == *"******"* ]]
}

@test "parse_mapping_file works under zsh" {
    local conf="/tmp/bats_zsh_conf_$$.conf"
    cat > "$conf" << 'EOF'
KEY1=val1
KEY2=val2
EOF
    run zsh -c ". '$SCRIPTS_DIR/utils.sh'; count=\$(parse_mapping_file '$conf' echo); echo \$count"
    [[ "$output" == *"2"* ]]
    rm -f "$conf"
}
