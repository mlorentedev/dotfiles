#!/bin/bash

# healthcheck.sh: Post-setup tool and version verification
# Usage: ./scripts/healthcheck.sh
# Exit: 0 if all checks pass, 1 if any fail

set -euo pipefail

# Load utility functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
if [ -f "$SCRIPT_DIR/utils.sh" ]; then
    # shellcheck source=utils.sh disable=SC1091
    . "$SCRIPT_DIR/utils.sh"
else
    echo "Error: utils.sh not found in $SCRIPT_DIR"
    exit 1
fi

# Load versions.conf
DOTFILES_DIR="${DOTFILES_DIR:-$HOME/.dotfiles}"
VERSIONS_CONF="$DOTFILES_DIR/versions.conf"
if [ -f "$VERSIONS_CONF" ]; then
    # shellcheck disable=SC1090
    . "$VERSIONS_CONF"
else
    # Try repo-local copy
    REPO_VERSIONS="$(dirname "$SCRIPT_DIR")/versions.conf"
    if [ -f "$REPO_VERSIONS" ]; then
        # shellcheck disable=SC1090
        . "$REPO_VERSIONS"
    else
        log_warning "versions.conf not found, version checks will be skipped"
    fi
fi

# Test counters
CHECKS_PASSED=0
CHECKS_FAILED=0
CHECKS_SKIPPED=0

pass() {
    printf '  %bPASS%b: %s\n' "$GREEN" "$NC" "$1"
    CHECKS_PASSED=$((CHECKS_PASSED + 1))
}

fail() {
    printf '  %bFAIL%b: %s\n' "$RED" "$NC" "$1"
    CHECKS_FAILED=$((CHECKS_FAILED + 1))
}

skip() {
    printf '  %bSKIP%b: %s - %s\n' "$YELLOW" "$NC" "$1" "$2"
    CHECKS_SKIPPED=$((CHECKS_SKIPPED + 1))
}

section() {
    echo ""
    printf '%b[%s]%b %s\n' "$BLUE" "$1" "$NC" "$2"
}

# ============================================================
echo "========================================"
echo "   DOTFILES HEALTH CHECK"
echo "========================================"
echo "Checking from: $DOTFILES_DIR"

# ==================================================
section "1/7" "Core Tools in PATH"

CORE_TOOLS="git zsh bash curl wget jq eza direnv node npm zoxide docker kubectl terraform"
for tool in $CORE_TOOLS; do
    if command_exists "$tool"; then
        pass "$tool found"
    else
        fail "$tool not in PATH"
    fi
done

# ==================================================
section "2/7" "Versioned Tool Paths"

check_tool_home() {
    local name="$1"
    local dir="$2"
    local binary="$3"

    if [ -z "$dir" ]; then
        skip "$name" "variable not set"
        return
    fi

    if [ -d "$dir" ] && [ -x "$dir/bin/$binary" ]; then
        pass "$name ($dir)"
    elif [ -d "$dir" ]; then
        fail "$name directory exists but $binary not found in $dir/bin/"
    else
        fail "$name directory missing: $dir"
    fi
}

check_tool_home "JAVA_HOME" "${JAVA_HOME:-}" "java"
check_tool_home "MAVEN_HOME" "${MAVEN_HOME:-}" "mvn"
check_tool_home "PYTHON_HOME" "${PYTHON_HOME:-}" "python3"
check_tool_home "RUBY_HOME" "${RUBY_HOME:-}" "ruby"
check_tool_home "GO_HOME" "${GO_HOME:-}" "go"

# Minikube has no bin/ subdirectory
if [ -n "${MINIKUBE_HOME:-}" ] && [ -d "${MINIKUBE_HOME:-}" ]; then
    if [ -x "$MINIKUBE_HOME/minikube" ]; then
        pass "MINIKUBE_HOME ($MINIKUBE_HOME)"
    else
        fail "MINIKUBE_HOME directory exists but minikube binary not found"
    fi
elif [ -n "${MINIKUBE_HOME:-}" ]; then
    fail "MINIKUBE_HOME directory missing: $MINIKUBE_HOME"
else
    skip "MINIKUBE_HOME" "variable not set"
fi

# ==================================================
section "3/7" "Version Match (versions.conf)"

check_version_match() {
    local name="$1"
    local expected="$2"
    local dir_path="$3"

    if [ -z "$expected" ]; then
        skip "$name version" "not set in versions.conf"
        return
    fi

    if [ -d "$dir_path" ]; then
        pass "$name version $expected (directory exists)"
    else
        fail "$name expected version $expected but directory missing: $dir_path"
    fi
}

APPS_HOME="${APPS_HOME:-$HOME/Applications}"
check_version_match "Java" "${JAVA_VERSION:-}" "$APPS_HOME/jdk-${JAVA_VERSION:-}"
check_version_match "Maven" "${MAVEN_VERSION:-}" "$APPS_HOME/apache-maven-${MAVEN_VERSION:-}"
check_version_match "Python" "${PYTHON_VERSION:-}" "$APPS_HOME/python-${PYTHON_VERSION:-}"
check_version_match "Ruby" "${RUBY_VERSION:-}" "$APPS_HOME/ruby-${RUBY_VERSION:-}"
check_version_match "Minikube" "${MINIKUBE_VERSION:-}" "$APPS_HOME/minikube-${MINIKUBE_VERSION:-}"
check_version_match "Go" "${GO_VERSION:-}" "$APPS_HOME/go-${GO_VERSION:-}"

# ==================================================
section "4/7" "Key Symlinks"

check_symlink() {
    local path="$1"
    local name="$2"

    if [ -L "$path" ] && [ -e "$path" ]; then
        pass "$name symlink valid"
    elif [ -L "$path" ]; then
        fail "$name symlink broken (dangling)"
    elif [ -e "$path" ]; then
        pass "$name exists (not a symlink)"
    else
        fail "$name missing: $path"
    fi
}

check_symlink "$HOME/.dotfiles" ".dotfiles"
check_symlink "$HOME/.zshrc" ".zshrc"
check_symlink "$HOME/.bashrc" ".bashrc"
check_symlink "$HOME/.zsh/aliases.zsh" ".zsh/aliases.zsh"
check_symlink "$HOME/.zsh/functions.zsh" ".zsh/functions.zsh"
check_symlink "$HOME/.ssh/config" ".ssh/config"

# ==================================================
section "5/7" "Environment Variables"

ENV_VARS="DOTFILES_DIR APPS_HOME JAVA_HOME MAVEN_HOME PYTHON_HOME RUBY_HOME GO_HOME MINIKUBE_HOME GEM_HOME"
for var in $ENV_VARS; do
    if var_is_set "$var"; then
        pass "$var is set"
    else
        fail "$var is not set"
    fi
done

# ==================================================
section "6/7" "Optional Tools"

OPTIONAL_TOOLS="age gh claude gemini bats shellcheck helm ansible pip"
for tool in $OPTIONAL_TOOLS; do
    if command_exists "$tool"; then
        pass "$tool found"
    else
        skip "$tool" "not installed"
    fi
done

# ==================================================
section "7/7" "Knowledge Vault"

VAULT_DIR="${VAULT_DIR:-$HOME/Projects/knowledge}"

if [ -d "$VAULT_DIR" ]; then
    pass "Vault directory exists ($VAULT_DIR)"
else
    fail "Vault directory missing: $VAULT_DIR"
fi

if [ -d "$VAULT_DIR/.obsidian" ]; then
    pass ".obsidian/ configured"
else
    fail ".obsidian/ directory missing"
fi

if [ -f "$VAULT_DIR/.obsidian/types.json" ]; then
    pass "types.json present"
else
    fail "types.json missing (property schema)"
fi

if command_exists obsidian; then
    pass "Obsidian CLI in PATH"
else
    fail "Obsidian CLI not in PATH"
fi

if [ -x "$SCRIPT_DIR/vault-health.sh" ]; then
    pass "vault-health.sh exists and executable"
else
    fail "vault-health.sh missing or not executable"
fi

LINTER_CONFIG="$VAULT_DIR/.obsidian/plugins/obsidian-linter/data.json"
if [ -f "$LINTER_CONFIG" ]; then
    if grep -q '"lintOnSave": true' "$LINTER_CONFIG" 2>/dev/null; then
        pass "Linter lintOnSave enabled"
    else
        fail "Linter lintOnSave disabled"
    fi
else
    skip "Linter config" "obsidian-linter not installed"
fi

VAULT_DIRS="00_meta 10_projects 40_resources"
for dir in $VAULT_DIRS; do
    if [ -d "$VAULT_DIR/$dir" ]; then
        pass "Vault directory: $dir/"
    else
        fail "Vault directory missing: $dir/"
    fi
done

# ==================================================
echo ""
echo "========================================"
printf 'Results: %b%d passed%b, %b%d failed%b, %b%d skipped%b\n' \
    "$GREEN" "$CHECKS_PASSED" "$NC" \
    "$RED" "$CHECKS_FAILED" "$NC" \
    "$YELLOW" "$CHECKS_SKIPPED" "$NC"
echo "========================================"

if [ "$CHECKS_FAILED" -gt 0 ]; then
    exit 1
fi
exit 0
