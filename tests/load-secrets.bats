#!/usr/bin/env bats
# Tests for scripts/load-secrets.sh - secrets management

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
    # Source without auto-loading secrets (set as direct execution)
    export SECRETS_DIR="/tmp/bats_secrets_$$"
    export SECRETS_MAPPING_FILE="$SECRETS_DIR/env-mapping.conf"
    export SECRETS_KEY_PATH="/tmp/bats_nonexistent_key_$$"
    mkdir -p "$SECRETS_DIR"
}

teardown() {
    rm -rf "/tmp/bats_secrets_$$"
}

@test "load-secrets.sh sources without error" {
    # Source utils first, then load-secrets (without auto-load since key is missing)
    run bash -c 'source "$1/utils.sh"; source "$1/load-secrets.sh"' -- "$SCRIPTS_DIR"
    [[ $status -eq 0 ]]
}

@test "secrets functions are defined after sourcing" {
    run bash -c '
        source "$1/utils.sh"
        source "$1/load-secrets.sh"
        type secrets_load >/dev/null 2>&1 && echo "load:ok"
        type secrets_list >/dev/null 2>&1 && echo "list:ok"
        type secrets_get >/dev/null 2>&1 && echo "get:ok"
        type secrets_refresh >/dev/null 2>&1 && echo "refresh:ok"
        type secrets_add >/dev/null 2>&1 && echo "add:ok"
        type secrets_rotate >/dev/null 2>&1 && echo "rotate:ok"
        type secrets_check >/dev/null 2>&1 && echo "check:ok"
        type secrets_clean >/dev/null 2>&1 && echo "clean:ok"
        type secrets_help >/dev/null 2>&1 && echo "help:ok"
        type secrets_audit >/dev/null 2>&1 && echo "audit:ok"
        type secrets_sync >/dev/null 2>&1 && echo "sync:ok"
        type secrets_add_file >/dev/null 2>&1 && echo "add_file:ok"
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"load:ok"* ]]
    [[ "$output" == *"list:ok"* ]]
    [[ "$output" == *"get:ok"* ]]
    [[ "$output" == *"refresh:ok"* ]]
    [[ "$output" == *"add:ok"* ]]
    [[ "$output" == *"rotate:ok"* ]]
    [[ "$output" == *"check:ok"* ]]
    [[ "$output" == *"clean:ok"* ]]
    [[ "$output" == *"help:ok"* ]]
    [[ "$output" == *"audit:ok"* ]]
    [[ "$output" == *"sync:ok"* ]]
    [[ "$output" == *"add_file:ok"* ]]
}

@test "secrets_help shows usage information" {
    run bash -c 'source "$1/utils.sh"; source "$1/load-secrets.sh"; secrets_help' -- "$SCRIPTS_DIR"
    [[ "$output" == *"Secrets Management Commands"* ]]
    [[ "$output" == *"secrets_add"* ]]
    [[ "$output" == *"secrets_rotate"* ]]
}

@test "secrets_list handles missing mapping file" {
    run bash -c '
        export DOTFILES_DIR="/tmp/nonexist_dotfiles_$$"
        source "$1/utils.sh"
        source "$1/load-secrets.sh"
        SECRETS_MAPPING_FILE="/nonexistent_mapping"
        secrets_list
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"not found"* ]]
}

@test "secrets_add validates inputs" {
    run bash -c 'source "$1/utils.sh"; source "$1/load-secrets.sh"; secrets_add' -- "$SCRIPTS_DIR"
    [[ "$output" == *"Usage"* ]]
}

@test "secrets_rotate validates inputs" {
    run bash -c 'source "$1/utils.sh"; source "$1/load-secrets.sh"; secrets_rotate' -- "$SCRIPTS_DIR"
    [[ "$output" == *"Usage"* ]]
}

@test "secrets_clean --dry-run works" {
    run bash -c '
        export SECRETS_DIR="'"$SECRETS_DIR"'"
        source "$1/utils.sh"
        source "$1/load-secrets.sh"
        secrets_clean --dry-run
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"Dry run complete"* ]]
}

@test "secrets_sync requires DOTFILES_REPO_DIR" {
    run bash -c '
        unset DOTFILES_REPO_DIR
        source "$1/utils.sh"
        source "$1/load-secrets.sh"
        secrets_sync
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"DOTFILES_REPO_DIR"* ]]
}

# --- File secret helpers (bash) ---

@test "_is_file_secret detects @ prefix" {
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        _is_file_secret "@KUBECONFIG" && echo "yes" || echo "no"
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"yes"* ]]
}

@test "_is_file_secret rejects without @ prefix" {
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        _is_file_secret "KUBECONFIG" && echo "yes" || echo "no"
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"no"* ]]
}

@test "_parse_file_secret_value splits filename and dest_path" {
    run bash -c '
        export HOME="/home/testuser"
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        _parse_file_secret_value "kubelab.kubeconfig>~/.kube/kubelab.config"
        echo "file=$_FS_FILENAME"
        echo "dest=$_FS_DEST_PATH"
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"file=kubelab.kubeconfig"* ]]
    [[ "$output" == *"dest=/home/testuser/.kube/kubelab.config"* ]]
}

@test "secrets_list shows File Secrets section" {
    cat > "$SECRETS_DIR/env-mapping.conf" <<'CONF'
TOKEN=my.token
@KUBECONFIG=kubelab.kubeconfig>~/.kube/kubelab.config
CONF
    # Create the .age files so they show as valid
    touch "$SECRETS_DIR/my.token.secret.age"
    touch "$SECRETS_DIR/kubelab.kubeconfig.secret.age"

    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        secrets_list
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"Environment Variables:"* ]]
    [[ "$output" == *"File Secrets:"* ]]
    [[ "$output" == *"KUBECONFIG"* ]]
}

@test "secrets_check detects missing file secret .age" {
    cat > "$SECRETS_DIR/env-mapping.conf" <<'CONF'
@KUBECONFIG=kubelab.kubeconfig>~/.kube/kubelab.config
CONF

    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        secrets_check
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"KUBECONFIG"* ]]
    [[ "$output" == *"missing"* ]]
}

@test "secrets_check validates file secret with .age present" {
    cat > "$SECRETS_DIR/env-mapping.conf" <<'CONF'
@KUBECONFIG=kubelab.kubeconfig>~/.kube/kubelab.config
CONF
    touch "$SECRETS_DIR/kubelab.kubeconfig.secret.age"

    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        secrets_check
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"KUBECONFIG [file]"* ]]
    [[ "$output" == *"Valid:    1"* ]]
}

@test "secrets_add_file validates inputs" {
    run bash -c 'source "$1/utils.sh"; source "$1/load-secrets.sh"; secrets_add_file' -- "$SCRIPTS_DIR"
    [[ "$output" == *"Usage"* ]]
}

@test "secrets_help mentions FILE SECRETS and secrets_add_file" {
    run bash -c 'source "$1/utils.sh"; source "$1/load-secrets.sh"; secrets_help' -- "$SCRIPTS_DIR"
    [[ "$output" == *"FILE SECRETS"* ]]
    [[ "$output" == *"secrets_add_file"* ]]
}

# --- Zsh compatibility ---

@test "load-secrets.sh sources under zsh" {
    run zsh -c '. "$1/utils.sh"; . "$1/load-secrets.sh"' -- "$SCRIPTS_DIR"
    [[ $status -eq 0 ]]
}

@test "secrets functions defined under zsh" {
    run zsh -c '
        . "$1/utils.sh"
        . "$1/load-secrets.sh"
        type secrets_help >/dev/null 2>&1 && echo "ok"
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"ok"* ]]
}

@test "secrets_help works under zsh" {
    run zsh -c '. "$1/utils.sh"; . "$1/load-secrets.sh"; secrets_help' -- "$SCRIPTS_DIR"
    [[ "$output" == *"Secrets Management Commands"* ]]
}

@test "secrets_clean --dry-run works under zsh" {
    run zsh -c '
        export SECRETS_DIR="'"$SECRETS_DIR"'"
        . "$1/utils.sh"
        . "$1/load-secrets.sh"
        secrets_clean --dry-run
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"Dry run"* ]]
}

# --- File secret helpers (zsh) ---

@test "_is_file_secret works under zsh" {
    run zsh -c '
        . "$1/utils.sh"; . "$1/load-secrets.sh"
        _is_file_secret "@KUBECONFIG" && echo "yes" || echo "no"
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"yes"* ]]
}

@test "_parse_file_secret_value works under zsh" {
    run zsh -c '
        export HOME="/home/testuser"
        . "$1/utils.sh"; . "$1/load-secrets.sh"
        _parse_file_secret_value "kubelab.kubeconfig>~/.kube/kubelab.config"
        echo "file=$_FS_FILENAME"
        echo "dest=$_FS_DEST_PATH"
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"file=kubelab.kubeconfig"* ]]
    [[ "$output" == *"dest=/home/testuser/.kube/kubelab.config"* ]]
}

@test "secrets_add_file validates inputs under zsh" {
    run zsh -c '. "$1/utils.sh"; . "$1/load-secrets.sh"; secrets_add_file' -- "$SCRIPTS_DIR"
    [[ "$output" == *"Usage"* ]]
}
