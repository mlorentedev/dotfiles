#!/bin/bash

# Uploads environment variables from .env files to GitHub repository secrets
# Usage: ./github-secrets-manager.sh [ENV_PATH]
# Without ENV_PATH, checks default locations

set -e

# Source utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/utils.sh"

# Check dependencies
require_command "gh" "GitHub CLI" "Visit: https://cli.github.com/"
gh_is_authenticated || exit_error "Not authenticated with GitHub CLI. Run 'gh auth login' first."

# Get repository
REPO=$(gh_get_repo)
[[ -z "$REPO" ]] && exit_error "Could not determine repository. Make sure you're in a Git repository directory."

log_success "Configuring secrets for repository: $REPO"

# Define environment files to process
if [[ -n "$1" ]]; then
    ENV_FILES=("$1")
    log_info "Using custom environment file: $1"
else
    PROJECT_ROOT="$(get_project_root "$SCRIPT_DIR")"
    ENV_FILES=(
        "$PROJECT_ROOT/.env"
        "$PROJECT_ROOT/.env.local"
        "$PROJECT_ROOT/.env.production"
        "$PROJECT_ROOT/.env.development"
        "$PROJECT_ROOT/.env.staging"
    )
    log_info "Using default environment file paths"
fi

# Track processed secrets to avoid duplicates
declare -a PROCESSED_SECRETS=()

# Handler for each env entry
process_secret() {
    local key="$1"
    local value="$2"

    # Skip duplicates
    for processed in "${PROCESSED_SECRETS[@]}"; do
        [[ "$processed" == "$key" ]] && { log_warning "Skipping duplicate: $key"; return; }
    done
    PROCESSED_SECRETS+=("$key")

    # Handle base64 encoded SSH keys
    if [[ "$key" == *"SSH_PRIVATE_KEY"* && "$key" == *"BASE64"* ]]; then
        log_info "Found base64 encoded SSH key: $key"

        local new_key="${key/_BASE64/}"
        local tmp
        tmp=$(create_temp_file "ssh_key")

        base64_decode "$value" > "$tmp"
        gh secret set "$new_key" --repo "$REPO" < "$tmp"
        safe_remove "$tmp"

        log_info "Set decoded SSH key as: $new_key"
        return
    fi

    # Set regular secret
    log_info "Setting secret: $key ($(mask_value "$value" 4))"
    gh_set_secret "$key" "$value" "$REPO"
}

# Process each env file
for env_file in "${ENV_FILES[@]}"; do
    file_exists "$env_file" || { log_warning "File not found: $env_file"; continue; }

    log_info "Processing: $env_file"
    parse_mapping_file "$env_file" process_secret >/dev/null
done

log_success "Secrets upload completed"

# Show SSH key configuration info
log_info "SSH Key Configuration:"
log_info "  1. If you provided SSH_PRIVATE_KEY_BASE64, it's been decoded and set as SSH_PRIVATE_KEY"
log_info "  2. Make sure the corresponding public key is added to your server's authorized_keys"
log_info "  3. Use the SSH_PRIVATE_KEY in your GitHub Actions workflows"
