#!/usr/bin/env bash

# Encrypts *.secret files to *.secret.age
# Decrypts *.age files to *.dec 
# Needs your private key at ~/.config/age/key.txt
# Usage: ./age-encrypt-decrypt.sh encrypt | decrypt [directory]

set -e

# Source utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/utils.sh
source "$SCRIPT_DIR/utils.sh"

# shellcheck disable=SC2119
PROJECT_ROOT="$(get_project_root)"
SECRETS_DIR="${2:-$PROJECT_ROOT/sensitive}"
KEY_PATH="${AGE_KEY_PATH:-$HOME/.config/age/key.txt}"

# Displays usage information and exits
# Input: none
# Output: help text to stdout, exits with code 1
# Usage: usage
function usage() {
  echo "Usage: $0 [encrypt|decrypt] [directory]"
  echo
  echo "Arguments:"
  echo "  directory      Directory containing secrets (default: $PROJECT_ROOT/sensitive)"
  echo
  echo "Environment variables:"
  echo "  AGE_KEY_PATH   Alternative path to the private key file (default: $KEY_PATH)"
  exit 1
}

# Checks if age encryption tool is available
# Input: none
# Output: exits if age is not installed
# Usage: check_dependencies
# shellcheck disable=SC2119
function check_dependencies() {
  require_command "age" "age" "Visit: https://github.com/FiloSottile/age"
}

# Verifies that the age private key file exists
# Input: none (uses global KEY_PATH)
# Output: exits if key file is not found
# Usage: check_key_file
function check_key_file() {
  check_file_exists "$KEY_PATH" "Private key file not found: $KEY_PATH. Generate one with: age-keygen -o $KEY_PATH"
}

# Encrypts a single file using age
# Input: $1 - file to encrypt, $2 - public key string
# Output: encrypted .age file, increments counter, returns 0 on success
# Usage: encrypt_file "secret.txt" "age1abc123..."
function encrypt_file() {
  local file="$1"
  local pubkey="$2"
  local outfile="$file.age"
  
  log_info "Encrypting: $(basename "$file") → $(basename "$outfile")"
  
  if age -r "$pubkey" -o "$outfile" "$file"; then
    increment_counter "processed_count"
    return 0
  else
    exit_error "Failed to encrypt $file"
  fi
}

# Encrypts all *.secret files in the secrets directory
# Input: none (uses global SECRETS_DIR and KEY_PATH)
# Output: encrypted .age files, success message with count
# Usage: encrypt_all
function encrypt_all() {
  # shellcheck disable=SC2119
  check_dependencies
  check_key_file
  ensure_directory "$SECRETS_DIR"
  
  log_info "Looking for *.secret files in $SECRETS_DIR"
  
  if ! pubkey=$(grep -o 'age1[0-9a-z]*' "$KEY_PATH" 2>/dev/null); then
    exit_error "Could not extract public key from $KEY_PATH"
  fi
  
  init_counter "processed_count"
  
  for file in "$SECRETS_DIR"/*.secret; do
    [[ ! -f "$file" ]] && continue
    encrypt_file "$file" "$pubkey"
  done

  local count
  count=$(get_counter "processed_count")
  log_success "Encrypted $count files"
}

# Decrypts a single .age file
# Input: $1 - .age file to decrypt
# Output: decrypted .dec file, increments counter, returns 0 on success
# Usage: decrypt_file "secret.txt.age"
function decrypt_file() {
  local file="$1"
  local outfile="${file%.age}.dec"
  
  log_info "Decrypting: $(basename "$file") → $(basename "$outfile")"
  
  if age -d -i "$KEY_PATH" -o "$outfile" "$file"; then
    increment_counter "processed_count"
    return 0
  else
    exit_error "Failed to decrypt $file"
  fi
}

# Decrypts all *.age files in the secrets directory
# Input: none (uses global SECRETS_DIR and KEY_PATH)
# Output: decrypted .dec files, success message with count
# Usage: decrypt_all
function decrypt_all() {
  # shellcheck disable=SC2119
  check_dependencies
  check_key_file
  ensure_directory "$SECRETS_DIR"
  
  log_info "Decrypting *.age files in $SECRETS_DIR"
  
  init_counter "processed_count"
  
  for file in "$SECRETS_DIR"/*.age; do
    [[ ! -f "$file" ]] && continue
    decrypt_file "$file"
  done

  local count
  count=$(get_counter "processed_count")
  log_success "Decrypted $count files"
}

# Main
case "$1" in
  encrypt) encrypt_all ;;
  decrypt) decrypt_all ;;
  *) usage ;;
esac

log_success "Operation completed: $1"
