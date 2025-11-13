#!/usr/bin/env bash

# Test Shell Aliases

# Tests that aliases are properly defined and functional


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

test_bash_aliases() {
    log_info "Testing bash aliases..."

    # Source bashrc
    if [ -f "$DOTFILES_DIR/bash/.bashrc" ]; then
        # Create a temporary bash environment (interactive to load aliases)
        bash -i -c "
            source '$DOTFILES_DIR/bash/.bashrc' 2>/dev/null || true

            # Check if basic aliases exist
            if alias ll &>/dev/null && alias k &>/dev/null; then
                exit 0
            else
                exit 1
            fi
        " 2>/dev/null

        if [ $? -eq 0 ]; then
            log_success "Bash aliases loaded correctly"
            return 0
        else
            log_error "Bash aliases not loaded"
            return 1
        fi
    else
        log_error "bash/.bashrc not found"
        return 1
    fi
}

test_zsh_aliases() {
    log_info "Testing zsh aliases..."

    # Skip if zsh not available
    if ! command -v zsh &> /dev/null; then
        log_info "Zsh not available, skipping zsh alias tests"
        return 0
    fi

    # Check if aliases file exists
    if [ -f "$DOTFILES_DIR/zsh/.zsh/aliases.zsh" ]; then
        # Create a temporary zsh environment
        zsh -c "
            source '$DOTFILES_DIR/zsh/.zsh/aliases.zsh' 2>/dev/null || true

            # Check if basic aliases exist
            if alias ll &>/dev/null && alias k &>/dev/null; then
                exit 0
            else
                exit 1
            fi
        "

        if [ $? -eq 0 ]; then
            log_success "Zsh aliases loaded correctly"
            return 0
        else
            log_error "Zsh aliases not loaded"
            return 1
        fi
    else
        log_error "zsh/.zsh/aliases.zsh not found"
        return 1
    fi
}

test_common_aliases() {
    log_info "Testing common DevOps aliases..."

    local aliases_file="$DOTFILES_DIR/zsh/.zsh/aliases.zsh"

    if [ ! -f "$aliases_file" ]; then
        log_error "Aliases file not found"
        return 1
    fi

    # Check for common aliases
    local required_aliases=(
        "alias k="
        "alias h="
        "alias tf="
    )

    local all_found=true
    for alias_def in "${required_aliases[@]}"; do
        if ! grep -q "$alias_def" "$aliases_file"; then
            log_error "Alias not found: $alias_def"
            all_found=false
        fi
    done

    if $all_found; then
        log_success "Common aliases defined correctly"
        return 0
    else
        log_error "Some common aliases missing"
        return 1
    fi
}


# Main

main() {
    local failed=0

    test_bash_aliases || ((failed++))
    test_zsh_aliases || ((failed++))
    test_common_aliases || ((failed++))

    if [ $failed -eq 0 ]; then
        log_success "All alias tests passed"
        exit 0
    else
        log_error "$failed alias test(s) failed"
        exit 1
    fi
}

main "$@"
