#!/bin/bash
# Common utilities for scripts
# Gets sourced by other scripts for shared functions

# Colors for output
export RED='\033[0;31m'
export GREEN='\033[0;32m'
export YELLOW='\033[1;33m'
export BLUE='\033[0;34m'
export PURPLE='\033[0;35m'
export CYAN='\033[0;36m'
export NC='\033[0m' 

# Logging functions

# Prints an info message with blue color
# Input: $1 - message text
# Output: colored info message to stdout
# Usage: log_info "Starting process"
log_info() {
    printf '%b[INFO]%b %s\n' "$BLUE" "$NC" "$1"
}

# Prints a success message with green color
# Input: $1 - message text
# Output: colored success message to stdout
# Usage: log_success "Task completed"
log_success() {
    printf '%b[SUCCESS]%b %s\n' "$GREEN" "$NC" "$1"
}

# Prints a warning message with yellow color
# Input: $1 - message text
# Output: colored warning message to stdout
# Usage: log_warning "File not found"
log_warning() {
    printf '%b[WARNING]%b %s\n' "$YELLOW" "$NC" "$1"
}

# Prints an error message with red color
# Input: $1 - message text
# Output: colored error message to stdout
# Usage: log_error "Operation failed"
log_error() {
    printf '%b[ERROR]%b %s\n' "$RED" "$NC" "$1"
}

# Logs error message and exits with specified code
# Input: $1 - error message, $2 - exit code (optional, defaults to 1)
# Output: error message to stdout, then exits
# Usage: exit_error "Failed to connect" 2
exit_error() {
    message=$1
    code=${2:-1} # Default exit code is 1

    log_error "$message"
    exit $code
}

# version_gte: is an installed version at least the pinned minimum?
# versions.conf pins are MINIMUMS, not exact pins (REFACTOR-013): an install
# gate must upgrade when installed < pin, and leave a NEWER install untouched
# (an exact-match `!=` reconcile would downgrade it). Pure semver compare via
# `sort -V`; no external deps.
# Input:  $1 - installed version (e.g. "1.16.2"), $2 - pinned minimum
# Returns: 0 if installed >= minimum (satisfied), 1 if installed < minimum
#          (needs install/upgrade). Empty minimum -> 0 (no pin = anything OK);
#          empty installed -> 1 (nothing there = needs install).
# Usage:   version_gte "$installed" "$PIN" || install_or_upgrade
version_gte() {
    installed="$1"
    minimum="$2"
    [ -z "$minimum" ] && return 0
    [ -z "$installed" ] && return 1
    [ "$installed" = "$minimum" ] && return 0
    # installed >= minimum iff the minimum sorts as the lower of the two.
    [ "$(printf '%s\n%s\n' "$installed" "$minimum" | sort -V | head -n1)" = "$minimum" ]
}

# Silent check functions (for conditionals and shell startup)

# Checks if a command exists in PATH (silent, no output)
# Input: $1 - command name
# Output: none, returns 0 if found, 1 if missing
# Usage: command_exists "age" && echo "age is installed"
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Checks if a file exists (silent, no output)
# Input: $1 - file path
# Output: none, returns 0 if exists, 1 if missing
# Usage: file_exists "$config" && source "$config"
file_exists() {
    [[ -f "$1" ]]
}

# Checks if a directory exists (silent, no output)
# Input: $1 - directory path
# Output: none, returns 0 if exists, 1 if missing
# Usage: dir_exists "$path" && cd "$path"
dir_exists() {
    [[ -d "$1" ]]
}

# Checks if a symlink exists and points to a valid target
# Input: $1 - symlink path
# Output: none, returns 0 if valid symlink, 1 otherwise
# Usage: symlink_valid "$link" && echo "ok"
symlink_valid() {
    [[ -L "$1" && -e "$1" ]]
}

# Checks if an environment variable is set and non-empty
# Input: $1 - variable name (not value)
# Output: none, returns 0 if set and non-empty, 1 otherwise
# Usage: var_is_set "GITHUB_TOKEN" && echo "configured"
var_is_set() {
    var_name="$1"
    if [[ -n "${ZSH_VERSION:-}" ]]; then
        # shellcheck disable=SC2296
        [[ -n "${(P)var_name:-}" ]]
    else
        [[ -n "${!var_name:-}" ]]
    fi
}

# Environment functions

# Loads environment variables from .env file
# Input: $1 - path to .env file
# Output: environment variables loaded into shell
# Usage: load_env_file ".env"
load_env_file() {
    env_file="$1"
    
    # Check if file exists
    if [ ! -f "$env_file" ]; then
        log_warning "Environment file not found: $env_file"
        return 1
    fi

    # Read file line by line, handling multiline and quoted values
    while IFS= read -r line || [ -n "$line" ]; do
        # Skip comments and empty lines
        [[ "$line" =~ ^[[:space:]]*#.* ]] && continue
        [[ -z "$line" ]] && continue

        # Extract key and value
        key=$(echo "$line" | cut -d '=' -f1)
        value=$(echo "$line" | cut -d '=' -f2-)

        # Remove leading/trailing whitespace
        key=$(echo "$key" | xargs)
        value=$(echo "$value" | xargs)

        # Remove surrounding quotes if present
        value="${value%\"}"
        value="${value#\"}"

        # Export the variable, preserving multiline content
        export "$key=$value"
    done < "$env_file"
}

# Prints environment variables from .env file for debugging
# Input: $1 - path to .env file (optional, defaults to .env)
# Output: formatted list of environment variables to stdout
# Usage: debug_print_env ".env.local"
debug_print_env() {
    env_file="${1:-.env}"
    echo "=== Loaded Environment Variables ==="
    
    # Print all variables from the env file
    while IFS='=' read -r key value; do
        # Skip comments and empty lines
        [[ "$key" =~ ^[[:space:]]*#.* ]] && continue
        [[ -z "$key" ]] && continue
        
        # Print the key and its value
        echo "${key}=${value}"
    done < "$env_file"
}

# Config parsing functions

# Parses a KEY=VALUE mapping file, calling a handler for each entry
# Input: $1 - file path, $2 - callback function name (receives key, value)
# Output: echoes count of entries processed, returns 1 if file not found
# Usage: parse_mapping_file "config.conf" my_handler
parse_mapping_file() {
    file="$1"
    callback="$2"
    count=0

    [[ ! -f "$file" ]] && return 1

    local line key value
    while IFS= read -r line || [[ -n "$line" ]]; do
        # Skip comments and empty lines
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$line" || ! "$line" =~ = ]] && continue

        # Split key=value
        key="${line%%=*}"
        value="${line#*=}"

        # Trim whitespace
        key="${key#"${key%%[![:space:]]*}"}"
        key="${key%"${key##*[![:space:]]}"}"
        value="${value#"${value%%[![:space:]]*}"}"
        value="${value%"${value##*[![:space:]]}"}"

        # Remove surrounding quotes from value
        value="${value#\"}"
        value="${value%\"}"
        value="${value#\'}"
        value="${value%\'}"

        # Call handler if provided
        [[ -n "$callback" ]] && "$callback" "$key" "$value"
        count=$((count + 1))
    done < "$file"

    echo "$count"
}

# Age encryption functions

# Default path for age private key
AGE_DEFAULT_KEY="${AGE_KEY_PATH:-$HOME/.config/age/key.txt}"

# Decrypts an age-encrypted file to stdout
# Input: $1 - encrypted file path, $2 - key file path (optional)
# Output: decrypted content to stdout, returns 0 on success, 1 on failure
# Usage: value=$(age_decrypt "secret.age")
age_decrypt() {
    file="$1"
    key_file="${2:-$AGE_DEFAULT_KEY}"

    file_exists "$file" || return 1
    file_exists "$key_file" || return 1
    command_exists age || return 1

    age -d -i "$key_file" "$file" 2>/dev/null
}

# Encrypts stdin or file to an age-encrypted output file
# Input: $1 - output file, $2 - key file (optional), $3 - input file (optional, uses stdin if not provided)
# Output: encrypted file, returns 0 on success, 1 on failure
# Usage: echo "secret" | age_encrypt "output.age"
# Usage: age_encrypt "output.age" "" "input.txt"
age_encrypt() {
    output="$1"
    key_file="${2:-$AGE_DEFAULT_KEY}"
    input="$3"

    file_exists "$key_file" || return 1
    command_exists age || return 1

    local pubkey
    pubkey=$(grep -o 'age1[0-9a-z]*' "$key_file" 2>/dev/null) || return 1

    local tmp_output="${output}.tmp.$$"
    if [[ -n "$input" ]]; then
        age -r "$pubkey" -o "$tmp_output" "$input" 2>/dev/null
    else
        age -r "$pubkey" -o "$tmp_output" 2>/dev/null
    fi
    local rc=$?
    if [ $rc -eq 0 ]; then
        mv "$tmp_output" "$output"
    else
        rm -f "$tmp_output"
        return $rc
    fi
}

# Extracts public key from age private key file
# Input: $1 - key file path (optional, defaults to AGE_DEFAULT_KEY)
# Output: public key string to stdout
# Usage: pubkey=$(age_get_pubkey)
age_get_pubkey() {
    key_file="${1:-$AGE_DEFAULT_KEY}"
    grep -o 'age1[0-9a-z]*' "$key_file" 2>/dev/null
}

# GitHub CLI functions

# Checks if GitHub CLI is authenticated
# Input: none
# Output: none, returns 0 if authenticated, 1 otherwise
# Usage: gh_is_authenticated || exit_error "Run 'gh auth login' first"
gh_is_authenticated() {
    command_exists gh && gh auth status >/dev/null 2>&1
}

# Gets the current repository name in owner/repo format
# Input: none
# Output: repository name to stdout (e.g., "owner/repo")
# Usage: repo=$(gh_get_repo)
gh_get_repo() {
    gh repo view --json nameWithOwner -q ".nameWithOwner" 2>/dev/null
}

# Sets a GitHub repository secret
# Input: $1 - secret name, $2 - secret value, $3 - repository (optional, defaults to current)
# Output: none, returns 0 on success, 1 on failure
# Usage: gh_set_secret "API_TOKEN" "$token"
gh_set_secret() {
    name="$1"
    value="$2"
    repo="${3:-$(gh_get_repo)}"

    [[ -z "$repo" ]] && return 1
    echo "$value" | gh secret set "$name" --repo "$repo"
}

# Server connectivity functions

# Tests SSH connection to a server
# Input: $1 - server hostname or IP
# Output: success/error messages, returns 0 if connected, 1 if failed
# Usage: check_server_connectivity "user@server.com"
check_server_connectivity() {
    server=$1
    
    log_info "Checking SSH connectivity to $server..."
    
    if ! ssh -q -o BatchMode=yes -o ConnectTimeout=5 "$server" exit 2>/dev/null; then
        exit_error "Cannot connect to server $server. Verify that the server is correctly configured and SSH key is in place."
    fi
    
    log_success "Server connection successful."
}

# Dependency check functions

# Checks if a command exists in PATH
# Input: $1 - command name, $2 - package name (optional, defaults to command name)
# Output: error message if not found, returns 0 if found, 1 if missing
# Usage: check_command "git" "git-core"
check_command() {
    cmd=$1
    package=${2:-$cmd}
    
    if ! command -v "$cmd" >/dev/null 2>&1; then
        log_error "$cmd not found. Please install $package."
        return 1
    fi

    return 0
}

# check_dependencies was removed by OPS-043 (#1337) along with its only call
# site. `dotf doctor` reports every tool it named, at an equal-or-stronger
# severity -- it only ever log_warning'd, so it never failed anything.
# TestSetupShellParity holds the fourteen-row proof.

# User confirmation functions

# Prompts user for yes/no confirmation
# Input: $1 - confirmation message (optional, defaults to standard message)
# Output: confirmation prompt, returns 0 if confirmed, 1 if cancelled
# Usage: confirm_action "Delete all files?"
confirm_action() {
    message=${1:-"Are you sure you want to continue?"}
    
    printf '%b%s (y/n)%b\n' "$YELLOW" "$message" "$NC"
    read -r confirm
    
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        log_warning "Operation cancelled."
        return 1
    fi
    
    return 0
}

# File and backup functions

# Creates a timestamped backup of a file
# Input: $1 - path to file to backup
# Output: success/warning messages, backup file created
# Usage: backup_file "/path/to/config.txt"
backup_file() {
    file=$1
    backup="${file}.$(date +%Y%m%d%H%M%S).bak"
    
    if [ -f "$file" ]; then
        log_info "Creating backup of $file..."
        cp "$file" "$backup"
        log_success "Backup created: $backup"
    else
        log_warning "File $file does not exist, no backup created."
    fi
}

# Generates a timestamp string for file naming
# Input: none
# Output: timestamp in YYYYMMDDHHMMSS format to stdout
# Usage: timestamp=$(get_timestamp)
get_timestamp() {
    echo "$(date +%Y%m%d%H%M%S)"
}

# Directory and path functions

# Gets the absolute path to the directory containing the calling script
# Input: none (uses BASH_SOURCE)
# Output: absolute directory path to stdout
# Usage: script_dir=$(get_script_dir)
get_script_dir() {
    local calling_script
    if [[ -n "${ZSH_VERSION:-}" ]]; then
        # shellcheck disable=SC2296,SC2154
        calling_script="${funcfiletrace[1]%:*}"
    else
        calling_script="${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}"
    fi
    (cd "$(dirname "$calling_script")" && pwd)
}

# Gets the project root directory (parent of scripts directory)
# Input: $1 - script directory path (optional, auto-detected if not provided)
# Output: absolute path to project root to stdout
# Usage: project_root=$(get_project_root)
get_project_root() {
    script_dir="${1:-$(get_script_dir)}"
    echo "$(dirname "$script_dir")"
}

# Creates directory if it doesn't exist
# Input: $1 - directory path to create
# Output: info/success messages, exits on failure
# Usage: ensure_directory "/path/to/new/dir"
ensure_directory() {
    dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        log_info "Creating directory: $dir"
        if mkdir -p "$dir"; then
            log_success "Directory created: $dir"
        else
            exit_error "Failed to create directory: $dir"
        fi
    fi
}

# Deploys a file from repo source to home target unconditionally.
# Replaces `ln -sf` for config-deploy paths (SDD-007 IaC strategy).
# Input: $1 - source path (must exist), $2 - target path
# Behavior:
#   - declarative overwrite: every run guarantees DST matches SRC
#   - removes any existing symlink at target (drift recovery from old strategy)
#   - writes via tempfile + mv (atomic; no partial state)
# Output: returns 0 on success, 1 on error
# Usage: deploy_file "$DOTFILES_DIR/.zshrc" "$HOME/.zshrc"
deploy_file() {
    src="$1"
    dst="$2"

    if [[ ! -f "$src" ]]; then
        log_error "deploy_file: source missing: $src"
        return 1
    fi

    # Drift recovery: old strategy left a symlink here. Remove before write.
    if [[ -L "$dst" ]]; then
        rm -f "$dst"
    fi

    # Ensure parent directory exists
    dst_dir="$(dirname "$dst")"
    [[ -d "$dst_dir" ]] || mkdir -p "$dst_dir"

    # Atomic write via tempfile in same dir (cross-fs mv safe)
    tmp="$(mktemp "${dst}.XXXXXX")"
    if cp "$src" "$tmp" && mv -f "$tmp" "$dst"; then
        log_success "Deployed: $dst"
        return 0
    fi

    rm -f "$tmp" >/dev/null 2>&1
    log_error "deploy_file: failed to deploy $src -> $dst"
    return 1
}

# check_deployed was removed by OPS-043 (#1337) along with its only call site.
# It was deploy_file's pair -- the assertion that a file in $HOME still matches
# the copy in ~/.dotfiles -- and it was the ONLY byte-level check on that leg
# anywhere in the repo. It moved rather than being dropped: doctor's
# `Deploy-dir<->$HOME drift` section carries the same two severities (content
# drift and a symlink where ADR-012 expects a regular file) over eleven files
# instead of three, and covers Windows, which never had this function's twin
# wired to anything.

# Checks if file exists and exits if not found
# Input: $1 - file path, $2 - custom error message (optional)
# Output: exits with error if file not found
# Usage: check_file_exists "/path/to/file" "Config file missing"
check_file_exists() {
    file="$1"
    error_msg="${2:-File not found: $file}"
    
    if [[ ! -f "$file" ]]; then
        exit_error "$error_msg"
    fi
}

# Enhanced dependency functions

# Checks if command exists and exits if not found (strict mode)
# Input: $1 - command name, $2 - package name (optional), $3 - install message (optional)
# Output: info message if found, exits with error if not found
# Usage: require_command "git" "git-core" "Visit: https://git-scm.com"
require_command() {
    cmd="$1"
    package="${2:-$cmd}"
    install_msg="${3:-Please install $package}"
    
    if ! command -v "$cmd" >/dev/null 2>&1; then
        exit_error "$cmd is not installed. $install_msg"
    else
        log_info "$cmd is available"
    fi
}

# Checks if current directory is a git repository
# Input: none
# Output: exits with error if not in git repository
# Usage: check_git_repo
check_git_repo() {
    if [[ ! -d .git ]]; then
        exit_error "Not in a git repository. Please run this from the root of your git project."
    fi
}

# Progress and counting functions

# Initializes a counter variable to zero
# Input: $1 - variable name for the counter
# Output: creates global variable with initial value 0
# Usage: init_counter "file_count"
init_counter() {
    eval "$1=0"
}

# Increments a counter variable by 1
# Input: $1 - variable name of the counter
# Output: increments the global variable value
# Usage: increment_counter "file_count"
increment_counter() {
    var_name="$1"
    local current_val=0
    if [[ -n "${ZSH_VERSION:-}" ]]; then
        # shellcheck disable=SC2296,SC2034
        current_val="${(P)var_name}"
    else
        eval "current_val=\${$var_name}"
    fi
    eval "$var_name=\$((current_val + 1))"
}

# Gets the current value of a counter variable
# Input: $1 - variable name of the counter
# Output: current counter value to stdout
# Usage: count=$(get_counter "file_count")
get_counter() {
    var_name="$1"
    if [[ -n "${ZSH_VERSION:-}" ]]; then
        # shellcheck disable=SC2296
        echo "${(P)var_name}"
    else
        echo "${!var_name}"
    fi
}

# File processing functions

# Processes files matching a pattern in a directory
# Input: $1 - directory path, $2 - file pattern, $3 - callback function name (optional)
# Output: number of files processed to stdout, returns 1 if directory not found
# Usage: count=$(process_files "/path/to/dir" "*.txt" "my_function")
process_files() {
    dir="$1"
    pattern="$2"
    callback="$3"
    count=0
    
    if [[ ! -d "$dir" ]]; then
        log_warning "Directory not found: $dir"
        return 1
    fi
    
    for file in "$dir"/$pattern; do
        [[ ! -f "$file" ]] && continue
        
        if [[ -n "$callback" ]]; then
            "$callback" "$file"
        fi
        count=$((count + 1))
    done
    
    echo "$count"
}

# Safe file operations with error checking

# Copies a file with error checking
# Input: $1 - source file, $2 - destination path
# Output: success/error messages, returns 0 on success, 1 on failure
# Usage: safe_copy "source.txt" "backup/source.txt"
safe_copy() {
    src="$1"
    dest="$2"
    
    if cp "$src" "$dest" 2>/dev/null; then
        log_success "Copied: $(basename "$src") → $(basename "$dest")"
        return 0
    else
        log_error "Failed to copy: $src → $dest"
        return 1
    fi
}

# Moves a file with error checking
# Input: $1 - source file, $2 - destination path
# Output: success/error messages, returns 0 on success, 1 on failure
# Usage: safe_move "old_file.txt" "new_location/file.txt"
safe_move() {
    src="$1"
    dest="$2"
    
    if mv "$src" "$dest" 2>/dev/null; then
        log_success "Moved: $(basename "$src") → $(basename "$dest")"
        return 0
    else
        log_error "Failed to move: $src → $dest"
        return 1
    fi
}

# Removes a file with error checking
# Input: $1 - file path to remove
# Output: success/error messages, returns 0 on success, 1 on failure
# Usage: safe_remove "temp_file.txt"
safe_remove() {
    file="$1"
    
    if rm -f "$file" 2>/dev/null; then
        log_success "Removed: $(basename "$file")"
        return 0
    else
        log_error "Failed to remove: $file"
        return 1
    fi
}

# Enhanced file operations

# Adds a line to a file if not already present
# Input: $1 - file path, $2 - line to add
# Output: returns 0 if line was added, 1 if already exists or file missing
# Usage: ensure_line_in_file "$HOME/.zshrc" 'export PATH=$HOME/bin:$PATH'
ensure_line_in_file() {
    file="$1"
    line="$2"

    file_exists "$file" || return 1

    if ! grep -Fq "$line" "$file"; then
        # Ensure file ends with newline before appending
        [[ -n "$(tail -c1 "$file")" ]] && echo "" >> "$file"
        echo "$line" >> "$file"
        return 0
    fi
    return 1
}

# Creates a secure temporary file and echoes its path
# Input: $1 - prefix for temp file name (optional, defaults to "tmp")
# Output: path to created temp file
# Usage: tmp=$(create_temp_file "ssh_key")
create_temp_file() {
    prefix="${1:-tmp}"
    local tmp_file
    tmp_file=$(mktemp "/tmp/${prefix}.XXXXXX")
    chmod 600 "$tmp_file"
    echo "$tmp_file"
}

# Verifies a symlink exists and is valid, logs result
# Input: $1 - symlink path, $2 - display name (optional)
# Output: success/error log message, returns 0 if valid, 1 otherwise
# Usage: verify_symlink "$HOME/.zshrc" ".zshrc"
verify_symlink() {
    link="$1"
    name="${2:-$(basename "$link")}"

    if symlink_valid "$link"; then
        log_success "$name linked successfully"
        return 0
    else
        log_error "$name link issue"
        return 1
    fi
}

# String helper functions

# Masks a sensitive value for safe logging
# Input: $1 - value to mask, $2 - chars to show at start (optional, defaults to 0)
# Output: masked string to stdout
# Usage: log_info "Token: $(mask_value "$token" 4)"
mask_value() {
    value="$1"
    show_chars="${2:-0}"
    len=${#value}

    # If show_chars >= length, return original (nothing to mask)
    if [[ $show_chars -ge $len ]]; then
        echo "$value"
        return
    fi

    if [[ $show_chars -gt 0 ]]; then
        # shellcheck disable=SC2183
        echo "${value:0:$show_chars}$(printf '%*s' $((len - show_chars)) '' | tr ' ' '*')"
    else
        # shellcheck disable=SC2183
        printf '%*s' "$len" '' | tr ' ' '*'
    fi
}

# Decodes a base64-encoded string
# Input: $1 - base64-encoded string
# Output: decoded string to stdout
# Usage: decoded=$(base64_decode "$encoded")
base64_decode() {
    echo "$1" | base64 --decode
}

# Encodes a string to base64
# Input: $1 - string to encode
# Output: base64-encoded string to stdout
# Usage: encoded=$(base64_encode "$value")
base64_encode() {
    echo -n "$1" | base64
}

# Trims leading and trailing whitespace from a string
# Input: $1 - string to trim
# Output: trimmed string to stdout
# Usage: trimmed=$(trim "  hello  ")
trim() {
    str="$1"
    str="${str#"${str%%[![:space:]]*}"}"
    str="${str%"${str##*[![:space:]]}"}"
    echo "$str"
}

# Environment variable functions

# Safely exports an environment variable
# Input: $1 - variable name, $2 - value
# Output: exports the variable
# Usage: export_var "MY_TOKEN" "secret_value"
export_var() {
    export "$1=$2"
}

# Unsets an environment variable if it exists
# Input: $1 - variable name
# Output: unsets the variable
# Usage: unset_var "MY_TOKEN"
unset_var() {
    unset "$1" 2>/dev/null || true
}

# Obsidian helpers

# Emits the canonical "Cannot reach Obsidian GUI" error + start hint, then
# exits 2. Caller owns the probe shape; this owns the user-facing message
# so it stays in one place across obs-cli.sh and vault-health.sh.
# Input: none
# Output: error log + info hint to stderr/stdout, then exit 2
# Usage: <your_probe> || fail_obsidian_gui
fail_obsidian_gui() {
    log_error "Cannot reach Obsidian GUI. Is Obsidian running?"
    log_info "Start Obsidian or run: obsidian --no-sandbox &"
    exit 2
}

