#!/bin/bash

# Uploads secrets to GitHub repository secrets
# Usage: ./github-secrets-manager.sh [OPTIONS] [ENV_PATH]
#
# Options:
#   --from-mapping    Use env-mapping.conf as source (decrypts secrets)
#   --list            List available secrets from env-mapping.conf
#   --select VAR...   Only upload specific variables
#
# Without options, reads from .env files

set -euo pipefail

# Source utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
. "$SCRIPT_DIR/utils.sh"

# Configuration
DOTFILES_DIR="${DOTFILES_DIR:-$HOME/.dotfiles}"
SECRETS_DIR="$DOTFILES_DIR/sensitive"
SECRETS_MAPPING_FILE="$SECRETS_DIR/env-mapping.conf"
SECRETS_KEY_PATH="${AGE_KEY_PATH:-$HOME/.config/age/key.txt}"

# Parse arguments
USE_MAPPING=false
LIST_ONLY=false
SELECTED_VARS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --from-mapping) USE_MAPPING=true; shift ;;
        --list) LIST_ONLY=true; USE_MAPPING=true; shift ;;
        --select) shift; while [[ $# -gt 0 && ! "$1" =~ ^-- ]]; do SELECTED_VARS+=("$1"); shift; done ;;
        *) break ;;
    esac
done

# Check dependencies
require_command "gh" "GitHub CLI" "Visit: https://cli.github.com/"

if $USE_MAPPING; then
    require_command "age" "age encryption" "Visit: https://github.com/FiloSottile/age"
    file_exists "$SECRETS_KEY_PATH" || exit_error "Age key not found at $SECRETS_KEY_PATH"
    file_exists "$SECRETS_MAPPING_FILE" || exit_error "Mapping file not found at $SECRETS_MAPPING_FILE"
fi

# List secrets from mapping
if $LIST_ONLY; then
    log_info "Available secrets in env-mapping.conf:"
    echo ""
    while IFS='=' read -r var_name filename || [[ -n "$var_name" ]]; do
        [[ -z "$var_name" || "$var_name" =~ ^# ]] && continue
        [[ "$var_name" == @* ]] && continue  # File secrets not suitable for GitHub Actions
        [[ "$var_name" =~ ^GITHUB_ ]] && continue  # GitHub reserves this prefix
        encrypted_file="$SECRETS_DIR/${filename}.secret.age"
        if file_exists "$encrypted_file"; then
            echo "  $var_name"
        else
            echo "  $var_name (missing .age file)"
        fi
    done < "$SECRETS_MAPPING_FILE"
    exit 0
fi

gh_is_authenticated || exit_error "Not authenticated with GitHub CLI. Run 'gh auth login' first."

# Get repository
REPO=$(gh_get_repo)
[[ -z "$REPO" ]] && exit_error "Could not determine repository. Make sure you're in a Git repository directory."

log_success "Configuring secrets for repository: $REPO"

# Define source based on mode
if $USE_MAPPING; then
    log_info "Using env-mapping.conf as source"
else
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
fi

# Track processed secrets to avoid duplicates
PROCESSED_SECRETS=()

# Check if variable is selected (when --select is used)
is_selected() {
    var="$1"
    [[ ${#SELECTED_VARS[@]} -eq 0 ]] && return 0  # No filter = all selected
    for sel in "${SELECTED_VARS[@]}"; do
        [[ "$sel" == "$var" ]] && return 0
    done
    return 1
}

# Handler for each env entry
process_secret() {
    key="$1"
    value="$2"

    # Check if selected
    is_selected "$key" || return 0

    # Skip duplicates
    for processed in "${PROCESSED_SECRETS[@]}"; do
        [[ "$processed" == "$key" ]] && { log_warning "Skipping duplicate: $key"; return; }
    done
    PROCESSED_SECRETS+=("$key")

    # Handle base64 encoded SSH keys
    if [[ "$key" == *"SSH_PRIVATE_KEY"* && "$key" == *"BASE64"* ]]; then
        log_info "Found base64 encoded SSH key: $key"

        new_key="${key/_BASE64/}"
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

# Process based on mode
if $USE_MAPPING; then
    # Process from env-mapping.conf (decrypt and upload)
    log_info "Processing secrets from env-mapping.conf..."
    echo ""

    while IFS='=' read -r var_name filename || [[ -n "$var_name" ]]; do
        [[ -z "$var_name" || "$var_name" =~ ^# ]] && continue
        [[ "$var_name" == @* ]] && continue  # File secrets not suitable for GitHub Actions
        [[ "$var_name" =~ ^GITHUB_ ]] && { log_warning "Skipping $var_name (GITHUB_ prefix reserved)"; continue; }

        # Check if selected
        is_selected "$var_name" || continue

        encrypted_file="$SECRETS_DIR/${filename}.secret.age"

        if ! file_exists "$encrypted_file"; then
            log_warning "Missing file for $var_name: ${filename}.secret.age"
            continue
        fi

        # Decrypt and get value
        value=$(age_decrypt "$encrypted_file" "$SECRETS_KEY_PATH")

        if [[ -z "$value" ]]; then
            log_warning "Failed to decrypt: $var_name"
            continue
        fi

        process_secret "$var_name" "$value"
    done < "$SECRETS_MAPPING_FILE"
else
    # Process from .env files
    for env_file in "${ENV_FILES[@]}"; do
        file_exists "$env_file" || { log_warning "File not found: $env_file"; continue; }

        log_info "Processing: $env_file"
        parse_mapping_file "$env_file" process_secret >/dev/null
    done
fi

echo ""
log_success "Secrets upload completed (${#PROCESSED_SECRETS[@]} secrets)"

# Show usage hints
if $USE_MAPPING; then
    log_info "Uploaded from env-mapping.conf"
    log_info "Use --select VAR1 VAR2 to upload specific secrets"
else
    log_info "SSH Key Configuration:"
    log_info "  1. If you provided SSH_PRIVATE_KEY_BASE64, it's been decoded and set as SSH_PRIVATE_KEY"
    log_info "  2. Make sure the corresponding public key is added to your server's authorized_keys"
    log_info "  3. Use the SSH_PRIVATE_KEY in your GitHub Actions workflows"
fi
