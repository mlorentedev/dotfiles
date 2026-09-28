#!/usr/bin/env bash

# Push the dotfiles repo, then copy it to the installation (ADR-005)
# Usage: dotfiles-sync.sh
#
# sensitive/ is never touched, in either direction. Secrets live in Bitwarden
# behind secrets/registry.yaml (ADR-028); the one age blob left is git-tracked
# and deployed by setup. A two-way copy of sensitive/ used to live here, and it
# copied retired blobs from the installation back into the repo (#1795).

set -euo pipefail

# Configuration
DOTFILES_LOCAL="${DOTFILES_DIR:-$HOME/.dotfiles}"
DOTFILES_REPO="${DOTFILES_REPO_DIR:-$HOME/Projects/dotfiles}"

# Colors (if terminal supports it)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED='' GREEN='' YELLOW='' BLUE='' NC=''
fi

log_info()    { printf '%b→%b %s\n' "$BLUE" "$NC" "$1"; }
log_success() { printf '%b✓%b %s\n' "$GREEN" "$NC" "$1"; }
log_warning() { printf '%b!%b %s\n' "$YELLOW" "$NC" "$1"; }
log_error()   { printf '%b✗%b %s\n' "$RED" "$NC" "$1" >&2; }

# Validate directories exist
validate_dirs() {
    [[ -d "$DOTFILES_LOCAL" ]] || { log_error "Local dotfiles not found: $DOTFILES_LOCAL"; return 1; }
    [[ -d "$DOTFILES_REPO" ]] || { log_error "Repo dotfiles not found: $DOTFILES_REPO"; return 1; }
    [[ "$DOTFILES_LOCAL" != "$DOTFILES_REPO" ]] || { log_warning "Local and repo are same directory"; return 1; }
}

# Push repo to remote, then copy files to local installation (ADR-005)
sync_repo_to_local() {
    log_info "Pushing from repo..."
    if git -C "$DOTFILES_REPO" diff --quiet && git -C "$DOTFILES_REPO" diff --cached --quiet; then
        log_info "  No changes to push"
    else
        log_warning "  Uncommitted changes in repo - commit first"
        return 1
    fi

    if git -C "$DOTFILES_REPO" push 2>/dev/null; then
        log_success "Push complete"
    else
        log_warning "  Nothing to push or push failed"
    fi

    echo ""
    log_info "Copying repo → local..."
    if rsync -a --delete \
        --exclude='.git/' \
        --exclude='sensitive/' \
        "$DOTFILES_REPO/" "$DOTFILES_LOCAL/"; then
        log_success "Copy complete"
    else
        log_error "Copy failed"
        return 1
    fi
}

# Main
main() {
    if [[ $# -gt 0 ]]; then
        log_error "dotfiles-sync takes no arguments. --secrets-only is gone: sensitive/ no longer syncs here (#1795)."
        exit 2
    fi
    echo "Dotfiles Sync"
    echo "============="
    echo "Local: $DOTFILES_LOCAL"
    echo "Repo:  $DOTFILES_REPO"
    echo ""

    validate_dirs || exit 1

    # Push repo + copy to local (ADR-005: ~/.dotfiles is not a git repo)
    sync_repo_to_local || exit 1

    echo ""
    log_success "Sync complete. Run 'source ~/.zshrc' or 'source ~/.bashrc' to reload."
}

main "$@"
