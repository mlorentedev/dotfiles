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
    # $2 is optional. Under `set -u`, callers passing only the test name
    # (e.g. skip "Antigravity CLI Health" when agy not in PATH) would
    # otherwise abort the script with `$2: unbound variable`.
    printf '  %bSKIP%b: %s%s\n' "$YELLOW" "$NC" "$1" "${2:+ - $2}"
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
section "1/13" "Core Tools in PATH"

CORE_TOOLS="git zsh bash curl wget jq eza direnv node npm zoxide docker kubectl terraform"
for tool in $CORE_TOOLS; do
    if command_exists "$tool"; then
        pass "$tool found"
    else
        fail "$tool not in PATH"
    fi
done

# ==================================================
section "2/13" "Versioned Tool Paths"

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
section "3/13" "Version Match (versions.conf)"

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
check_version_match "Minikube" "${MINIKUBE_VERSION:-}" "$APPS_HOME/minikube-${MINIKUBE_VERSION:-}"
check_version_match "Go" "${GO_VERSION:-}" "$APPS_HOME/go-${GO_VERSION:-}"

# ==================================================
section "4/13" "Key Symlinks"

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

# BUG-014 canonical install-state assertion
installed_plugins_json="$HOME/.claude/plugins/installed_plugins.json"
if [ -f "$installed_plugins_json" ]; then
    if grep -qF 'claude-mem@thedotmack' "$installed_plugins_json"; then
        pass "claude-mem@thedotmack installed (BUG-014 canonical check)"
    else
        fail "claude-mem@thedotmack NOT in installed_plugins.json -- re-run setup-linux.sh (BUG-014)"
    fi
else
    skip "claude-mem install state -- installed_plugins.json missing (Claude Code never ran)"
fi

# BUG-015: assert the claude-mem hook's path-resolution cascade
_cmhook_C="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
_cmhook_E="${PLUGIN_ROOT:-}"
_cmhook_cands=$({
    [ -n "$_cmhook_E" ] && printf '%s\n' "$_cmhook_E"
    ls -dt "$_cmhook_C/plugins/cache/thedotmack/claude-mem"/[0-9]*/ 2>/dev/null
    printf '%s\n' "$_cmhook_C/plugins/marketplaces/thedotmack-claude-mem/plugin"
})
_cmhook_P=""
while IFS= read -r _r; do
    _r="${_r%/}"
    [ -d "$_r/plugin/scripts" ] && _q="$_r/plugin" || _q="$_r"
    if [ -f "$_q/scripts/bun-runner.js" ] && [ -f "$_q/scripts/worker-service.cjs" ]; then
        _cmhook_P="$_q"
        break
    fi
done <<<"$_cmhook_cands"
if [ -n "$_cmhook_P" ]; then
    pass "claude-mem hook path resolves to: $_cmhook_P (BUG-015)"
else
    fail "claude-mem hook path resolution FAILED -- run claude-mem-heal.sh (BUG-015)"
fi

# ==================================================
section "5/13" "Environment Variables"

ENV_VARS="DOTFILES_DIR APPS_HOME JAVA_HOME MAVEN_HOME PYTHON_HOME GO_HOME MINIKUBE_HOME"
for var in $ENV_VARS; do
    if var_is_set "$var"; then
        pass "$var is set"
    else
        fail "$var is not set"
    fi
done

# ==================================================
section "6/13" "Optional Tools"

OPTIONAL_TOOLS="age gh claude gemini agy bats shellcheck helm ansible pip"
for tool in $OPTIONAL_TOOLS; do
    if command_exists "$tool"; then
        pass "$tool found"
    else
        skip "$tool" "not installed"
    fi
done

# ==================================================
section "7/13" "Knowledge Vault"

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
section "8/13" "Secrets Integrity"

SECRETS_DIR="${DOTFILES_DIR}/sensitive"
SECRETS_MAPPING="$SECRETS_DIR/env-mapping.conf"

if [ -f "$SECRETS_MAPPING" ]; then
    pass "env-mapping.conf exists"

    while IFS= read -r _line || [ -n "$_line" ]; do
        case "$_line" in
            "#"*|"") continue ;;
        esac
        [ -z "$_line" ] && continue
        echo "$_line" | grep -q '=' || continue

        _var="${_line%%=*}"
        _val="${_line#*=}"
        _var="$(echo "$_var" | tr -d ' ')"
        _val="$(echo "$_val" | tr -d ' ')"

        case "$_var" in
            @*) _fname="${_val%%>*}" ; _var="${_var#@} [file]" ;;
            *)  _fname="$_val" ;;
        esac

        if [ -f "$SECRETS_DIR/${_fname}.secret.age" ]; then
            pass "$_var -> ${_fname}.secret.age"
        else
            fail "$_var -> ${_fname}.secret.age (missing)"
        fi
    done < "$SECRETS_MAPPING"

    for _age_file in "$SECRETS_DIR"/*.secret.age; do
        [ -f "$_age_file" ] || continue
        _base=$(basename "$_age_file" .secret.age)
        if ! grep -q "=${_base}$" "$SECRETS_MAPPING" 2>/dev/null &&
           ! grep -q "=${_base}>" "$SECRETS_MAPPING" 2>/dev/null; then
            fail "Orphan: ${_base}.secret.age (no mapping)"
        fi
    done
else
    fail "env-mapping.conf not found"
fi

# ==================================================
section "9/13" "tmux"

if ! command -v tmux >/dev/null 2>&1; then
    fail "tmux not installed (run: sudo apt install -y tmux)"
else
    pass "tmux installed: $(tmux -V)"
    # SDD-007: tmux.conf deployed via deploy_file (copy, not symlink)
    _tmux_src="${DOTFILES_DIR:-$HOME/.dotfiles}/tmux.conf"
    if [ -f "$_tmux_src" ]; then
        check_deployed "$_tmux_src" "$HOME/.tmux.conf" "$HOME/.tmux.conf"
    elif [ -f "$HOME/.tmux.conf" ]; then
        pass "$HOME/.tmux.conf exists (source unavailable for drift check)"
    else
        fail "$HOME/.tmux.conf missing (run setup-linux.sh)"
    fi
fi

# ==================================================
section "10/13" "OpenCode"

OPENCODE_BIN="$HOME/.opencode/bin/opencode"
OPENCODE_CFG="$HOME/.config/opencode/opencode.jsonc"
if command -v opencode >/dev/null 2>&1; then
    pass "opencode in PATH: $(opencode --version 2>&1 | head -1)"
elif [ -x "$OPENCODE_BIN" ]; then
    fail "opencode binary exists at $OPENCODE_BIN but not in PATH (reload shell or check .zshrc/.bashrc)"
else
    fail "opencode binary missing (run setup-linux.sh)"
fi
if [ -f "$OPENCODE_CFG" ]; then
    pass "opencode.jsonc deployed: $OPENCODE_CFG"
    # shellcheck disable=SC2016  # literal "$schema" string match, no expansion intended
    if grep -q '"\$schema":' "$OPENCODE_CFG"; then
        pass "opencode.jsonc has \$schema declaration"
    else
        fail "opencode.jsonc missing \$schema declaration (run setup-linux.sh to redeploy)"
    fi
else
    fail "opencode.jsonc missing: $OPENCODE_CFG (run setup-linux.sh)"
fi

# ==================================================
section "11/13" "Ghostty"

GHOSTTY_CFG="$HOME/.config/ghostty/config"
GHOSTTY_PINNED="$(grep -E '^GHOSTTY_VERSION=' "$DOTFILES_DIR/versions.conf" 2>/dev/null | cut -d= -f2)"
if command -v ghostty >/dev/null 2>&1; then
    pass "ghostty in PATH: $(ghostty --version 2>&1 | head -1)"
    INSTALLED_GHOSTTY=$(ghostty --version 2>&1 | head -1 | awk '{print $2}' | sed 's/-.*//')
    if [ -n "$GHOSTTY_PINNED" ] && [ "$INSTALLED_GHOSTTY" = "$GHOSTTY_PINNED" ]; then
        pass "ghostty version matches versions.conf ($GHOSTTY_PINNED)"
    elif [ -n "$GHOSTTY_PINNED" ]; then
        fail "ghostty version drift: installed=$INSTALLED_GHOSTTY pinned=$GHOSTTY_PINNED"
    else
        skip "GHOSTTY_VERSION not set in versions.conf — version match not verified"
    fi
else
    skip "ghostty not installed (sudo apt install -y ghostty)"
fi
if [ -f "$GHOSTTY_CFG" ]; then
    pass "ghostty config deployed: $GHOSTTY_CFG"
    if grep -q '^theme =' "$GHOSTTY_CFG" && grep -q '^font-family =' "$GHOSTTY_CFG"; then
        pass "ghostty config has theme + font-family"
    else
        fail "ghostty config missing theme or font-family (run setup-linux.sh to redeploy)"
    fi
else
    fail "ghostty config missing: $GHOSTTY_CFG (run setup-linux.sh)"
fi

# ==================================================
section "12/13" "Repo ↔ Deploy-Dir Drift"

if [ -x "$SCRIPT_DIR/diff-check.sh" ]; then
    if "$SCRIPT_DIR/diff-check.sh" >/dev/null 2>&1; then
        pass "no drift between repo and $DOTFILES_DIR"
    else
        fail "drift detected (run: $SCRIPT_DIR/diff-check.sh to inspect, then ./setup-linux.sh)"
    fi
else
    skip "diff-check.sh not found at $SCRIPT_DIR/diff-check.sh"
fi

# Harness override drift (ENGINE-001): generated blocks vs their source-of-record.
# Offline (no vault). compile-harness resolves its own root from the script's
# location, so this works from the non-git deploy copy too — no CWD juggling.
# --check also validates every committed skill record (SDD-008) renders cleanly,
# so this one gate covers both the override blocks and the skill records.
if [ -x "$SCRIPT_DIR/compile-harness.sh" ]; then
    if "$SCRIPT_DIR/compile-harness.sh" --check >/dev/null 2>&1; then
        pass "harness blocks + skill records match their source-of-record (no drift)"
    else
        fail "harness/skill drift (run: $SCRIPT_DIR/compile-harness.sh --refresh, then re-deploy)"
    fi
else
    skip "compile-harness.sh not found at $SCRIPT_DIR/compile-harness.sh"
fi

# Skill deployment integrity (SDD-008, AC1): deployed skills MUST be regular
# copies, never symlinks/junctions (the BUG-100 fragility class). --check above
# verifies the records render; this asserts the on-machine deploy is symlink-free.
# Deployed skill paths MUST be regular hard copies, never symlinks (BUG-100 /
# ADR-012). The invariant is intentionally strict — ANY symlink here is fragile
# (a dangling/foreign link breaks discovery), so we surface all of them. Report
# every offending path (not just the first) so cleanup is one pass.
_skill_dirs=""
for _d in "$HOME/.claude/skills" "$HOME/.config/opencode/commands" "$HOME/.gemini/skills" "$HOME/.gemini/prompts"; do
    [ -d "$_d" ] && _skill_dirs="$_skill_dirs $_d"
done
if [ -n "$_skill_dirs" ]; then
    # shellcheck disable=SC2086  # _skill_dirs is an intentional space-separated list
    _skill_links="$(find $_skill_dirs -type l 2>/dev/null)"
    if [ -n "$_skill_links" ]; then
        fail "deployed skill path(s) are symlinks (must be hard copies — BUG-100):"
        printf '%s\n' "$_skill_links" | sed 's|^|        |'
        printf '        Fix: ours -> %s/compile-harness.sh --deploy; foreign/dangling -> remove the link.\n' "$SCRIPT_DIR"
    else
        pass "deployed skills are regular copies (no symlinks in skill paths)"
    fi
else
    skip "no deployed skill paths found (run ./setup-linux.sh to deploy skills)"
fi

# ==================================================
section "13/13" "Antigravity CLI Health"

if command_exists agy; then
    # Verify production endpoint (with fallback for check)
    _agy_endpoint="${ANTIGRAVITY_ENDPOINT:-https://cloudcode-pa.googleapis.com}"
    if [ "$_agy_endpoint" = "https://cloudcode-pa.googleapis.com" ]; then
        pass "ANTIGRAVITY_ENDPOINT set to production"
    else
        fail "ANTIGRAVITY_ENDPOINT is not production: $_agy_endpoint"
    fi

    # Verify AGY_APP_DATA is absolute (with fallback to default location)
    _agy_data="${AGY_APP_DATA:-$HOME/.gemini/antigravity-cli}"
    if [[ "$_agy_data" == /* ]]; then
        pass "AGY_APP_DATA is absolute"
    else
        fail "AGY_APP_DATA is relative or unset: $_agy_data"
    fi

    # Verify master MCP config at agy's canonical read path (SDD-007: no symlinks).
    _master_config="$GEMINI_HOME/config/mcp_config.json"
    if [ ! -e "$_master_config" ]; then
        fail "Master mcp_config.json missing at $_master_config (run setup-linux.sh)"
    elif [ -L "$_master_config" ]; then
        fail "Master mcp_config.json is a symlink (BUG-100 regression — recursion risk)"
    elif ! jq '.' "$_master_config" >/dev/null 2>&1; then
        fail "Master mcp_config.json at $_master_config is invalid JSON"
    else
        pass "Master mcp_config.json is a real file with valid JSON"
    fi

    # Assert no symlinks anywhere under ~/.gemini/config/ (BUG-100 regression guard).
    if [ -d "$GEMINI_HOME/config" ]; then
        _bad_links=$(find "$GEMINI_HOME/config" -maxdepth 3 -type l 2>/dev/null | head -3)
        if [ -n "$_bad_links" ]; then
            fail "Symlinks found under ~/.gemini/config/ (BUG-100 regression): $_bad_links"
        else
            pass "No symlinks under ~/.gemini/config/ (BUG-100 guard)"
        fi
    fi
else
    skip "Antigravity CLI Health" "agy not in PATH"
fi

# ============================================================
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
