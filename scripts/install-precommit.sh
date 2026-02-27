#!/bin/bash

# Installs pre-commit hooks for the current git repository
# Usage: ./install-precommit.sh

set -euo pipefail

# Source utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
. "$SCRIPT_DIR/utils.sh"

log_info "Installing pre-commit dependencies..."

# Check if pre-commit is available, install if needed
if ! command -v pre-commit >/dev/null 2>&1; then
  log_info "Installing pre-commit with pip"
  
  if command -v pip >/dev/null 2>&1; then
    pip install pre-commit
  elif command -v pip3 >/dev/null 2>&1; then
    pip3 install pre-commit
  else
    exit_error "pip is not available. Please install pip first."
  fi
else
  log_info "pre-commit is already installed"
fi

# Check if we're in a git repository
check_git_repo

# Install hooks, including special types
log_info "Installing pre-commit hooks..."
if pre-commit install --hook-type prepare-commit-msg --hook-type commit-msg; then
  log_success "Pre-commit hooks installed successfully."
else
  exit_error "Failed to install pre-commit hooks."
fi

# Show installed hooks
log_info "Installed hooks:"
pre-commit --version
