#!/usr/bin/env bash

# Backs up secrets and standalone age script to USB
# Usage: ./backup-secrets-to-usb.sh /media/user/USB
# The USB should already contain key.txt in its root

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
. "$SCRIPT_DIR/utils.sh"

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

[[ -z "${1:-}" ]] && usage

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

# Copy the disaster-recovery escrow — the full Bitwarden export, age-encrypted.
#
# The loop above matches only the FLAT top level of sensitive/, while
# `dotf secrets backup` writes into sensitive/dr/. So the USB carried the
# per-secret age floor and never the full-vault escrow, leaving the escrow's
# only off-machine copy in the GitHub repo: lose that account and the key
# survives on this USB while the ciphertext it decrypts does not (#1000).
#
# That gap matters more than it used to, not less: per #971 `migrate` drops the
# `age:` pointer, so the per-secret floor no longer exists for the 28 migrated
# secrets and the escrow is the only thing standing behind them.
#
# An explicit path rather than a widened glob, so what lands on the USB stays a
# declared list — a `**` here would silently start shipping whatever else ever
# appears under sensitive/.
DR_ESCROW="$SECRETS_DIR/dr/bitwarden-export.age"
if file_exists "$DR_ESCROW"; then
    mkdir -p "$USB_SECRETS/dr"
    cp "$DR_ESCROW" "$USB_SECRETS/dr/"
    log_info "Copied: dr/$(basename "$DR_ESCROW")"
    increment_counter "count"
else
    # Loud, because a USB that looks complete and silently lacks the escrow is
    # the failure this block exists to end.
    log_warning "No DR escrow at $DR_ESCROW — run 'dotf secrets backup' first; this USB will NOT carry the full-vault export"
fi

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
