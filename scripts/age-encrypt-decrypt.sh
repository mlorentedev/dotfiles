#!/usr/bin/env bash

# Encrypts *.secret files to *.secret.age
# Decrypts *.age files to *.dec
# Needs your private key at ~/.config/age/key.txt
# Usage: ./age-encrypt-decrypt.sh encrypt | decrypt [directory]

set -e

# Source utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/utils.sh"

PROJECT_ROOT="$(get_project_root "$SCRIPT_DIR")"
SECRETS_DIR="${2:-$PROJECT_ROOT/sensitive}"

# Displays usage information and exits
usage() {
    echo "Usage: $0 [encrypt|decrypt] [directory]"
    echo
    echo "Arguments:"
    echo "  encrypt      Encrypt all *.secret files to *.secret.age"
    echo "  decrypt      Decrypt all *.age files to *.dec"
    echo "  directory    Directory containing secrets (default: $PROJECT_ROOT/sensitive)"
    echo
    echo "Environment variables:"
    echo "  AGE_KEY_PATH   Path to private key file (default: ~/.config/age/key.txt)"
    exit 1
}

# Encrypts all *.secret files in the secrets directory
encrypt_all() {
    require_command "age" "age" "Visit: https://github.com/FiloSottile/age"
    check_file_exists "$AGE_DEFAULT_KEY" "Private key not found: $AGE_DEFAULT_KEY. Generate: age-keygen -o $AGE_DEFAULT_KEY"
    ensure_directory "$SECRETS_DIR"

    pubkey=$(age_get_pubkey)
    [[ -z "$pubkey" ]] && exit_error "Could not extract public key from $AGE_DEFAULT_KEY"

    log_info "Encrypting *.secret files in $SECRETS_DIR"

    init_counter "processed_count"

    for file in "$SECRETS_DIR"/*.secret; do
        file_exists "$file" || continue

        outfile="${file}.age"
        log_info "Encrypting: $(basename "$file") -> $(basename "$outfile")"

        if age_encrypt "$outfile" "$AGE_DEFAULT_KEY" "$file"; then
            increment_counter "processed_count"
        else
            exit_error "Failed to encrypt $file"
        fi
    done

    log_success "Encrypted $(get_counter processed_count) files"
}

# Decrypts all *.age files in the secrets directory
decrypt_all() {
    require_command "age" "age" "Visit: https://github.com/FiloSottile/age"
    check_file_exists "$AGE_DEFAULT_KEY" "Private key not found: $AGE_DEFAULT_KEY"
    ensure_directory "$SECRETS_DIR"

    log_info "Decrypting *.age files in $SECRETS_DIR"

    init_counter "processed_count"

    for file in "$SECRETS_DIR"/*.age; do
        file_exists "$file" || continue

        outfile="${file%.age}.dec"
        log_info "Decrypting: $(basename "$file") -> $(basename "$outfile")"

        if age_decrypt "$file" "$AGE_DEFAULT_KEY" > "$outfile"; then
            increment_counter "processed_count"
        else
            exit_error "Failed to decrypt $file"
        fi
    done

    log_success "Decrypted $(get_counter processed_count) files"
}

# Main
case "$1" in
    encrypt) encrypt_all ;;
    decrypt) decrypt_all ;;
    *) usage ;;
esac

log_success "Operation completed: $1"
