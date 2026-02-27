#!/usr/bin/env bash

# Standalone age encrypt/decrypt script for USB backup
# No external dependencies except 'age' binary
# Usage: ./age-standalone.sh encrypt|decrypt [directory]
#
# Expects key.txt in the same directory as this script
# or set AGE_KEY_PATH environment variable

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
KEY_PATH="${AGE_KEY_PATH:-$SCRIPT_DIR/key.txt}"
SECRETS_DIR="${2:-$SCRIPT_DIR/secrets}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()    { printf '%b[INFO]%b %s\n' "$BLUE" "$NC" "$1"; }
log_success() { printf '%b[OK]%b %s\n' "$GREEN" "$NC" "$1"; }
log_error()   { printf '%b[ERROR]%b %s\n' "$RED" "$NC" "$1"; }

die() {
    log_error "$1"
    exit 1
}

usage() {
    echo "Usage: $0 [encrypt|decrypt] [directory]"
    echo
    echo "Arguments:"
    echo "  encrypt    Encrypt all *.secret files to *.secret.age"
    echo "  decrypt    Decrypt all *.age files to *.dec"
    echo "  directory  Directory containing secrets (default: $SCRIPT_DIR/secrets)"
    echo
    echo "Environment:"
    echo "  AGE_KEY_PATH  Path to private key (default: $SCRIPT_DIR/key.txt)"
    echo
    echo "Examples:"
    echo "  $0 decrypt                    # Decrypt secrets/ using key.txt"
    echo "  $0 decrypt /path/to/secrets   # Decrypt custom directory"
    echo "  AGE_KEY_PATH=~/key.txt $0 decrypt"
    exit 1
}

check_requirements() {
    command -v age >/dev/null 2>&1 || die "age not installed. Install: sudo apt install age"
    [[ -f "$KEY_PATH" ]] || die "Key not found: $KEY_PATH"
    [[ -d "$SECRETS_DIR" ]] || die "Directory not found: $SECRETS_DIR"
}

get_pubkey() {
    grep -o 'age1[0-9a-z]*' "$KEY_PATH"
}

encrypt_all() {
    check_requirements

    local pubkey count=0
    pubkey=$(get_pubkey) || die "Could not extract public key from $KEY_PATH"

    log_info "Encrypting *.secret files in $SECRETS_DIR"

    for file in "$SECRETS_DIR"/*.secret; do
        [[ -f "$file" ]] || continue

        outfile="${file}.age"
        log_info "Encrypting: $(basename "$file")"

        rm -f "$outfile"
        age -r "$pubkey" -o "$outfile" "$file" || die "Failed to encrypt $file"
        count=$((count + 1))
    done

    log_success "Encrypted $count files"
}

decrypt_all() {
    check_requirements

    local count=0

    log_info "Decrypting *.age files in $SECRETS_DIR"

    for file in "$SECRETS_DIR"/*.age; do
        [[ -f "$file" ]] || continue

        outfile="${file%.age}.dec"
        log_info "Decrypting: $(basename "$file")"

        rm -f "$outfile"
        age -d -i "$KEY_PATH" -o "$outfile" "$file" || die "Failed to decrypt $file"
        count=$((count + 1))
    done

    log_success "Decrypted $count files"
}

# Main
ACTION="${1:-}"
case "$ACTION" in
    encrypt) encrypt_all ;;
    decrypt) decrypt_all ;;
    *) usage ;;
esac
