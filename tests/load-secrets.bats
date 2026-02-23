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
        type secrets_show >/dev/null 2>&1 && echo "show:ok"
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
    [[ "$output" == *"show:ok"* ]]
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

# --- secrets_show (bash) ---

@test "secrets_show is defined after sourcing" {
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        type secrets_show >/dev/null 2>&1 && echo "ok"
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"ok"* ]]
}

@test "secrets_show without args shows usage" {
    run bash -c 'source "$1/utils.sh"; source "$1/load-secrets.sh"; secrets_show' -- "$SCRIPTS_DIR"
    [[ $status -ne 0 ]]
    [[ "$output" == *"Usage"* ]]
}

@test "secrets_show --help shows help" {
    run bash -c 'source "$1/utils.sh"; source "$1/load-secrets.sh"; secrets_show --help' -- "$SCRIPTS_DIR"
    [[ $status -eq 0 ]]
    [[ "$output" == *"Usage"* ]]
    [[ "$output" == *"--raw"* ]]
}

@test "secrets_show unknown option errors" {
    run bash -c 'source "$1/utils.sh"; source "$1/load-secrets.sh"; secrets_show --bogus' -- "$SCRIPTS_DIR"
    [[ $status -ne 0 ]]
    [[ "$output" == *"Unknown option"* ]]
}

@test "secrets_show reports missing mapping" {
    cat > "$SECRETS_DIR/env-mapping.conf" <<'CONF'
TOKEN=my.token
CONF
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        secrets_show NONEXISTENT
    ' -- "$SCRIPTS_DIR"
    [[ $status -ne 0 ]]
    [[ "$output" == *"No mapping found"* ]]
}

@test "secrets_show default mode shows env var from memory" {
    cat > "$SECRETS_DIR/env-mapping.conf" <<'CONF'
TOKEN=my.token
CONF
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        export TOKEN="supersecret123"
        secrets_show TOKEN
    ' -- "$SCRIPTS_DIR"
    [[ $status -eq 0 ]]
    [[ "$output" == "supersecret123" ]]
}

@test "secrets_show default mode reports unloaded env var" {
    cat > "$SECRETS_DIR/env-mapping.conf" <<'CONF'
TOKEN=my.token
CONF
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        unset TOKEN
        secrets_show TOKEN
    ' -- "$SCRIPTS_DIR"
    [[ $status -ne 0 ]]
    [[ "$output" == *"not loaded"* ]]
}

@test "secrets_show default mode shows file secret from disk" {
    local dest_file="/tmp/bats_deployed_secret_$$"
    echo "file-content-here" > "$dest_file"
    cat > "$SECRETS_DIR/env-mapping.conf" <<CONF
@MYFILE=myfile.data>${dest_file}
CONF
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        secrets_show MYFILE
    ' -- "$SCRIPTS_DIR"
    rm -f "$dest_file"
    [[ $status -eq 0 ]]
    [[ "$output" == *"file-content-here"* ]]
}

@test "secrets_show default mode reports undeployed file secret" {
    cat > "$SECRETS_DIR/env-mapping.conf" <<'CONF'
@MYFILE=myfile.data>/tmp/nonexistent_deployed_$$
CONF
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        secrets_show MYFILE
    ' -- "$SCRIPTS_DIR"
    [[ $status -ne 0 ]]
    [[ "$output" == *"not deployed"* ]]
}

@test "secrets_show env var fallback decrypts from .age when not loaded" {
    # Create a real age key and encrypt a test value
    age-keygen -o "$SECRETS_DIR/test_key.txt" 2>/dev/null
    local pubkey
    pubkey=$(grep -o 'age1[0-9a-z]*' "$SECRETS_DIR/test_key.txt")
    echo -n "fallback_secret_value" | age -r "$pubkey" -o "$SECRETS_DIR/my.token.secret.age"

    cat > "$SECRETS_DIR/env-mapping.conf" <<'CONF'
TOKEN=my.token
CONF
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        SECRETS_KEY_PATH="'"$SECRETS_DIR"'/test_key.txt"
        unset TOKEN
        secrets_show TOKEN
    ' -- "$SCRIPTS_DIR"
    [[ $status -eq 0 ]]
    [[ "$output" == "fallback_secret_value" ]]
}

@test "secrets_show file secret fallback decrypts from .age when not deployed" {
    # Create a real age key and encrypt a test file
    age-keygen -o "$SECRETS_DIR/test_key.txt" 2>/dev/null
    local pubkey
    pubkey=$(grep -o 'age1[0-9a-z]*' "$SECRETS_DIR/test_key.txt")
    echo "fallback_file_content" | age -r "$pubkey" -o "$SECRETS_DIR/myfile.data.secret.age"

    local dest_file="/tmp/bats_fallback_deployed_$$"
    rm -f "$dest_file"
    cat > "$SECRETS_DIR/env-mapping.conf" <<CONF
@MYFILE=myfile.data>${dest_file}
CONF
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        SECRETS_KEY_PATH="'"$SECRETS_DIR"'/test_key.txt"
        secrets_show MYFILE
    ' -- "$SCRIPTS_DIR"
    rm -f "$dest_file"
    [[ $status -eq 0 ]]
    [[ "$output" == *"fallback_file_content"* ]]
}

@test "secrets_show --raw reports missing key file" {
    cat > "$SECRETS_DIR/env-mapping.conf" <<'CONF'
TOKEN=my.token
CONF
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        SECRETS_KEY_PATH="/tmp/nonexistent_key_$$"
        secrets_show --raw TOKEN
    ' -- "$SCRIPTS_DIR"
    [[ $status -ne 0 ]]
    [[ "$output" == *"Key file not found"* ]]
}

@test "secrets_show --raw reports missing .age file" {
    cat > "$SECRETS_DIR/env-mapping.conf" <<'CONF'
TOKEN=my.token
CONF
    # Create a fake key file so it passes that check
    touch "$SECRETS_DIR/fake_key.txt"
    run bash -c '
        source "$1/utils.sh"; source "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        SECRETS_KEY_PATH="'"$SECRETS_DIR"'/fake_key.txt"
        secrets_show --raw TOKEN
    ' -- "$SCRIPTS_DIR"
    [[ $status -ne 0 ]]
    [[ "$output" == *"Encrypted file not found"* ]]
}

@test "secrets_help mentions secrets_show" {
    run bash -c 'source "$1/utils.sh"; source "$1/load-secrets.sh"; secrets_help' -- "$SCRIPTS_DIR"
    [[ "$output" == *"secrets_show"* ]]
}

# --- secrets_show (zsh) ---

@test "secrets_show is defined under zsh" {
    run zsh -c '
        . "$1/utils.sh"; . "$1/load-secrets.sh"
        type secrets_show >/dev/null 2>&1 && echo "ok"
    ' -- "$SCRIPTS_DIR"
    [[ "$output" == *"ok"* ]]
}

@test "secrets_show validates inputs under zsh" {
    run zsh -c '. "$1/utils.sh"; . "$1/load-secrets.sh"; secrets_show' -- "$SCRIPTS_DIR"
    [[ $status -ne 0 ]]
    [[ "$output" == *"Usage"* ]]
}

@test "secrets_show default mode works under zsh" {
    cat > "$SECRETS_DIR/env-mapping.conf" <<'CONF'
TOKEN=my.token
CONF
    run zsh -c '
        . "$1/utils.sh"; . "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        export TOKEN="zsh_secret_val"
        secrets_show TOKEN
    ' -- "$SCRIPTS_DIR"
    [[ $status -eq 0 ]]
    [[ "$output" == "zsh_secret_val" ]]
}

@test "secrets_show file secret works under zsh" {
    local dest_file="/tmp/bats_zsh_deployed_$$"
    echo "zsh-file-content" > "$dest_file"
    cat > "$SECRETS_DIR/env-mapping.conf" <<CONF
@MYFILE=myfile.data>${dest_file}
CONF
    run zsh -c '
        . "$1/utils.sh"; . "$1/load-secrets.sh"
        SECRETS_DIR="'"$SECRETS_DIR"'"
        SECRETS_MAPPING_FILE="'"$SECRETS_DIR"'/env-mapping.conf"
        secrets_show MYFILE
    ' -- "$SCRIPTS_DIR"
    rm -f "$dest_file"
    [[ $status -eq 0 ]]
    [[ "$output" == *"zsh-file-content"* ]]
}
