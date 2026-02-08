#!/usr/bin/env bash

# Bidirectional sync between dotfiles repo and installation
# Usage: dotfiles-sync.sh [--secrets-only]
#
# Syncs:
#   - Git changes (push from repo, pull to local)
#   - Secrets: *.secret.age, env-mapping.conf, .secrets-audit.log

set -e

# Configuration
DOTFILES_LOCAL="${DOTFILES_DIR:-$HOME/.dotfiles}"
DOTFILES_REPO="${DOTFILES_REPO_DIR:-$HOME/Projects/dotfiles}"
SENSITIVE_DIR="sensitive"

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

# Sync secrets bidirectionally (newest wins)
sync_secrets() {
    local_dir="$DOTFILES_LOCAL/$SENSITIVE_DIR"
    repo_dir="$DOTFILES_REPO/$SENSITIVE_DIR"

    [[ -d "$local_dir" ]] || { log_warning "Local sensitive dir not found"; return 0; }
    [[ -d "$repo_dir" ]] || { log_warning "Repo sensitive dir not found"; return 0; }

    log_info "Syncing secrets..."

    synced=0

    # Sync .age files, env-mapping.conf, and audit log
    files_to_sync=(
        "env-mapping.conf"
        ".secrets-audit.log"
    )

    # Add all .age files from both directories
    for f in "$local_dir"/*.secret.age "$repo_dir"/*.secret.age; do
        [[ -f "$f" ]] || continue
        basename="${f##*/}"
        # Add to array if not already present
        found=0
        for existing in "${files_to_sync[@]}"; do
            [[ "$existing" == "$basename" ]] && { found=1; break; }
        done
        [[ $found -eq 0 ]] && files_to_sync+=("$basename")
    done

    for file in "${files_to_sync[@]}"; do
        local_file="$local_dir/$file"
        repo_file="$repo_dir/$file"

        # Both exist - sync newer to older
        if [[ -f "$local_file" && -f "$repo_file" ]]; then
            if [[ "$local_file" -nt "$repo_file" ]]; then
                cat "$local_file" > "$repo_file"
                log_info "  $file (→ repo)"
                synced=$((synced + 1))
            elif [[ "$repo_file" -nt "$local_file" ]]; then
                cat "$repo_file" > "$local_file"
                log_info "  $file (repo → local)"
                synced=$((synced + 1))
            fi
        # Only in - copy to repo
        elif [[ -f "$local_file" ]]; then
            cat "$local_file" > "$repo_file"
            log_info "  $file (→ repo) [new]"
            synced=$((synced + 1))
        # Only in repo - copy to local
        elif [[ -f "$repo_file" ]]; then
            cat "$repo_file" > "$local_file"
            log_info "  $file (repo → local) [new]"
            synced=$((synced + 1))
        fi
    done

    [[ $synced -eq 0 ]] && log_info "  All secrets already in sync"
    return 0
}

# Sync git repos
sync_git() {
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
    log_info "Pulling to local..."
    if git -C "$DOTFILES_LOCAL" pull; then
        log_success "Pull complete"
    else
        log_error "Pull failed"
        return 1
    fi
}

# Main
main() {
    echo "Dotfiles Sync"
    echo "============="
    echo "Local: $DOTFILES_LOCAL"
    echo "Repo:  $DOTFILES_REPO"
    echo ""

    validate_dirs || exit 1

    # Sync secrets first (bidirectional)
    sync_secrets || exit 1
    echo ""

    # If --secrets-only, stop here
    [[ "$1" == "--secrets-only" ]] && { log_success "Secrets sync complete"; exit 0; }

    # Sync git
    sync_git || exit 1

    echo ""
    log_success "Sync complete. Run 'source ~/.zshrc' or 'source ~/.bashrc' to reload."
}

main "$@"
