#!/usr/bin/env bash

# Backs up secrets and standalone age script to USB
# Usage: ./backup-secrets-to-usb.sh /media/user/USB
# The USB should already contain key.txt in its root

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/utils.sh"

PROJECT_ROOT="$(get_project_root "$SCRIPT_DIR")"
SECRETS_DIR="$PROJECT_ROOT/sensitive"

usage() {
    echo "Usage: $0 <usb-path>"
    echo
    echo "Example:"
    echo "  $0 /media/manu/SECRETS_USB"
    echo "  $0 /run/media/manu/SECRETS_USB"
    echo
    echo "Prerequisites:"
    echo "  - USB must be mounted"
    echo "  - key.txt must already exist in USB root"
    exit 1
}

[[ -z "$1" ]] && usage

USB_PATH="$1"
USB_SECRETS="$USB_PATH/secrets"
USB_KEY="$USB_PATH/key.txt"

# Validations
dir_exists "$USB_PATH" || exit_error "USB path not found: $USB_PATH"
file_exists "$USB_KEY" || exit_error "key.txt not found in USB. Copy it manually first for security."
dir_exists "$SECRETS_DIR" || exit_error "Secrets directory not found: $SECRETS_DIR"

log_info "Backing up secrets to USB: $USB_PATH"

# Create secrets directory on USB
mkdir -p "$USB_SECRETS"

# Copy all secrets (both .secret and .secret.age)
init_counter "count"

for file in "$SECRETS_DIR"/*.secret "$SECRETS_DIR"/*.secret.age; do
    file_exists "$file" || continue
    cp "$file" "$USB_SECRETS/"
    log_info "Copied: $(basename "$file")"
    increment_counter "count"
done

# Copy standalone age script
cp "$SCRIPT_DIR/age-standalone.sh" "$USB_PATH/"
chmod +x "$USB_PATH/age-standalone.sh"
log_info "Copied: age-standalone.sh"

log_success "Backup complete: $(get_counter count) secret files copied"
echo
echo "USB contents:"
echo "  $USB_PATH/"
echo "  ├── key.txt              # Your private key (already present)"
echo "  ├── age-standalone.sh    # Encrypt/decrypt script"
echo "  └── secrets/             # All your secrets"
echo
echo "To decrypt on any Linux system:"
echo "  1. Install age: sudo apt install age"
echo "  2. Run: ./age-standalone.sh decrypt secrets"
