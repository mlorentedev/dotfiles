#!/usr/bin/env bash
# =============================================================================
# Run All Tests
# =============================================================================
# Comprehensive test suite for dotfiles installation
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOTFILES_DIR="$(dirname "$SCRIPT_DIR")"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# =============================================================================
# Logging
# =============================================================================
log_test_start() {
    echo -e "\n${BLUE}==> Testing: $1${NC}"
}

log_test_pass() {
    echo -e "${GREEN}✓ PASS${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
}

log_test_fail() {
    echo -e "${RED}✗ FAIL${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# =============================================================================
# Test Execution
# =============================================================================
run_test_script() {
    local script="$1"
    local script_path="$SCRIPT_DIR/$script"

    if [ ! -f "$script_path" ]; then
        log_error "Test script not found: $script"
        return 1
    fi

    log_test_start "$script"

    if bash "$script_path"; then
        log_test_pass "$script"
        return 0
    else
        log_test_fail "$script"
        return 1
    fi
}

# =============================================================================
# Pre-flight Checks
# =============================================================================
preflight_checks() {
    log_info "Running pre-flight checks..."

    # Check if we're in the dotfiles directory
    if [ ! -f "$DOTFILES_DIR/bootstrap.sh" ]; then
        log_error "Not in dotfiles directory"
        exit 1
    fi

    # Check required commands
    local missing=()
    for cmd in bash git; do
        if ! command -v "$cmd" &> /dev/null; then
            missing+=("$cmd")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Missing required commands: ${missing[*]}"
        exit 1
    fi

    log_success "Pre-flight checks passed"
}

# =============================================================================
# Individual Test Categories
# =============================================================================

test_directory_structure() {
    log_test_start "Directory Structure"

    local required_dirs=(
        "bash"
        "zsh"
        "git"
        "shell-common"
        "scripts"
        "test"
        "tools"
    )

    local all_exist=true
    for dir in "${required_dirs[@]}"; do
        if [ ! -d "$DOTFILES_DIR/$dir" ]; then
            log_error "Missing directory: $dir"
            all_exist=false
        fi
    done

    if $all_exist; then
        log_test_pass "All required directories exist"
        return 0
    else
        log_test_fail "Some directories are missing"
        return 0  # Don't fail the whole script
    fi
}

test_core_files() {
    log_test_start "Core Files"

    local required_files=(
        "bash/.bashrc"
        "bash/.bash_profile"
        "zsh/.zshrc"
        "git/.gitconfig"
        "shell-common/.profile"
        "bootstrap.sh"
    )

    local all_exist=true
    for file in "${required_files[@]}"; do
        if [ ! -f "$DOTFILES_DIR/$file" ]; then
            log_error "Missing file: $file"
            all_exist=false
        fi
    done

    if $all_exist; then
        log_test_pass "All core files exist"
        return 0
    else
        log_test_fail "Some core files are missing"
        return 0  # Don't fail the whole script
    fi
}

test_scripts_executable() {
    log_test_start "Scripts Executable"

    local all_executable=true

    # Check bootstrap.sh
    if [ ! -x "$DOTFILES_DIR/bootstrap.sh" ]; then
        log_error "bootstrap.sh is not executable"
        all_executable=false
    fi

    # Check tool installation scripts
    for script in "$DOTFILES_DIR"/tools/*.sh; do
        if [ -f "$script" ] && [ ! -x "$script" ]; then
            log_error "$(basename "$script") is not executable"
            all_executable=false
        fi
    done

    if $all_executable; then
        log_test_pass "All scripts are executable"
        return 0
    else
        log_test_fail "Some scripts are not executable"
        return 0  # Don't fail the whole script
    fi
}

test_shell_syntax() {
    log_test_start "Shell Script Syntax"

    local all_valid=true

    # Check bash files
    for file in "$DOTFILES_DIR"/bash/.bashrc "$DOTFILES_DIR"/bash/.bash_profile; do
        if [ -f "$file" ]; then
            if ! bash -n "$file" 2>/dev/null; then
                log_error "Syntax error in $(basename "$file")"
                all_valid=false
            fi
        fi
    done

    # Check zsh files
    for file in "$DOTFILES_DIR"/zsh/.zshrc "$DOTFILES_DIR"/zsh/.zsh/*.zsh; do
        if [ -f "$file" ]; then
            if ! zsh -n "$file" 2>/dev/null; then
                # If zsh not available, skip
                if command -v zsh &> /dev/null; then
                    log_error "Syntax error in $(basename "$file")"
                    all_valid=false
                fi
            fi
        fi
    done

    # Check shell scripts
    for script in "$DOTFILES_DIR"/*.sh "$DOTFILES_DIR"/tools/*.sh; do
        if [ -f "$script" ]; then
            if ! bash -n "$script" 2>/dev/null; then
                log_error "Syntax error in $(basename "$script")"
                all_valid=false
            fi
        fi
    done

    if $all_valid; then
        log_test_pass "All shell scripts have valid syntax"
        return 0
    else
        log_test_fail "Some shell scripts have syntax errors"
        return 0  # Don't fail the whole script
    fi
}

# =============================================================================
# Summary
# =============================================================================
print_summary() {
    echo ""
    echo "=========================================="
    echo "           Test Results Summary"
    echo "=========================================="
    echo ""
    echo "Total Tests:  $TESTS_TOTAL"
    echo -e "Passed:       ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed:       ${RED}$TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        return 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        return 1
    fi
}

# =============================================================================
# Main
# =============================================================================
main() {
    echo -e "${BLUE}"
    cat << "EOF"
╔═══════════════════════════════════════════════════════════╗
║              Dotfiles Test Suite                         ║
╚═══════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"

    preflight_checks

    echo ""
    echo "Starting tests..."
    echo ""

    # Run built-in tests
    test_directory_structure
    test_core_files
    test_scripts_executable
    test_shell_syntax

    # Run test scripts if they exist
    local test_scripts=(
        "test-stow.sh"
        "test-aliases.sh"
        "test-path.sh"
    )

    for script in "${test_scripts[@]}"; do
        if [ -f "$SCRIPT_DIR/$script" ]; then
            run_test_script "$script" || true
        fi
    done

    # Print summary
    print_summary
}

main "$@"
