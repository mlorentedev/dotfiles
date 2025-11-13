#!/usr/bin/env bash

# Test PATH Configuration

# Tests that PATH is configured correctly with no duplicates


set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOTFILES_DIR="$(dirname "$SCRIPT_DIR")"

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


# Tests

test_bash_path() {
    log_info "Testing bash PATH configuration..."

    # Create ~/.local/bin if it doesn't exist for testing
    mkdir -p "$HOME/.local/bin"

    # Test with interactive bash to ensure .bashrc loads fully
    if bash -i -c "
        source '$DOTFILES_DIR/bash/.bashrc' 2>/dev/null || true

        # Check if ~/.local/bin is in PATH
        if [[ \":\$PATH:\" == *\":$HOME/.local/bin:\"* ]]; then
            echo 'PATH_OK'
        fi
    " 2>/dev/null | grep -q "PATH_OK"; then
        log_success "Bash PATH includes ~/.local/bin"
        return 0
    else
        log_error "Bash PATH missing ~/.local/bin"
        return 1
    fi
}

test_zsh_path() {
    log_info "Testing zsh PATH configuration..."

    # Skip if zsh not available
    if ! command -v zsh &> /dev/null; then
        log_info "Zsh not available, skipping zsh PATH tests"
        return 0
    fi

    # Source zshrc and check PATH
    if zsh -c "
        source '$DOTFILES_DIR/zsh/.zshrc' 2>/dev/null || true

        # Check if ~/.local/bin is in PATH
        if [[ \":\$PATH:\" == *\":$HOME/.local/bin:\"* ]]; then
            echo 'PATH_OK'
        fi
    " | grep -q "PATH_OK"; then
        log_success "Zsh PATH includes ~/.local/bin"
        return 0
    else
        log_error "Zsh PATH missing ~/.local/bin"
        return 1
    fi
}

test_no_duplicates_zsh() {
    log_info "Testing zsh PATH for duplicates..."

    # Skip if zsh not available
    if ! command -v zsh &> /dev/null; then
        log_info "Zsh not available, skipping duplicate tests"
        return 0
    fi

    # Source zshrc multiple times and check for duplicates
    if zsh -c "
        source '$DOTFILES_DIR/zsh/.zshrc' 2>/dev/null || true
        source '$DOTFILES_DIR/zsh/.zshrc' 2>/dev/null || true
        source '$DOTFILES_DIR/zsh/.zshrc' 2>/dev/null || true

        # Count occurrences of .local/bin in PATH (escape the dot)
        count=\$(echo \"\$PATH\" | tr ':' '\n' | grep -c '\.local/bin' || echo 0)

        if [ \"\$count\" -le 1 ]; then
            echo 'NO_DUPLICATES'
        fi
    " 2>/dev/null | grep -q "NO_DUPLICATES"; then
        log_success "Zsh PATH has no duplicates"
        return 0
    else
        log_error "Zsh PATH has duplicates"
        return 1
    fi
}


# Main

main() {
    local failed=0

    test_bash_path || ((failed++))
    test_zsh_path || ((failed++))
    test_no_duplicates_zsh || ((failed++))

    if [ $failed -eq 0 ]; then
        log_success "All PATH tests passed"
        exit 0
    else
        log_error "$failed PATH test(s) failed"
        exit 1
    fi
}

main "$@"
