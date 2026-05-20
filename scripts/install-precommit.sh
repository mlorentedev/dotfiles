#!/bin/bash

# Installs pre-commit hooks for the current git repository.
#
# Usage: ./install-precommit.sh [--with-sdd-gate]
#
#   --with-sdd-gate   Also install the pre-push hook that runs the SDD
#                     spec-gate locally (catches violations before push).
#                     Opt-in; CI enforces the gate regardless.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
. "$SCRIPT_DIR/utils.sh"

WITH_SDD_GATE=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --with-sdd-gate) WITH_SDD_GATE=1; shift ;;
        -h|--help)
            grep '^#' "$0" | sed 's/^# \{0,1\}//'
            exit 0
            ;;
        *) exit_error "Unknown argument: $1" ;;
    esac
done

log_info "Installing pre-commit dependencies..."

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

check_git_repo

log_info "Installing pre-commit hooks..."
HOOK_TYPES=("--hook-type" "prepare-commit-msg" "--hook-type" "commit-msg")
if [[ "$WITH_SDD_GATE" -eq 1 ]]; then
    log_info "  + SDD spec-gate pre-push hook (opt-in via --with-sdd-gate)"
    HOOK_TYPES+=("--hook-type" "pre-push")
fi

if pre-commit install "${HOOK_TYPES[@]}"; then
    log_success "Pre-commit hooks installed successfully."
else
    exit_error "Failed to install pre-commit hooks."
fi

log_info "Installed hooks:"
pre-commit --version
