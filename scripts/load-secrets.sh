#!/usr/bin/env bash

# Decrypts age-encrypted secrets and exports as environment variables
# Sources from: $DOTFILES_DIR/sensitive/env-mapping.conf
# Requires: age, private key at ~/.config/age/key.txt
#
# Usage in .zshrc/.bashrc:
#   source "$DOTFILES_DIR/scripts/load-secrets.sh"
#
# Commands available after sourcing:
#   secrets_help    - Show all commands and usage
#   secrets_list    - Show mapped variables and their load status
#   secrets_refresh - Force re-decrypt and reload all secrets
#   secrets_show    - Show secret content (memory/disk, fallback .age)
#   secrets_add     - Add a new secret interactively
#   secrets_rotate  - Update an existing secret's value
#   secrets_check   - Validate mapping integrity
#   secrets_clean   - Remove plaintext .dec and .secret files
#   secrets_audit   - Show audit log of secret changes
#   secrets_sync    - Sync all secrets to repo (requires DOTFILES_REPO_DIR)

# Source utils.sh for helper functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
if [[ -f "$SCRIPT_DIR/utils.sh" ]]; then
    . "$SCRIPT_DIR/utils.sh"
else
    # Minimal fallback if utils.sh not found
    command_exists() { command -v "$1" >/dev/null 2>&1; }
    file_exists() { [[ -f "$1" ]]; }
    var_is_set() {
        var_name="$1"
        if [[ -n "$ZSH_VERSION" ]]; then
            # shellcheck disable=SC2296
            [[ -n "${(P)var_name:-}" ]]
        else
            [[ -n "${!var_name:-}" ]]
        fi
    }
    export_var() { export "$1=$2"; }
    unset_var() { unset "$1" 2>/dev/null || true; }
    log_warning() { printf '[WARNING] %s\n' "$1" >&2; }
fi

# Configuration
SECRETS_DIR="${DOTFILES_DIR:-$HOME/.dotfiles}/sensitive"
SECRETS_MAPPING_FILE="$SECRETS_DIR/env-mapping.conf"
SECRETS_KEY_PATH="${AGE_KEY_PATH:-$HOME/.config/age/key.txt}"
SECRETS_LOADED=0

# --- File-secret helpers ---

# Check if a mapping key is a file secret (prefixed with @)
_is_file_secret() {
    case "$1" in
        @*) return 0 ;;
        *) return 1 ;;
    esac
}

# Parse a file-secret value "filename>dest_path" into variables
# Sets _FS_FILENAME and _FS_DEST_PATH (avoids subshell for bash/zsh compat)
_parse_file_secret_value() {
    _FS_FILENAME="${1%%>*}"
    _FS_DEST_PATH="${1#*>}"
    # Expand ~ to $HOME
    case "$_FS_DEST_PATH" in
        "~"/*) _FS_DEST_PATH="$HOME/${_FS_DEST_PATH#"~/"}" ;;
        "~") _FS_DEST_PATH="$HOME" ;;
    esac
}

# Deploy a file secret: decrypt .age -> dest with caching
_secrets_load_file_entry() {
    raw_var="$1"
    raw_value="$2"

    # Strip @ prefix for env var name
    var_name="${raw_var#@}"

    _parse_file_secret_value "$raw_value"
    local filename="$_FS_FILENAME"
    local dest_path="$_FS_DEST_PATH"
    local encrypted_file="$SECRETS_DIR/${filename}.secret.age"

    file_exists "$encrypted_file" || return 1

    # Cache: skip if dest is newer than encrypted source
    if [[ -f "$dest_path" && "$dest_path" -nt "$encrypted_file" ]]; then
        export_var "$var_name" "$dest_path"
        SECRETS_LOADED=$((SECRETS_LOADED + 1))
        return 0
    fi

    # Ensure parent directory exists
    local dest_dir
    dest_dir="$(dirname "$dest_path")"
    if [[ ! -d "$dest_dir" ]] && ! mkdir -p "$dest_dir"; then
        log_warning "Failed to create directory $dest_dir for file secret ${var_name#@}"
        return 1
    fi

    # Decrypt to destination
    if age_decrypt "$encrypted_file" "$SECRETS_KEY_PATH" > "$dest_path" 2>/dev/null; then
        chmod 600 "$dest_path"
        export_var "$var_name" "$dest_path"
        SECRETS_LOADED=$((SECRETS_LOADED + 1))
    else
        # Remove potentially corrupt partial file
        rm -f "$dest_path"
        return 1
    fi
}

# Helper to get repo dir dynamically
_get_secrets_repo_dir() {
    echo "${DOTFILES_REPO_DIR:+${DOTFILES_REPO_DIR}/sensitive}"
}

# Sync file to repo if DOTFILES_REPO_DIR is configured
_secrets_sync_to_repo() {
    filename="$1"
    sync_mapping="${2:-false}"
    silent="${3:-false}"

    local repo_dir
    repo_dir="$(_get_secrets_repo_dir)"

    [[ -z "$repo_dir" ]] && return 0
    [[ "$repo_dir" == "$SECRETS_DIR" ]] && return 0
    [[ ! -d "$repo_dir" ]] && return 0

    # Sync file
    if [[ -f "$SECRETS_DIR/${filename}" ]]; then
        command cp "$SECRETS_DIR/${filename}" "$repo_dir/${filename}"
        [[ "$silent" != "true" ]] && echo "  Synced to repo: $repo_dir/${filename}"
    fi

    # Sync mapping file if requested
    if [[ "$sync_mapping" == "true" && -f "$SECRETS_MAPPING_FILE" ]]; then
        command cp "$SECRETS_MAPPING_FILE" "$repo_dir/env-mapping.conf"
    fi
}

# Handler for each mapping entry (used by parse_mapping_file)
_secrets_load_entry() {
    var_name="$1"
    filename="$2"

    # Dispatch file secrets to dedicated handler
    if _is_file_secret "$var_name"; then
        _secrets_load_file_entry "$var_name" "$filename"
        return $?
    fi

    encrypted_file="$SECRETS_DIR/${filename}.secret.age"

    file_exists "$encrypted_file" || return 1

    local value
    if ! value=$(age_decrypt "$encrypted_file" "$SECRETS_KEY_PATH" | tr -d '\n'); then
        log_warning "Failed to decrypt secret for $var_name ($encrypted_file)"
        return 1
    fi

    if [[ -n "$value" ]]; then
        export_var "$var_name" "$value"
        SECRETS_LOADED=$((SECRETS_LOADED + 1))
    else
        log_warning "Decrypted value is empty for $var_name ($encrypted_file)"
    fi
}

# Load all secrets from mapping file
secrets_load() {
    file_exists "$SECRETS_MAPPING_FILE" || return 1
    file_exists "$SECRETS_KEY_PATH" || return 1
    command_exists age || return 1

    SECRETS_LOADED=0

    # Use parse_mapping_file if available, otherwise fallback
    if type parse_mapping_file >/dev/null 2>&1; then
        parse_mapping_file "$SECRETS_MAPPING_FILE" _secrets_load_entry >/dev/null
    else
        # Fallback parsing
        local line key value
        while IFS= read -r line || [[ -n "$line" ]]; do
            [[ "$line" =~ ^[[:space:]]*# ]] && continue
            [[ -z "$line" || ! "$line" =~ = ]] && continue
            key="${line%%=*}"
            value="${line#*=}"
            key="${key// }"
            value="${value// }"
            _secrets_load_entry "$key" "$value"
        done < "$SECRETS_MAPPING_FILE"
    fi
}

# Force reload all secrets
secrets_refresh() {
    # Unset existing vars first
    if file_exists "$SECRETS_MAPPING_FILE"; then
        local line var_name raw_value
        while IFS= read -r line || [[ -n "$line" ]]; do
            [[ "$line" =~ ^[[:space:]]*# ]] && continue
            [[ -z "$line" || ! "$line" =~ = ]] && continue
            var_name="${line%%=*}"
            var_name="${var_name// }"
            if _is_file_secret "$var_name"; then
                # Strip @ for unset, remove deployed file to force re-decrypt
                raw_value="${line#*=}"
                raw_value="${raw_value// }"
                _parse_file_secret_value "$raw_value"
                rm -f "$_FS_DEST_PATH"
                unset_var "${var_name#@}"
            else
                unset_var "$var_name"
            fi
        done < "$SECRETS_MAPPING_FILE"
    fi

    secrets_load
    echo "Secrets refreshed: $SECRETS_LOADED variables loaded"
}

# List all mapped secrets and their load status
secrets_list() {
    if ! file_exists "$SECRETS_MAPPING_FILE"; then
        echo "Mapping file not found: $SECRETS_MAPPING_FILE"
        return 1
    fi

    echo "Secret mappings ($SECRETS_MAPPING_FILE):"
    echo ""

    # Pass 1: Environment variable secrets
    echo "Environment Variables:"
    local line var_name filename encrypted_file has_file_secrets=false
    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$line" || ! "$line" =~ = ]] && continue

        var_name="${line%%=*}"
        filename="${line#*=}"
        var_name="${var_name// }"
        filename="${filename// }"

        if _is_file_secret "$var_name"; then
            has_file_secrets=true
            continue
        fi

        encrypted_file="$SECRETS_DIR/${filename}.secret.age"

        if ! file_exists "$encrypted_file"; then
            echo "  x $var_name (missing: ${filename}.secret.age)"
        elif var_is_set "$var_name"; then
            echo "  * $var_name"
        else
            echo "  o $var_name (not loaded)"
        fi
    done < "$SECRETS_MAPPING_FILE"

    # Pass 2: File secrets
    if [[ "$has_file_secrets" == "true" ]]; then
        echo ""
        echo "File Secrets:"
        while IFS= read -r line || [[ -n "$line" ]]; do
            [[ "$line" =~ ^[[:space:]]*# ]] && continue
            [[ -z "$line" || ! "$line" =~ = ]] && continue

            var_name="${line%%=*}"
            filename="${line#*=}"
            var_name="${var_name// }"
            filename="${filename// }"

            _is_file_secret "$var_name" || continue

            local clean_var="${var_name#@}"
            _parse_file_secret_value "$filename"
            encrypted_file="$SECRETS_DIR/${_FS_FILENAME}.secret.age"

            if ! file_exists "$encrypted_file"; then
                echo "  x $clean_var -> $_FS_DEST_PATH (missing: ${_FS_FILENAME}.secret.age)"
            elif [[ -f "$_FS_DEST_PATH" ]]; then
                echo "  * $clean_var -> $_FS_DEST_PATH (deployed)"
            else
                echo "  o $clean_var -> $_FS_DEST_PATH (not deployed)"
            fi
        done < "$SECRETS_MAPPING_FILE"
    fi
}

# Show secret content from memory/disk (default) or from .age file (--raw)
# Usage: secrets_show [--raw] VAR_NAME
secrets_show() {
    local raw_mode=false var_name=""

    # Parse args: support both "secrets_show --raw VAR" and "secrets_show VAR --raw"
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --raw) raw_mode=true ;;
            --help|-h)
                echo "Usage: secrets_show [--raw] VAR_NAME"
                echo "  Default: show value from memory (env var) or deployed file"
                echo "  --raw:   decrypt directly from .age file"
                return 0
                ;;
            --*)
                echo "Unknown option: $1" >&2
                echo "Usage: secrets_show [--raw] VAR_NAME" >&2
                return 1
                ;;
            *) var_name="$1" ;;
        esac
        shift
    done

    if [[ -z "$var_name" ]]; then
        echo "Usage: secrets_show [--raw] VAR_NAME" >&2
        return 1
    fi

    file_exists "$SECRETS_MAPPING_FILE" || { echo "Mapping file not found" >&2; return 1; }

    # Lookup: try exact match first, then with @ prefix for file secrets
    local mapping_value="" is_file=false
    mapping_value=$(grep "^${var_name}=" "$SECRETS_MAPPING_FILE" 2>/dev/null | head -1 | cut -d'=' -f2-)
    if [[ -z "$mapping_value" ]]; then
        mapping_value=$(grep "^@${var_name}=" "$SECRETS_MAPPING_FILE" 2>/dev/null | head -1 | cut -d'=' -f2-)
        [[ -n "$mapping_value" ]] && is_file=true
    fi

    if [[ -z "$mapping_value" ]]; then
        echo "No mapping found for: $var_name" >&2
        return 1
    fi

    if [[ "$raw_mode" == "true" ]]; then
        # --raw: decrypt directly from .age
        file_exists "$SECRETS_KEY_PATH" || { echo "Key file not found: $SECRETS_KEY_PATH" >&2; return 1; }
        command_exists age || { echo "age not installed" >&2; return 1; }

        local actual_filename="$mapping_value"
        if [[ "$is_file" == "true" ]]; then
            actual_filename="${mapping_value%%>*}"
        fi

        local encrypted_file="$SECRETS_DIR/${actual_filename}.secret.age"
        if ! file_exists "$encrypted_file"; then
            echo "Encrypted file not found: $encrypted_file" >&2
            return 1
        fi

        if [[ "$is_file" == "true" ]]; then
            age_decrypt "$encrypted_file" "$SECRETS_KEY_PATH"
        else
            age_decrypt "$encrypted_file" "$SECRETS_KEY_PATH" | tr -d '\n'
        fi
    else
        # Default: read from memory/disk (fast), fallback to .age decrypt
        if [[ "$is_file" == "true" ]]; then
            _parse_file_secret_value "$mapping_value"
            if [[ -f "$_FS_DEST_PATH" ]]; then
                cat "$_FS_DEST_PATH"
                return 0
            fi
            # Fallback: try .age decrypt
            local fs_filename="${mapping_value%%>*}"
            local encrypted_file="$SECRETS_DIR/${fs_filename}.secret.age"
            if file_exists "$SECRETS_KEY_PATH" && command_exists age && file_exists "$encrypted_file"; then
                age_decrypt "$encrypted_file" "$SECRETS_KEY_PATH" && return 0
            fi
            echo "File not deployed: $_FS_DEST_PATH (run secrets_refresh)" >&2
            return 1
        else
            if var_is_set "$var_name"; then
                if [[ -n "$ZSH_VERSION" ]]; then
                    # shellcheck disable=SC2296
                    printf '%s' "${(P)var_name}"
                else
                    printf '%s' "${!var_name}"
                fi
                return 0
            fi
            # Fallback: try .age decrypt
            local encrypted_file="$SECRETS_DIR/${mapping_value}.secret.age"
            if file_exists "$SECRETS_KEY_PATH" && command_exists age && file_exists "$encrypted_file"; then
                local decrypted
                decrypted=$(age_decrypt "$encrypted_file" "$SECRETS_KEY_PATH" | tr -d '\n')
                if [[ -n "$decrypted" ]]; then
                    printf '%s' "$decrypted"
                    return 0
                fi
            fi
            echo "Variable not loaded: $var_name (run secrets_refresh)" >&2
            return 1
        fi
    fi
}

# Add a new secret interactively
# Usage: secrets_add VAR_NAME filename
secrets_add() {
    var_name="$1"
    filename="$2"

    # Validate inputs
    if [[ -z "$var_name" || -z "$filename" ]]; then
        echo "Usage: secrets_add VAR_NAME filename"
        echo "Example: secrets_add MYSERVICE_TOKEN myservice.api"
        return 1
    fi

    # Check dependencies
    file_exists "$SECRETS_KEY_PATH" || { echo "Error: Key file not found at $SECRETS_KEY_PATH" >&2; return 1; }
    command_exists age || { echo "Error: age not installed" >&2; return 1; }

    # Check if mapping already exists
    if grep -q "^${var_name}=" "$SECRETS_MAPPING_FILE" 2>/dev/null; then
        echo "Error: Mapping already exists for $var_name"
        echo "Use 'secrets_rotate $var_name' to update the value"
        return 1
    fi

    encrypted_file="$SECRETS_DIR/${filename}.secret.age"

    # Check if files already exist
    if file_exists "$encrypted_file"; then
        echo "Error: Encrypted file already exists: $encrypted_file"
        return 1
    fi

    # Prompt for value
    echo -n "Enter value for $var_name: "
    read -rs value
    echo ""

    if [[ -z "$value" ]]; then
        echo "Error: Value cannot be empty"
        return 1
    fi

    # Encrypt directly from stdin (no plaintext touches disk)
    if ! printf '%s' "$value" | age_encrypt "$encrypted_file" "$SECRETS_KEY_PATH"; then
        echo "Error: Encryption failed"
        return 1
    fi

    # Add mapping
    echo "${var_name}=${filename}" >> "$SECRETS_MAPPING_FILE"

    # Update audit log
    _secrets_audit_log "$var_name" "added"

    echo "Secret added successfully:"
    echo "  Variable: $var_name"
    echo "  File: ${filename}.secret.age"

    # Sync to repo if configured
    _secrets_sync_to_repo "${filename}.secret.age" true

    echo ""
    echo "Run 'secrets_refresh' or restart terminal to load"
}

# Validate mapping integrity - check for missing .age files or orphaned mappings
# Usage: secrets_check
secrets_check() {
    if ! file_exists "$SECRETS_MAPPING_FILE"; then
        echo "Mapping file not found: $SECRETS_MAPPING_FILE"
        return 1
    fi

    echo "Checking secrets integrity..."
    echo ""

    valid=0 missing=0 orphaned=0
    local line var_name filename encrypted_file

    # Check each mapping has corresponding .age file
    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$line" || ! "$line" =~ = ]] && continue

        var_name="${line%%=*}"
        filename="${line#*=}"
        var_name="${var_name// }"
        filename="${filename// }"

        # File secrets: extract filename portion before >
        if _is_file_secret "$var_name"; then
            local clean_var="${var_name#@}"
            local fs_filename="${filename%%>*}"
            encrypted_file="$SECRETS_DIR/${fs_filename}.secret.age"
            if file_exists "$encrypted_file"; then
                echo "  ✓ $clean_var [file]"
                valid=$((valid + 1))
            else
                echo "  ✗ $clean_var [file] (missing: ${fs_filename}.secret.age)"
                missing=$((missing + 1))
            fi
        else
            encrypted_file="$SECRETS_DIR/${filename}.secret.age"
            if file_exists "$encrypted_file"; then
                echo "  ✓ $var_name"
                valid=$((valid + 1))
            else
                echo "  ✗ $var_name (missing: ${filename}.secret.age)"
                missing=$((missing + 1))
            fi
        fi
    done < "$SECRETS_MAPPING_FILE"

    # Check for .age files without mappings
    echo ""
    echo "Checking for orphaned .age files..."
    local base_name
    for age_file in "$SECRETS_DIR"/*.secret.age; do
        file_exists "$age_file" || continue
        base_name=$(basename "$age_file" .secret.age)
        # Check both env-var format (=name$) and file-secret format (=name>)
        if ! grep -q "=${base_name}$" "$SECRETS_MAPPING_FILE" 2>/dev/null &&
           ! grep -q "=${base_name}>" "$SECRETS_MAPPING_FILE" 2>/dev/null; then
            echo "  ? ${base_name}.secret.age (no mapping)"
            orphaned=$((orphaned + 1))
        fi
    done

    echo ""
    echo "Summary:"
    echo "  Valid:    $valid"
    echo "  Missing:  $missing"
    echo "  Orphaned: $orphaned"

    [[ $missing -eq 0 && $orphaned -eq 0 ]] && return 0 || return 1
}

# Remove .dec and .secret plaintext files
# Usage: secrets_clean [--dry-run]
secrets_clean() {
    dry_run=false
    [[ "$1" == "--dry-run" ]] && dry_run=true

    dec_count=0 secret_count=0

    echo "Scanning for plaintext files in $SECRETS_DIR..."
    echo ""

    # Find .dec files
    # Use a subshell so zsh NULL_GLOB doesn't leak into the parent shell
    while IFS= read -r file; do
        if $dry_run; then
            echo "  Would remove: $(basename "$file")"
        else
            rm -f "$file"
            echo "  Removed: $(basename "$file")"
        fi
        dec_count=$((dec_count + 1))
    done < <(
        [[ -n "${ZSH_VERSION:-}" ]] && setopt NULL_GLOB
        for f in "$SECRETS_DIR"/*.dec "$SECRETS_DIR"/*.secret.dec; do
            file_exists "$f" && printf '%s\n' "$f"
        done
    )

    # Find .secret files (plaintext, not .secret.age)
    while IFS= read -r file; do
        if $dry_run; then
            echo "  Would remove: $(basename "$file")"
        else
            rm -f "$file"
            echo "  Removed: $(basename "$file")"
        fi
        secret_count=$((secret_count + 1))
    done < <(
        [[ -n "${ZSH_VERSION:-}" ]] && setopt NULL_GLOB
        for f in "$SECRETS_DIR"/*.secret; do
            file_exists "$f" || continue
            [[ "$f" == *.secret.age ]] && continue
            printf '%s\n' "$f"
        done
    )

    echo ""
    if $dry_run; then
        echo "Dry run complete. Would remove: $dec_count .dec files, $secret_count .secret files"
        echo "Run 'secrets_clean' without --dry-run to actually delete"
    else
        echo "Cleanup complete: $dec_count .dec files, $secret_count .secret files removed"
    fi
}

# Rotate (update) an existing secret
# Usage: secrets_rotate VAR_NAME
secrets_rotate() {
    var_name="$1"

    if [[ -z "$var_name" ]]; then
        echo "Usage: secrets_rotate VAR_NAME"
        echo "Example: secrets_rotate GITHUB_PERSONAL_ACCESS_TOKEN"
        return 1
    fi

    # Check dependencies
    file_exists "$SECRETS_KEY_PATH" || { echo "Error: Key file not found" >&2; return 1; }
    command_exists age || { echo "Error: age not installed" >&2; return 1; }

    # Find the mapping
    local filename
    filename=$(grep "^${var_name}=" "$SECRETS_MAPPING_FILE" 2>/dev/null | cut -d'=' -f2)

    if [[ -z "$filename" ]]; then
        echo "Error: No mapping found for $var_name"
        echo "Use 'secrets_add $var_name <filename>' to create a new secret"
        return 1
    fi

    encrypted_file="$SECRETS_DIR/${filename}.secret.age"
    backup_file="${encrypted_file}.bak"

    # Backup existing encrypted file
    if file_exists "$encrypted_file"; then
        command cp "$encrypted_file" "$backup_file"
    fi

    # Prompt for new value
    echo "Rotating secret: $var_name"
    echo -n "Enter new value: "
    read -rs value
    echo ""

    if [[ -z "$value" ]]; then
        echo "Error: Value cannot be empty"
        rm -f "$backup_file"
        return 1
    fi

    # Encrypt directly from stdin (no plaintext touches disk)
    if ! printf '%s' "$value" | age_encrypt "$encrypted_file" "$SECRETS_KEY_PATH"; then
        # Restore backup
        [[ -f "$backup_file" ]] && mv "$backup_file" "$encrypted_file"
        echo "Error: Encryption failed, restored backup"
        return 1
    fi

    # Cleanup backup
    rm -f "$backup_file"

    # Update audit log
    _secrets_audit_log "$var_name" "rotated"

    echo "Secret rotated successfully: $var_name"

    # Sync to repo if configured
    _secrets_sync_to_repo "${filename}.secret.age" false

    echo "Run 'secrets_refresh' or restart terminal to reload"
}

# Sync all secrets to repo
# Usage: secrets_sync
secrets_sync() {
    local repo_dir
    repo_dir="$(_get_secrets_repo_dir)"

    if [[ -z "$repo_dir" ]]; then
        echo "Error: DOTFILES_REPO_DIR not set"
        echo "Set it in your shell config: export DOTFILES_REPO_DIR=\"\$HOME/Projects/dotfiles\""
        return 1
    fi

    if [[ "$repo_dir" == "$SECRETS_DIR" ]]; then
        echo "DOTFILES_REPO_DIR is same as SECRETS_DIR, nothing to sync"
        return 0
    fi

    if [[ ! -d "$repo_dir" ]]; then
        echo "Error: Repo directory not found: $repo_dir"
        return 1
    fi

    echo "Syncing secrets to repo: $repo_dir"
    echo ""

    local count=0

    # Sync all .age files
    for file in "$SECRETS_DIR"/*.secret.age; do
        [[ -f "$file" ]] || continue
        filename="${file##*/}"
        command cp "$file" "$repo_dir/$filename"
        echo "  $filename"
        count=$((count + 1))
    done

    # Sync mapping file
    if [[ -f "$SECRETS_MAPPING_FILE" ]]; then
        command cp "$SECRETS_MAPPING_FILE" "$repo_dir/env-mapping.conf"
        echo "  env-mapping.conf"
        count=$((count + 1))
    fi

    # Sync audit log
    if [[ -f "$SECRETS_AUDIT_FILE" ]]; then
        command cp "$SECRETS_AUDIT_FILE" "$repo_dir/.secrets-audit.log"
        echo "  .secrets-audit.log"
        count=$((count + 1))
    fi

    echo ""
    echo "Synced $count files to repo"
}

# Add a new file secret interactively
# Usage: secrets_add_file VAR_NAME filename dest_path
secrets_add_file() {
    var_name="$1"
    filename="$2"
    dest_path="$3"

    # Validate inputs
    if [[ -z "$var_name" || -z "$filename" || -z "$dest_path" ]]; then
        echo "Usage: secrets_add_file VAR_NAME filename dest_path"
        echo "Example: secrets_add_file KUBECONFIG kubelab.kubeconfig ~/.kube/kubelab.config"
        return 1
    fi

    # Check dependencies
    file_exists "$SECRETS_KEY_PATH" || { echo "Error: Key file not found at $SECRETS_KEY_PATH" >&2; return 1; }
    command_exists age || { echo "Error: age not installed" >&2; return 1; }

    # Check if mapping already exists
    if grep -q "^@${var_name}=" "$SECRETS_MAPPING_FILE" 2>/dev/null; then
        echo "Error: File secret mapping already exists for $var_name"
        return 1
    fi

    encrypted_file="$SECRETS_DIR/${filename}.secret.age"

    # Check if encrypted file already exists
    if file_exists "$encrypted_file"; then
        echo "Error: Encrypted file already exists: $encrypted_file"
        return 1
    fi

    # Prompt for source file path
    printf "Enter path to source file: "
    read -r source_path
    # Expand ~ manually
    case "$source_path" in
        "~"/*) source_path="$HOME/${source_path#"~/"}" ;;
        "~") source_path="$HOME" ;;
    esac

    if [[ ! -f "$source_path" ]]; then
        echo "Error: Source file not found: $source_path"
        return 1
    fi

    # Encrypt source file
    if ! age_encrypt "$encrypted_file" "$SECRETS_KEY_PATH" "$source_path"; then
        echo "Error: Encryption failed"
        return 1
    fi

    # Collapse $HOME back to ~ for mapping portability (literal ~ is intentional)
    local mapping_dest="$dest_path"
    # shellcheck disable=SC2088
    case "$mapping_dest" in
        "$HOME"/*) mapping_dest="~/${mapping_dest#"$HOME/"}" ;;
    esac

    # Add mapping with @ prefix
    echo "@${var_name}=${filename}>${mapping_dest}" >> "$SECRETS_MAPPING_FILE"

    # Update audit log
    _secrets_audit_log "$var_name" "added (file)"

    echo "File secret added successfully:"
    echo "  Variable:  $var_name"
    echo "  Encrypted: ${filename}.secret.age"
    echo "  Deploys to: $dest_path"

    # Sync to repo if configured
    _secrets_sync_to_repo "${filename}.secret.age" true

    echo ""
    echo "Run 'secrets_refresh' or restart terminal to deploy"
}

# Show help for all secrets commands
# Usage: secrets_help
secrets_help() {
    cat << 'EOF'
Secrets Management Commands
===========================

LOADING & VIEWING
  secrets_list          Show all mapped secrets and their load status
  secrets_show VAR      Show secret value (from memory/disk, fallback .age)
                        Use --raw to always decrypt from .age file
  secrets_refresh       Force reload all secrets from encrypted files

MANAGEMENT
  secrets_add VAR FILE  Add a new secret interactively
                        Example: secrets_add API_KEY myservice.api

  secrets_rotate VAR    Update an existing secret's value
                        Example: secrets_rotate GITHUB_TOKEN

  secrets_check         Validate mapping integrity (find missing/orphaned files)

  secrets_clean         Remove plaintext .dec and .secret files
                        Use --dry-run to preview without deleting

  secrets_sync          Sync all secrets to repo (requires DOTFILES_REPO_DIR)
                        Copies: *.secret.age, env-mapping.conf, .secrets-audit.log

AUDIT
  secrets_audit [VAR]   Show audit log (when secrets were added/rotated)
                        Without VAR shows all entries

FILES
  Mapping:    $DOTFILES_DIR/sensitive/env-mapping.conf
  Encrypted:  $DOTFILES_DIR/sensitive/*.secret.age
  Audit log:  $DOTFILES_DIR/sensitive/.secrets-audit.log

REPO SYNC
  Set DOTFILES_REPO_DIR to auto-sync secrets to your repo:
    export DOTFILES_REPO_DIR="$HOME/Projects/dotfiles"

  When configured:
    - secrets_add/secrets_rotate auto-sync .age files and mapping
    - Audit log is synced automatically on changes
    - Use secrets_sync for manual full sync

EXAMPLES
  # Add a new secret (auto-syncs to repo if configured)
  secrets_add STRIPE_KEY stripe.api

  # Sync everything to repo manually
  secrets_sync

  # Check for issues
  secrets_check

  # Rotate a compromised secret
  secrets_rotate STRIPE_KEY

  # Clean up plaintext files after encryption
  secrets_clean

FILE SECRETS
  Deploy encrypted files (kubeconfig, SSH keys, certificates) to specific paths.

  Mapping format:  @VAR_NAME=filename>dest_path
  Example:         @KUBECONFIG=kubelab.kubeconfig>~/.kube/kubelab.config

  secrets_add_file VAR FILE DEST
                        Add a new file secret interactively
                        Example: secrets_add_file KUBECONFIG kubelab.kubeconfig ~/.kube/kubelab.config

  File secrets are:
    - Decrypted to dest_path at shell startup (with caching via -nt)
    - Env var points to the deployed file path
    - Shown separately in secrets_list as "File Secrets:"
    - Skipped by github-secrets-manager (not suitable for GitHub Actions)

EOF
}

# Audit logging - track when secrets are added/modified
SECRETS_AUDIT_FILE="$SECRETS_DIR/.secrets-audit.log"

# Internal: Write to audit log
_secrets_audit_log() {
    var_name="$1"
    action="$2"
    local timestamp
    timestamp=$(date -Iseconds)
    echo "$timestamp|$var_name|$action" >> "$SECRETS_AUDIT_FILE"

    # Sync audit log to repo (silent)
    _secrets_sync_to_repo ".secrets-audit.log" false true
}

# Show audit log for secrets
# Usage: secrets_audit [VAR_NAME]
secrets_audit() {
    filter="$1"

    if ! file_exists "$SECRETS_AUDIT_FILE"; then
        echo "No audit log found (no secrets have been added/rotated yet)"
        return 0
    fi

    echo "Secrets Audit Log"
    echo "================="
    echo ""
    printf "%-25s %-30s %s\n" "TIMESTAMP" "VARIABLE" "ACTION"
    printf "%-25s %-30s %s\n" "---------" "--------" "------"

    while IFS='|' read -r timestamp var_name action || [[ -n "$timestamp" ]]; do
        [[ -z "$timestamp" ]] && continue
        # Filter if specified
        if [[ -z "$filter" || "$var_name" == "$filter" ]]; then
            printf "%-25s %-30s %s\n" "$timestamp" "$var_name" "$action"
        fi
    done < "$SECRETS_AUDIT_FILE"
}

# Auto-load secrets when this file is sourced (not executed directly)
if [[ "${BASH_SOURCE[0]}" != "${0}" ]] || [[ -n "$ZSH_VERSION" ]]; then
    secrets_load
fi
