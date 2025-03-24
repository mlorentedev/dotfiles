#!/bin/bash
# setup-github-secrets.sh - Configure secrets in GitHub repository from multiple .env files
# Usage: ./setup-github-secrets.sh [ENV_PATH]
# If ENV_PATH is not provided, it will use default locations
set -e

# Source utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/utils.sh"

# Check dependencies
check_dependencies "gh"

# Define environment files to process based on input parameter
if [ -n "$1" ]; then
    # User provided a custom .env path
    ENV_BASE_PATH="$1"
    ENV_FILES=("$ENV_BASE_PATH")
    log_info "Using custom environment file: $ENV_BASE_PATH"
else
    # Use default paths
    ENV_FILES=(
        "$(dirname "$0")/.env"
        "$(dirname "$0")/.env.local"
        "$(dirname "$0")/.env.production"
        "$(dirname "$0")/.env.development"
        "$(dirname "$0")/.env.staging"
    )
    log_info "Using default environment file paths"
fi

# Function to process env files and set GitHub secrets
process_env_files() {
    local processed_secrets=()

    for env_file in "${ENV_FILES[@]}"; do
        if [ ! -f "$env_file" ]; then
            log_warning "Environment file not found: $env_file"
            continue
        fi

        log_info "Processing environment file: $env_file"

        # Read and process each line in the env file
        while IFS= read -r line || [ -n "$line" ]; do
            # Skip comments, empty lines, and lines without '='
            if [[ "$line" =~ ^#.*$ || -z "$line" || ! "$line" =~ = ]]; then
                continue
            fi

            # Split key and value
            IFS='=' read -r key value <<< "$line"

            # Trim leading/trailing whitespace
            key=$(echo "$key" | xargs)
            value=$(echo "$value" | xargs)

            # Remove surrounding quotes from value if present
            value="${value%\"}"
            value="${value#\"}"

            # Skip already processed secrets to avoid duplicates
            if [[ " ${processed_secrets[*]} " =~ " $key " ]]; then
                log_warning "Skipping duplicate secret: $key"
                continue
            fi

            # Add to processed secrets to prevent duplicates
            processed_secrets+=("$key")

            # Handle special cases for SSH keys
            if [[ "$key" == *"SSH_PRIVATE_KEY"* && "$key" == *"BASE64"* ]]; then
                # Found a base64 encoded SSH key - decode it
                log_info "Found base64 encoded SSH key: $key"
                
                # Create a non-base64 key secret name
                new_key=${key/_BASE64/}
                
                # Decode the key and store in a temp file
                echo "$value" | base64 --decode > /tmp/ssh_key_decoded
                chmod 600 /tmp/ssh_key_decoded
                
                # Set the decoded key as a GitHub secret
                log_info "Setting decoded SSH key as: $new_key"
                gh secret set "$new_key" --repo "$REPO" < /tmp/ssh_key_decoded
                
                # Clean up
                rm -f /tmp/ssh_key_decoded
                
                # Continue processing other secrets
                continue
            fi            

            # Mask potentially sensitive values
            masked_value=$(echo "$value" | sed 's/./*/g')
            log_info "Setting secret: $key (value masked: $masked_value)"

            # Set GitHub secret
            echo "$value" | gh secret set "$key" --repo "$REPO"

        done < "$env_file"
    done
}

# Check GitHub authentication
if ! gh auth status &> /dev/null; then
    exit_error "You are not authenticated with GitHub CLI. Please run 'gh auth login' first."
fi

# Get repository name
REPO=$(gh repo view --json nameWithOwner -q ".nameWithOwner" 2>/dev/null)
if [ -z "$REPO" ]; then
    exit_error "Could not determine repository. Make sure you're in a Git repository directory."
fi

log_success "Configuring secrets for repository: $REPO"

# Process environment files and set GitHub secrets
process_env_files

# Show information about the SSH key configuration
log_info "SSH Key Configuration:"
log_info "1. If you provided SSH_PRIVATE_KEY_BASE64, it's been decoded and set as SSH_PRIVATE_KEY"
log_info "2. Make sure the corresponding public key is added to your server's authorized_keys"
log_info "3. Use the SSH_PRIVATE_KEY in your GitHub Actions workflows"