#!/usr/bin/env bash

# Test GNU Stow Functionality

# Tests symlink creation and module installation


set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOTFILES_DIR="$(dirname "$SCRIPT_DIR")"
TEST_HOME="/tmp/dotfiles-test-home-$$"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

log_info() {
    echo -e "[INFO] $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    if [ -d "$TEST_HOME" ]; then
        rm -rf "$TEST_HOME"
    fi
}

trap cleanup EXIT


# Tests

test_stow_bash_module() {
    log_info "Testing bash module stow..."

    mkdir -p "$TEST_HOME"
    cd "$DOTFILES_DIR"

    # Try to stow bash module
    if HOME="$TEST_HOME" stow -t "$TEST_HOME" bash 2>&1; then
        # Check if symlinks were created
        if [ -L "$TEST_HOME/.bashrc" ] && [ -L "$TEST_HOME/.bash_profile" ]; then
            log_success "bash module stowed correctly"
            return 0
        else
            log_error "Symlinks not created for bash module"
            return 1
        fi
    else
        log_error "Failed to stow bash module"
        return 1
    fi
}

test_stow_zsh_module() {
    log_info "Testing zsh module stow..."

    mkdir -p "$TEST_HOME"
    cd "$DOTFILES_DIR"

    # Try to stow zsh module
    if HOME="$TEST_HOME" stow -t "$TEST_HOME" zsh 2>&1; then
        # Check if symlinks were created
        if [ -L "$TEST_HOME/.zshrc" ]; then
            log_success "zsh module stowed correctly"
            return 0
        else
            log_error "Symlinks not created for zsh module"
            return 1
        fi
    else
        log_error "Failed to stow zsh module"
        return 1
    fi
}

test_stow_scripts_module() {
    log_info "Testing scripts module stow..."

    mkdir -p "$TEST_HOME"
    cd "$DOTFILES_DIR"

    # Try to stow scripts module
    if HOME="$TEST_HOME" stow -t "$TEST_HOME" scripts 2>&1; then
        # Check if .local/bin exists and has scripts
        if [ -d "$TEST_HOME/.local/bin" ]; then
            log_success "scripts module stowed correctly"
            return 0
        else
            log_error ".local/bin not created for scripts module"
            return 1
        fi
    else
        log_error "Failed to stow scripts module"
        return 1
    fi
}

test_stow_conflicts() {
    log_info "Testing stow conflict detection..."

    mkdir -p "$TEST_HOME"

    # Create a conflicting file
    echo "existing content" > "$TEST_HOME/.bashrc"

    cd "$DOTFILES_DIR"

    # Try to stow - should detect conflict
    if HOME="$TEST_HOME" stow -t "$TEST_HOME" bash 2>&1 | grep -q "conflict"; then
        log_success "Conflict detected correctly"
        return 0
    else
        log_error "Conflict not detected"
        return 1
    fi
}


# Main

main() {
    # Check if stow is available
    if ! command -v stow &> /dev/null; then
        log_error "GNU Stow not found, skipping stow tests"
        exit 0
    fi

    local failed=0

    test_stow_bash_module || ((failed++))
    cleanup

    test_stow_zsh_module || ((failed++))
    cleanup

    test_stow_scripts_module || ((failed++))
    cleanup

    test_stow_conflicts || ((failed++))
    cleanup

    if [ $failed -eq 0 ]; then
        log_success "All stow tests passed"
        exit 0
    else
        log_error "$failed stow test(s) failed"
        exit 1
    fi
}

main "$@"
