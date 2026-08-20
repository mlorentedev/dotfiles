#!/bin/bash

# setup-linux.sh: Install dotfiles and set up environment
# Usage: ./setup-linux.sh
#
# Philosophy: opportunistic configuration. The script attempts to set up
# every supported tool and warns when something is not available, rather
# than requiring opt-in flags. Heavy / disk-intensive installs (e.g.
# future Ollama) may add explicit flags case by case.

set -euo pipefail

# Load utility functions for logging
if [ -f ./scripts/utils.sh ]; then
    . ./scripts/utils.sh
else
    echo "Error: utils.sh not found"
    exit 1
fi

# Current directory and target directory
CURRENT_DIR=$(pwd)

# Create necessary directories
export DOTFILES_DIR="$HOME/.dotfiles"

# Refuse an in-place install (#695). The supported layout clones the repo to a
# SEPARATE directory (e.g. ~/dotfiles-repo) and deploys into ~/.dotfiles. When
# the repo is cloned directly into ~/.dotfiles, CURRENT_DIR == DOTFILES_DIR and
# setup's deploy copies collapse to same-file operations: `cp -f X X` aborts
# under `set -euo pipefail` (env-contract deploy), and the git-hooks clean-mirror
# (rm -rf dest + cp src/.) would EMPTY the dispatcher while still logging success.
# Fail fast HERE, before any directory is created or any file is copied.
if [ "$CURRENT_DIR" = "$DOTFILES_DIR" ]; then
    log_error "Refusing in-place install: the repo is checked out in \$DOTFILES_DIR ($DOTFILES_DIR)."
    log_error "That directory is the DEPLOY TARGET, not the checkout. Clone somewhere else and re-run:"
    log_error "    git clone https://github.com/mlorentedev/dotfiles.git ~/dotfiles-repo"
    log_error "    cd ~/dotfiles-repo && bash setup-linux.sh"
    exit 1
fi

# Single source of truth for tool versions (REFACTOR-011): source the manifest
# from the checkout so $OPENCODE_VERSION / etc. are reliable
# regardless of the parent shell. Read from $CURRENT_DIR: the copy under
# $DOTFILES_DIR does not exist yet on a first run (it is created below).
[ -f "$CURRENT_DIR/versions.conf" ] && . "$CURRENT_DIR/versions.conf"

log_info "Creating necessary directories..."
ensure_directory "$HOME/.zsh"
ensure_directory "$HOME/.bash"
ensure_directory "$DOTFILES_DIR"
ensure_directory "$DOTFILES_DIR/.zsh"
ensure_directory "$DOTFILES_DIR/scripts"

# Copy all files to the dotfiles directory
log_info "Setting up dotfiles in $DOTFILES_DIR..."
if [ "$CURRENT_DIR" != "$DOTFILES_DIR" ]; then
    # Copy files to the dotfiles directory
    log_info "Copying files from $CURRENT_DIR to $DOTFILES_DIR..."
    safe_copy "$CURRENT_DIR/versions.conf" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/packages.json" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/mcp-servers.json" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.zshrc" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.profile" "$DOTFILES_DIR/" 2>/dev/null || true
    if [ -f "$CURRENT_DIR/.bashrc" ]; then
        safe_copy "$CURRENT_DIR/.bashrc" "$DOTFILES_DIR/" 2>/dev/null || true
    fi    
    safe_copy "$CURRENT_DIR/.gitconfig" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/tmux.conf" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.inputrc" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.editorconfig" "$DOTFILES_DIR/" 2>/dev/null || true
    cp -rf "$CURRENT_DIR/.zsh/." "$DOTFILES_DIR/.zsh/" 2>/dev/null || true
    ensure_directory "$DOTFILES_DIR/ssh"
    cp -rf "$CURRENT_DIR/ssh/"* "$DOTFILES_DIR/ssh/" 2>/dev/null || true
    cp -rf "$CURRENT_DIR/scripts/"* "$DOTFILES_DIR/scripts/" 2>/dev/null || true
else
    log_info "Already in dotfiles directory, skipping copy..."
fi

# Deploy dotfiles via deploy_file (SDD-007 IaC strategy: atomic copy + idempotent).
# Edit-in-repo workflow: never modify ~/.zshrc directly — edit in repo, re-run setup.
log_info "Deploying main dotfiles..."
deploy_file "$DOTFILES_DIR/.zshrc" "$HOME/.zshrc"
[ -f "$DOTFILES_DIR/.profile" ] && deploy_file "$DOTFILES_DIR/.profile" "$HOME/.profile"

# SSH config and public key
log_info "Setting up SSH config..."
ensure_directory "$HOME/.ssh"
deploy_file "$DOTFILES_DIR/ssh/config" "$HOME/.ssh/config"
chmod 600 "$HOME/.ssh/config"
cp "$DOTFILES_DIR/ssh/id_ed25519.pub" "$HOME/.ssh/id_ed25519.pub" 2>/dev/null || true

# Git configuration
log_info "Setting up Git configuration..."
if [ -f "$DOTFILES_DIR/.gitconfig" ]; then
    deploy_file "$DOTFILES_DIR/.gitconfig" "$HOME/.gitconfig"
else
    log_warning ".gitconfig not found in dotfiles"
fi

# Deploy .zsh directory contents
log_info "Setting up .zsh directory and utils.sh..."
ensure_directory "$HOME/.zsh"
deploy_file "$DOTFILES_DIR/.zsh/aliases.zsh" "$HOME/.zsh/aliases.zsh"
deploy_file "$DOTFILES_DIR/.zsh/functions.zsh" "$HOME/.zsh/functions.zsh"
# Portable swiss-army functions (IDEAS-002) — shared by bash + zsh.
deploy_file "$DOTFILES_DIR/.zsh/functions.sh" "$HOME/.zsh/functions.sh"
deploy_file "$DOTFILES_DIR/.zsh/nvm.zsh" "$HOME/.zsh/nvm.zsh"
deploy_file "$DOTFILES_DIR/tmux.conf" "$HOME/.tmux.conf"
# readline config (POLISH-004): case-insensitive completion + smart history.
deploy_file "$DOTFILES_DIR/.inputrc" "$HOME/.inputrc"
chmod +x "$DOTFILES_DIR/scripts/utils.sh"
chmod +x "$DOTFILES_DIR/scripts/age-encrypt-decrypt.sh"
chmod +x "$DOTFILES_DIR/scripts/install-precommit.sh"
chmod +x "$DOTFILES_DIR/scripts/dotfiles-sync.sh"
chmod +x "$DOTFILES_DIR/scripts/vault-health.sh"
chmod +x "$DOTFILES_DIR/scripts/knowledge-crystallize.sh"

# Copy sensitive directory (encrypted *.secret.age files; the mapping lives in secrets/registry.yaml)
log_info "Setting up sensitive directory..."

# Preflight: warn if age identity key is missing. Without it `dotf secrets`
# can't decrypt, so $NAN_API_KEY / $OPENROUTER_API_KEY / etc. resolve empty --
# opencode + agy then 401 with no clear cause. Non-fatal: encrypted files still
# deploy so a key imported later works without re-running setup.
AGE_KEY="${AGE_KEY_PATH:-$HOME/.config/age/key.txt}"
if [ ! -f "$AGE_KEY" ]; then
    log_warning "age identity key not found at $AGE_KEY"
    log_warning "  Encrypted secrets will deploy but won't decrypt on demand."
    log_warning "  To enable: place your age identity at \$HOME/.config/age/key.txt"
    log_warning "  (or set AGE_KEY_PATH). Generate: age-keygen -o ~/.config/age/key.txt"
    log_warning "  See: docs/runbooks/secrets-management.md"
fi

ensure_directory "$DOTFILES_DIR/sensitive"
if [ "$CURRENT_DIR" != "$DOTFILES_DIR" ]; then
    cp -rf "$CURRENT_DIR/sensitive/"* "$DOTFILES_DIR/sensitive/" 2>/dev/null || true
fi

# Deploy the secrets registry (ADR-028 §2 mapping SSOT). dotf secrets reads it
# from $DOTFILES_DIR/secrets/registry.yaml; without it `dotf secrets {ls,show,run}`
# and the AI-CLI wrappers fail. Mirrors the sensitive/ deploy just above.
ensure_directory "$DOTFILES_DIR/secrets"
if [ "$CURRENT_DIR" != "$DOTFILES_DIR" ]; then
    cp -rf "$CURRENT_DIR/secrets/"* "$DOTFILES_DIR/secrets/" 2>/dev/null || true
fi

# Deploy-time secrets are fetched via `dotf secrets show` after dotf is installed
# (see the block right after install_dotf) -- the load-secrets eager-source was
# retired here (ADR-028 / #587).


# utils.sh is sourced declaratively from .zsh/functions.sh (loaded by both
# ~/.bashrc and ~/.zshrc), so setup no longer mutates the deployed functions.zsh
# to append a `. utils.sh` line. That deploy-time mutation made the deployed copy
# drift from the repo source, so the check_deployed assertion below failed. One
# shared entrypoint gives bash AND zsh the library without ever editing a
# deployed file (REFACTOR-010 / "presence is not convergence").

# Create a bash_aliases file for bash
log_info "Creating bash_aliases file..."
ensure_directory "$HOME/.bash"
cat > "$HOME/.bash/bash_aliases" << EOF
# Bash Aliases
$(grep -v '^\s*#' "$DOTFILES_DIR/.zsh/aliases.zsh" | grep "^alias ")
EOF

# Create a basic .bashrc if it doesn't exist
if [ -f "$DOTFILES_DIR/.bashrc" ]; then
    log_info "Using existing .bashrc from dotfiles..."
else
    log_info "Creating a basic .bashrc file (none found in source)..."
    cat > "$DOTFILES_DIR/.bashrc" << EOF
# ~/.bashrc: executed by bash(1) for non-login shells.

# Source aliases
if [ -f ~/.bash/bash_aliases ]; then
    . ~/.bash/bash_aliases
fi

# Source utility functions
if [ -f "$DOTFILES_DIR/scripts/utils.sh" ]; then
    . "$DOTFILES_DIR/scripts/utils.sh"
fi
EOF
fi

# Deploy .bashrc
deploy_file "$DOTFILES_DIR/.bashrc" "$HOME/.bashrc"

# ============================================================================
# DEVELOPER TOOLS (user-level installs to ~/.local/bin)
# ============================================================================

log_info "Installing developer tools..."
ensure_directory "$HOME/.local/bin"
export PATH="$HOME/.local/bin:$PATH"

# tmux (system package — this script avoids sudo, so user installs it once)
if ! command -v tmux >/dev/null 2>&1; then
    log_warning "tmux not installed. Run: sudo apt install -y tmux"
else
    log_info "tmux installed: $(tmux -V)"
fi

# xclip (X11 clipboard bridge — required for tmux mouse-copy to system clipboard)
if ! command -v xclip >/dev/null 2>&1; then
    log_warning "xclip not installed. Run: sudo apt install -y xclip  (needed for tmux clipboard integration on X11)"
else
    log_info "xclip installed: $(xclip -version 2>&1 | head -n1)"
fi

# age (file encryption — required by secrets system)
if ! command -v age >/dev/null 2>&1; then
    log_info "Installing age..."
    curl -Lo /tmp/age.tar.gz "https://dl.filippo.io/age/latest?for=linux/amd64" 2>/dev/null \
        && tar xzf /tmp/age.tar.gz -C /tmp \
        && cp /tmp/age/age /tmp/age/age-keygen "$HOME/.local/bin/" \
        && rm -rf /tmp/age.tar.gz /tmp/age \
        && log_success "age installed" \
        || log_warning "age installation failed"
else
    log_info "age already installed"
fi

# eza (modern ls replacement)
if ! command -v eza >/dev/null 2>&1; then
    log_info "Installing eza..."
    curl -Lo /tmp/eza.tar.gz "https://github.com/eza-community/eza/releases/latest/download/eza_x86_64-unknown-linux-gnu.tar.gz" 2>/dev/null \
        && tar xzf /tmp/eza.tar.gz -C "$HOME/.local/bin/" \
        && chmod +x "$HOME/.local/bin/eza" \
        && rm -f /tmp/eza.tar.gz \
        && log_success "eza installed" \
        || log_warning "eza installation failed"
else
    log_info "eza already installed"
fi

# jq (JSON processor — required by Claude hook registration)
if ! command -v jq >/dev/null 2>&1; then
    log_info "Installing jq..."
    curl -Lo "$HOME/.local/bin/jq" "https://github.com/jqlang/jq/releases/latest/download/jq-linux-amd64" 2>/dev/null \
        && chmod +x "$HOME/.local/bin/jq" \
        && log_success "jq installed" \
        || log_warning "jq installation failed"
else
    log_info "jq already installed"
fi

# gh (GitHub CLI — required by Copilot setup)
if ! command -v gh >/dev/null 2>&1; then
    log_info "Installing GitHub CLI..."
    GH_VERSION=$(curl -sI "https://github.com/cli/cli/releases/latest" 2>/dev/null | grep -i '^location:' | sed 's|.*/v||;s/[[:space:]]*$//')
    if [ -n "$GH_VERSION" ]; then
        curl -Lo /tmp/gh.tar.gz "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz" 2>/dev/null \
            && tar xzf /tmp/gh.tar.gz -C /tmp \
            && cp "/tmp/gh_${GH_VERSION}_linux_amd64/bin/gh" "$HOME/.local/bin/" \
            && rm -rf /tmp/gh.tar.gz "/tmp/gh_${GH_VERSION}_linux_amd64" \
            && log_success "GitHub CLI installed" \
            || log_warning "GitHub CLI installation failed"
    else
        log_warning "Could not determine gh version, skipping"
    fi
else
    log_info "gh already installed"
fi

# dotf (the dotfiles Go CLI — ADR-020). Fetch the pinned release binary,
# checksum-verified, into ~/.local/bin. Idempotent; converges on drift.
if [ -f ./scripts/install-dotf.sh ]; then
    # shellcheck source=/dev/null
    . ./scripts/install-dotf.sh
    install_dotf || log_warning "dotf installation failed (continuing; see healthcheck)"
else
    log_warning "scripts/install-dotf.sh not found; skipping dotf install"
fi

# Deploy-time secret for the agy MCP config: agy does NOT expand env vars inside
# JSON, so OPENROUTER_API_KEY must be baked into mcp_config.json at deploy. The
# agy block below reads it from the environment; fetch it via the `dotf secrets`
# facade (ADR-028) now that dotf + the registry/store are deployed above, into
# THIS one-shot setup process only -- never the user's interactive shell (that
# export was retired in #581). opencode/pi `{env:NAN_API_KEY}` is resolved
# independently by `dotf secrets render` at deploy time (and opencode/pi's own
# runtime resolver as fallback), which age-decrypt directly.
if command -v dotf >/dev/null 2>&1; then
    if ! OPENROUTER_API_KEY="$(dotf secrets show OPENROUTER_API_KEY 2>&1)"; then
        log_warning "dotf secrets show OPENROUTER_API_KEY failed: $OPENROUTER_API_KEY"
        OPENROUTER_API_KEY=""
    fi
    export OPENROUTER_API_KEY
fi

# Catalog tools (CLI-029): download + checksum-verify the declarative packages.json
# tools (currently sops) into ~/.local/bin via dotf — the same deterministic pattern
# as install_dotf, driven by data instead of a per-OS install block. Best-effort:
# an offline box or a single failed download must never abort setup (parity with the
# `install_dotf || log_warning` line above). Needs dotf on PATH (installed just
# above; ~/.local/bin was exported into PATH earlier in this script).
if command -v dotf >/dev/null 2>&1; then
    dotf tools install || log_warning "dotf tools install failed (continuing; re-run 'dotf tools install')"
fi

# GUARD-001 memory-sink dispatcher (#398/#418): deploy git-hooks/ into the
# ~/.dotfiles mirror and wire core.hooksPath machine-wide so the guard is active
# in every repo. Idempotent; preserves an unrelated pre-existing hooksPath. The
# deployed dotf release can't self-deploy these (no source tree), so the
# bootstrap places them (ADR-020 C7); `dotf doctor` verifies the wiring after.
if [ -f ./scripts/install-git-hooks.sh ]; then
    # shellcheck source=/dev/null
    . ./scripts/install-git-hooks.sh
    install_git_hooks || log_warning "git-hooks install failed (continuing; see 'dotf doctor')"
else
    log_warning "scripts/install-git-hooks.sh not found; skipping memory-sink guard install"
fi

# zoxide (smarter cd)
if ! command -v zoxide >/dev/null 2>&1; then
    log_info "Installing zoxide..."
    curl -sSfL https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/install.sh | sh 2>/dev/null \
        && log_success "zoxide installed" \
        || log_warning "zoxide installation failed"
else
    log_info "zoxide already installed"
fi

# direnv (per-directory environment variables)
if ! command -v direnv >/dev/null 2>&1; then
    log_info "Installing direnv..."
    curl -sfL https://direnv.net/install.sh | bin_path="$HOME/.local/bin" bash 2>/dev/null \
        && log_success "direnv installed" \
        || log_warning "direnv installation failed"
else
    log_info "direnv already installed"
fi

# ShellCheck (shell script linter)
if ! command -v shellcheck >/dev/null 2>&1; then
    log_info "Installing shellcheck..."
    # Versioned asset (the `-stable` alias 404s post-v0.10) + `-f` so an HTTP error
    # fails the curl loudly instead of saving the 404 body as a bogus "tarball" that
    # only blows up later at xz. The tarball's internal dir is shellcheck-v<ver>/.
    _sc_ver="v${SHELLCHECK_VERSION:-0.11.0}"
    curl -fsSLo /tmp/shellcheck.tar.xz "https://github.com/koalaman/ShellCheck/releases/download/${_sc_ver}/shellcheck-${_sc_ver}.linux.x86_64.tar.xz" \
        && tar xJf /tmp/shellcheck.tar.xz -C /tmp \
        && cp "/tmp/shellcheck-${_sc_ver}/shellcheck" "$HOME/.local/bin/" \
        && rm -rf /tmp/shellcheck.tar.xz "/tmp/shellcheck-${_sc_ver}" \
        && log_success "shellcheck installed" \
        || log_warning "shellcheck installation failed"
else
    log_info "shellcheck already installed"
fi

# bats (Bash Automated Testing System)
if ! command -v bats >/dev/null 2>&1; then
    log_info "Installing bats..."
    git clone --depth 1 https://github.com/bats-core/bats-core.git /tmp/bats-core 2>/dev/null \
        && /tmp/bats-core/install.sh "$HOME/.local" 2>/dev/null \
        && rm -rf /tmp/bats-core \
        && log_success "bats installed" \
        || log_warning "bats installation failed"
else
    log_info "bats already installed"
fi

# Antigravity CLI (agy) install — idempotent per pattern-setup-script-idempotence.
# Official install URL: https://antigravity.google/cli/install.sh
# If agy is not in PATH, install it automatically; otherwise skip.
if ! command -v agy >/dev/null 2>&1; then
    log_info "Installing Antigravity CLI (agy)..."
    if curl -fsSL https://antigravity.google/cli/install.sh | bash 2>/dev/null; then
        log_success "agy installed"
    else
        log_warning "agy install failed — re-run setup or install manually (https://antigravity.google)"
    fi
else
    log_info "agy already installed"
fi

# Setup Antigravity CLI configuration (fresh-install model, SDD-007).
# Strategy: write canonical config to where agy reads from (~/.gemini/config/),
# use deploy_file (atomic + idempotent), no symlinks. Closes #100.
log_info "Setting up Antigravity CLI configuration..."
export GEMINI_HOME="$HOME/.gemini"
export AGY_APP_DATA="$GEMINI_HOME/antigravity-cli"
# NOTE: ANTIGRAVITY_ENDPOINT / CLOUDCODE_URL kept for backward compat. Empirical
# finding (cli log 2026-05-25): agy 1.0.2 issues `loadCodeAssist` and
# `fetchAvailableModels` calls to `daily-cloudcode-pa.googleapis.com` regardless
# of these env vars — appears to be hardcoded metadata-plane routing. The chat
# completion itself goes through a separate gRPC channel whose endpoint is not
# clearly env-overridable. Setting these has no observed effect on routing.
export ANTIGRAVITY_ENDPOINT="https://cloudcode-pa.googleapis.com"
export CLOUDCODE_URL="https://cloudcode-pa.googleapis.com"

ensure_directory "$GEMINI_HOME"
ensure_directory "$GEMINI_HOME/config"
ensure_directory "$GEMINI_HOME/prompts"
ensure_directory "$AGY_APP_DATA"

# 1. Deploy agy settings.json + ignore file
if [ -f "$CURRENT_DIR/ai/agy/settings.json" ]; then
    deploy_file "$CURRENT_DIR/ai/agy/settings.json" "$AGY_APP_DATA/settings.json"
fi
if [ -f "$CURRENT_DIR/.geminiignore" ]; then
    deploy_file "$CURRENT_DIR/.geminiignore" "$GEMINI_HOME/.geminiignore"
fi

# 2. Consolidate MCP servers — master at ~/.gemini/config/mcp_config.json (agy's canonical read path)
if [ -f "$CURRENT_DIR/mcp-servers.json" ] && command -v jq >/dev/null 2>&1; then
    log_info "Consolidating Antigravity MCP servers..."

    # Recover OpenRouter key: env → existing config → secrets vault.
    # NOTE: canonical agy schema uses `mcpServers` (not `servers`) per Antigravity docs.
    OLD_KEY="${OPENROUTER_API_KEY:-}"
    if [ -z "$OLD_KEY" ] || [ "$OLD_KEY" = "null" ]; then
        # `|| true` keeps the assignment safe under set -e even when grep
        # returns 1 (no matches) or jq fails on a missing config file
        # (fresh-install path on CI containers without the master MCP yet).
        OLD_KEY=$(jq -r '.mcpServers["hive-vault"].env.OPENROUTER_API_KEY // empty' "$GEMINI_HOME/config/mcp_config.json" 2>/dev/null | grep -v "null" || true)
    fi
    if [ -z "$OLD_KEY" ] || [ "$OLD_KEY" = "null" ] || [ "$OLD_KEY" = '${OPENROUTER_API_KEY}' ]; then
        # Last resort: fetch straight from the dotf secrets facade (ADR-028).
        OLD_KEY="$(dotf secrets show OPENROUTER_API_KEY 2>/dev/null || true)"
    fi

    # Substitute ${VAULT_PATH} placeholder with the canonical Linux vault dir.
    # Committed JSON uses a placeholder so the same source works cross-OS;
    # agy does NOT expand env vars inside JSON values, so the substitution
    # must happen here at write time. Path is by convention: $HOME/Projects/knowledge.
    VAULT_PATH_DEFAULT="${VAULT_PATH:-$HOME/Projects/knowledge}"

    # Preflight: WARN (non-fatal) if the vault dir is missing. hive-vault MCP
    # will fail at first tool call without it, but encrypted secrets + AGY.md +
    # everything else still deploys cleanly so creating the vault later doesn't
    # require a setup re-run. Mirrors the age-key preflight in #106.
    if [ ! -d "$VAULT_PATH_DEFAULT" ]; then
        log_warning "Obsidian vault not found at $VAULT_PATH_DEFAULT"
        log_warning "  hive-vault MCP will error at runtime until this dir exists."
        log_warning "  Either clone/sync your vault to $VAULT_PATH_DEFAULT, or override via"
        log_warning "  the VAULT_PATH env var in your shell."
    fi

    NEW_MCP_CONFIG=$(jq --arg key "$OLD_KEY" --arg vault "$VAULT_PATH_DEFAULT" '
        .mcpServers["hive-vault"].env.OPENROUTER_API_KEY = (if $key != "" and $key != "null" then $key else .mcpServers["hive-vault"].env.OPENROUTER_API_KEY end)
        | .mcpServers["hive-vault"].env.VAULT_PATH = $vault
    ' "$CURRENT_DIR/ai/agy/mcp_servers.json")

    # Merge stdio servers from root mcp-servers.json into mcpServers map
    while IFS=$'\t' read -r name transport args; do
        [ "$name" = "hive" ] && continue
        [ "$transport" != "stdio" ] && continue

        cmd=$(echo "$args" | awk '{print $1}')
        cmd_args=$(echo "$args" | cut -d' ' -f2-)

        NEW_MCP_CONFIG=$(echo "$NEW_MCP_CONFIG" | jq --arg name "$name" --arg cmd "$cmd" --argjson args "$(echo "$cmd_args" | jq -R 'split(" ")')" \
            '.mcpServers[$name] = { "command": $cmd, "args": $args }')
    done < <(jq -r '.servers[] | [.name, .transport, .args] | @tsv' "$CURRENT_DIR/mcp-servers.json")

    # Write master config to agy's canonical read path. No symlinks (BUG-100).
    # Idempotent: write to tempfile then deploy_file (skips when content matches).
    MASTER_CONFIG="$GEMINI_HOME/config/mcp_config.json"
    [ -L "$MASTER_CONFIG" ] && rm -f "$MASTER_CONFIG"
    _mcp_tmp="$(mktemp)"
    echo "$NEW_MCP_CONFIG" > "$_mcp_tmp"
    deploy_file "$_mcp_tmp" "$MASTER_CONFIG"
    rm -f "$_mcp_tmp"

    # Hive plugin: discovery file at canonical plugin path
    HIVE_PLUGIN_DIR="$AGY_APP_DATA/plugins/hive-vault"
    ensure_directory "$HIVE_PLUGIN_DIR"
    echo '{"name": "hive-vault"}' > "$HIVE_PLUGIN_DIR/plugin.json"
    echo "$NEW_MCP_CONFIG" | jq '{ "mcpServers": { "hive-vault": .mcpServers["hive-vault"] } }' > "$HIVE_PLUGIN_DIR/mcp_config.json"

    log_success "Consolidated Antigravity MCP configuration and registered Hive plugin"
fi

# Drop project-local agy state cache from the workspace if agy leaked one
[ -d "$CURRENT_DIR/.antigravitycli" ] && rm -rf "$CURRENT_DIR/.antigravitycli"
[ -d "$CURRENT_DIR/.antigravitycli.bak" ] && rm -rf "$CURRENT_DIR/.antigravitycli.bak"
rm -f "$GEMINI_HOME/config/.migrated" 2>/dev/null

# SDD-007 one-time migration: drop orphan master at the pre-SDD-007 path
# (~/.gemini/mcp_config.json). Canonical path is now ~/.gemini/config/mcp_config.json.
if [ -f "$GEMINI_HOME/mcp_config.json" ] && [ ! -L "$GEMINI_HOME/mcp_config.json" ]; then
    log_info "Removing orphan master from pre-SDD-007 path: $GEMINI_HOME/mcp_config.json"
    rm -f "$GEMINI_HOME/mcp_config.json"
fi
# Drop the old AGY_APP_DATA symlink/copy of mcp_config.json (replaced by canonical path)
[ -e "$AGY_APP_DATA/mcp_config.json" ] && rm -f "$AGY_APP_DATA/mcp_config.json"

# Agy skills (native Shared skills + flat prompts) are deployed from the vault
# skill records by `compile-harness.sh --deploy` (SDD-008), which renders each
# committed record under harness/skills/ to ~/.gemini/skills/<n>/ and a
# frontmatter-stripped flat prompt to ~/.gemini/prompts/<n>.md, honoring each
# skill's targets[]. The single --deploy call near the end of this script does
# this for every agent at once; no per-agent skill loop here.
ensure_directory "$HOME/.gemini/skills"

# SDD-007 one-time migration: gemini-cli -> agy. Remove the legacy
# ~/.gemini/GEMINI.md identity file so it doesn't linger as an orphan
# pointing to the retired binary. Safe to repeat (no-op if absent).
if [ -f "$GEMINI_HOME/GEMINI.md" ]; then
    rm -f "$GEMINI_HOME/GEMINI.md"
    log_info "Removed legacy GEMINI.md (SDD-007 migration: agy replaces gemini-cli)"
fi

# Force copy master files (Neural Hive Protocol)
rm -f "$GEMINI_HOME/AGY.md"
cp "$CURRENT_DIR/ai/agy/AGY.md" "$GEMINI_HOME/AGY.md"
if grep -q 'First, read `AGENTS.md`' "$GEMINI_HOME/AGY.md"; then
    log_success "AGY.md deployed successfully (verified pointer to AGENTS.md)"
else
    echo "❌ Error: AGY.md deployment failed verification"
fi

# Note: legacy `agy plugin import gemini` removed (SDD-007). Fresh-install model:
# we don't carry over from legacy gemini-cli on every setup run. If the user
# needs a one-time migration, they run it manually once.

log_success "Antigravity CLI configuration complete"

# Harness deploy engine (ENGINE-001 / HARNESS-001): re-render the generated
# "Overrides of Harness Defaults" blocks in AGENTS.md + ai/claude/CLAUDE.md from
# the vault SSOT before deploying agent configs. Committed blocks are the fallback
# when the vault is absent (fresh machine), so this only re-renders when possible.
if [ -d "${VAULT_PATH:-$HOME/Projects/knowledge}/00_meta/patterns" ]; then
    if ( cd "$CURRENT_DIR" && "$CURRENT_DIR/scripts/compile-harness.sh" --refresh ) >/dev/null 2>&1; then
        log_success "Harness override blocks refreshed from vault SSOT"
        # OPS-003: --refresh regenerates the committed harness records (and the
        # generated blocks in AGENTS.md / ai/claude/CLAUDE.md) from the vault SSOT
        # -- generate-and-commit (ADR-013). When the vault is ahead, that leaves
        # UNCOMMITTED changes in the working tree. Announce them LOUDLY so they read
        # as an actionable commit, not a parallel session's WIP (the silent drift
        # misled a session, #295). Only meaningful inside the git repo -- the
        # ~/.dotfiles deploy mirror is not a repo (ADR-005).
        if git -C "$CURRENT_DIR" rev-parse --git-dir >/dev/null 2>&1; then
            harness_drift="$(git -C "$CURRENT_DIR" status --porcelain -- harness/ AGENTS.md ai/claude/CLAUDE.md 2>/dev/null)"
            if [ -n "$harness_drift" ]; then
                log_warning "Harness records changed by --refresh from the vault -- commit them:"
                printf '%s\n' "$harness_drift" | sed 's/^/      /'
                log_warning "  git add harness/ AGENTS.md ai/claude/CLAUDE.md && git commit -m 'chore(harness): refresh records from vault'"
            fi
        fi
    else
        log_warning "compile-harness --refresh failed; deploying committed blocks"
    fi
else
    log_info "Vault absent; deploying committed harness override blocks"
fi

# Mirror the harness inputs into the deploy dir so `compile-harness.sh --check`
# (healthcheck section 12) can verify drift offline from ~/.dotfiles. The engine
# resolves its root from its own location, and the rootresolve regression test
# models exactly this complete non-git copy: scripts/ (compile-harness.sh) +
# AGENTS.md + ai/claude/CLAUDE.md + harness/. setup copied none of the latter
# three, so --check exited 2 (manifest not found) and section 12 reported a false
# drift FAIL. This runs AFTER --refresh so the snapshot matches the refreshed
# repo state (else the repo<->deploy drift sub-check would then see drift).
if [ "$CURRENT_DIR" != "$DOTFILES_DIR" ]; then
    ensure_directory "$DOTFILES_DIR/harness"
    cp -rf "$CURRENT_DIR/harness/." "$DOTFILES_DIR/harness/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/AGENTS.md" "$DOTFILES_DIR/" 2>/dev/null || true
    ensure_directory "$DOTFILES_DIR/ai/claude"
    safe_copy "$CURRENT_DIR/ai/claude/CLAUDE.md" "$DOTFILES_DIR/ai/claude/" 2>/dev/null || true
fi

# Claude Code
ensure_directory "$HOME/.claude"
ensure_directory "$HOME/.claude/skills"
# Bulk copy ai/claude/* EXCEPT settings.json (SDD-002: handled by
# merge_claude_settings below, which substitutes __HOOK_COMMAND__ and applies
# the per-key merge policy preserving user customizations).
for _claude_src in "$CURRENT_DIR/ai/claude/"*; do
    [ "$(basename "$_claude_src")" = "settings.json" ] && continue
    cp -rf "$_claude_src" "$HOME/.claude/" 2>/dev/null || true
done
unset _claude_src
# CLI-014: init-project.sh was retired (repo scaffolding is now `dotf init`).
# Remove any copy a prior setup left in ~/.claude so it doesn't linger as an orphan.
rm -f "$HOME/.claude/init-project.sh"
# Claude skills are deployed from the vault skill records by
# `compile-harness.sh --deploy` (SDD-008): each committed record under
# harness/skills/ is rendered (with provenance) to ~/.claude/skills/<n>/,
# de-symlinking any pre-existing vault symlink first (BUG-100). The single
# --deploy call near the end of this script handles every agent at once.
# Force copy master files (Neural Hive Protocol)
rm -f "$HOME/.claude/CLAUDE.md"
cp "$CURRENT_DIR/ai/claude/CLAUDE.md" "$HOME/.claude/CLAUDE.md"
if grep -q 'First, read `AGENTS.md`' "$HOME/.claude/CLAUDE.md"; then
    log_success "CLAUDE.md deployed successfully (verified pointer to AGENTS.md)"
else
    echo "❌ Error: CLAUDE.md deployment failed verification"
fi
log_success "Claude Code configured with skills"

# Python tooling (uv + poetry) — used by hive MCP server (uvx hive-vault) and general Python workflows
# Install uv (Python package manager — provides uvx)
if ! command -v uv >/dev/null 2>&1; then
    log_info "Installing uv..."
    curl -LsSf https://astral.sh/uv/install.sh | sh 2>/dev/null || true
    export PATH="$HOME/.local/bin:$PATH"
    if command -v uv >/dev/null 2>&1; then
        log_success "uv installed"
    else
        log_warning "uv installation failed"
    fi
else
    log_info "uv already installed"
fi

# Install poetry via uv
if ! command -v poetry >/dev/null 2>&1; then
    if command -v uv >/dev/null 2>&1; then
        log_info "Installing poetry via uv..."
        uv tool install poetry 2>/dev/null || true
        if command -v poetry >/dev/null 2>&1; then
            log_success "Poetry installed"
        else
            log_warning "Poetry installation failed"
        fi
    else
        log_warning "uv not available, skipping poetry installation"
    fi
else
    log_info "poetry already installed"
fi

# Create convenience symlinks for versioned-only binaries
PYTHON_DIR=$(ls -d "$HOME/Applications"/python-* 2>/dev/null | sort -V | tail -1) || true
if [ -n "$PYTHON_DIR" ] && [ -d "$PYTHON_DIR/bin" ] && [ ! -e "$PYTHON_DIR/bin/python" ]; then
    log_info "Creating python symlink..."
    ln -s python3 "$PYTHON_DIR/bin/python"
    log_success "python -> python3 symlink created"
fi

# BUG-013b: install obsidian-cli via npm global. Mirror of the
# Windows side shipped in PR #77. The CLI provides the `obsidian` binary
# used by obs-cli.sh and the vault-health workflow. Idempotent: skip when
# `obsidian` is already on PATH. Gated on `npm` so machines without
# Node.js gracefully skip with a clear hint.
# Note: package was previously @vorillaz/obsidian-cli (404), fixed to obsidian-cli.
if ! command -v obsidian >/dev/null 2>&1; then
    if command -v npm >/dev/null 2>&1; then
        log_info "Installing Obsidian CLI (obsidian-cli) via npm..."
        if npm install -g 'obsidian-cli' >/dev/null 2>&1; then
            log_success "Obsidian CLI installed"
        else
            log_warning "Failed to install Obsidian CLI (npm install -g exited non-zero)"
        fi
    else
        log_warning "npm not available, skipping Obsidian CLI install (install Node.js then re-run)"
    fi
else
    log_info "Obsidian CLI already installed at $(command -v obsidian)"
fi

# OpenCode (secondary AI coding agent — see ADR-009 and specs/AI-011-opencode-bootstrap)
# Idempotent per pattern-setup-script-idempotence:
#   - install only if absent (no forced re-install on every run)
#   - reconcile config (not skip-if-exists) so source-of-truth wins on drift
#   - no silenced errors; failures surface as log_warning
log_info "Setting up OpenCode configuration..."
OPENCODE_BIN_DIR="$HOME/.opencode/bin"
OPENCODE_BINARY="$OPENCODE_BIN_DIR/opencode"

if ! command -v opencode >/dev/null 2>&1 && [ ! -x "$OPENCODE_BINARY" ]; then
    # Pin to $OPENCODE_VERSION from versions.conf (REFACTOR-011). The official
    # installer honors `--version`. If the manifest is missing (broken checkout)
    # OPENCODE_VERSION is empty -- fall back to latest rather than passing an
    # empty --version, and let healthcheck surface the drift.
    opencode_install_status=0
    if [ -n "$OPENCODE_VERSION" ]; then
        log_info "Installing opencode v$OPENCODE_VERSION via official install script..."
        curl -fsSL https://opencode.ai/install | bash -s -- --version "$OPENCODE_VERSION" || opencode_install_status=$?
    else
        log_info "Installing opencode (latest; OPENCODE_VERSION unset) via official install script..."
        curl -fsSL https://opencode.ai/install | bash || opencode_install_status=$?
    fi
    if [ "$opencode_install_status" -eq 0 ]; then
        log_success "opencode installed to $OPENCODE_BIN_DIR"
    else
        log_warning "opencode install failed — re-run setup or install manually (https://opencode.ai/docs/)"
    fi
elif [ -n "$OPENCODE_VERSION" ]; then
    # REFACTOR-011/013: presence is not convergence, and the pin is a MINIMUM.
    # An opencode OLDER than the pin was previously skipped as "already
    # installed"; an exact-match reconcile would conversely DOWNGRADE a newer
    # install. Upgrade only when installed < pin (version_gte), never downgrade.
    opencode_cmd="opencode"
    command -v opencode >/dev/null 2>&1 || opencode_cmd="$OPENCODE_BINARY"
    installed_opencode=$("$opencode_cmd" --version 2>/dev/null | head -1 | awk '{print $NF}')
    if ! version_gte "$installed_opencode" "$OPENCODE_VERSION"; then
        log_info "opencode ${installed_opencode:-unknown} below pinned minimum $OPENCODE_VERSION, upgrading..."
        curl -fsSL https://opencode.ai/install | bash -s -- --version "$OPENCODE_VERSION" || true
        # Re-query instead of trusting the installer exit code: when the
        # PATH-resolved binary comes from another package manager (npm
        # global), the installer converges its own copy but the shadowing
        # install keeps winning -- that is not convergence.
        converged_opencode=$("$opencode_cmd" --version 2>/dev/null | head -1 | awk '{print $NF}')
        if version_gte "$converged_opencode" "$OPENCODE_VERSION"; then
            log_success "opencode upgraded to ${converged_opencode:-$OPENCODE_VERSION} (>= $OPENCODE_VERSION)"
        else
            log_warning "opencode still ${converged_opencode:-unknown} after install of $OPENCODE_VERSION: another install shadows it in PATH (e.g. npm global); converge that install instead"
        fi
    else
        log_info "opencode already installed (${installed_opencode:-unknown} >= $OPENCODE_VERSION)"
    fi
else
    log_info "opencode already installed"
fi

# Deploy opencode.jsonc with deploy-time {env:VAR} substitution (SDD-009).
# Source ships placeholders like {env:NAN_API_KEY}; we substitute the literal
# age-decrypted value at deploy time so the deployed config is self-contained
# (no runtime env-var propagation needed when opencode launches from a
# non-shell parent). Placeholders without a resolvable mapping are left intact
# and opencode's runtime resolver acts as fallback.
ensure_directory "$HOME/.config/opencode"
OPENCODE_CONFIG_SRC="$CURRENT_DIR/ai/opencode/opencode.jsonc"
OPENCODE_CONFIG_DST="$HOME/.config/opencode/opencode.jsonc"
if [ -f "$OPENCODE_CONFIG_SRC" ]; then
    OPENCODE_CONFIG_TMP=$(mktemp)
    cp "$OPENCODE_CONFIG_SRC" "$OPENCODE_CONFIG_TMP"
    # Deploy-time {env:VAR} materialization via the dotf CLI (over secrets/registry.yaml,
    # ADR-020/ADR-028). Gate on the subcommand SUCCEEDING, not just dotf's presence: a
    # stale dotf passes `command -v` but fails `secrets render`, and under set -e that
    # would abort setup. Running it in the `if` condition exempts it from set -e; if it
    # fails, the {env:VAR} placeholders are left intact for opencode's runtime resolver.
    if command -v dotf >/dev/null 2>&1 && dotf secrets render "$OPENCODE_CONFIG_TMP"; then
        : # materialized via dotf secrets render
    else
        log_warning "dotf secrets render unavailable; opencode.jsonc deployed with literal {env:VAR} placeholders (resolved at runtime)"
    fi
    mv "$OPENCODE_CONFIG_TMP" "$OPENCODE_CONFIG_DST"
    log_success "Deployed opencode.jsonc (deploy-time secrets) to $OPENCODE_CONFIG_DST"
else
    log_warning "opencode.jsonc source missing: $OPENCODE_CONFIG_SRC"
fi

# Deploy the canonical AGENTS.md as opencode's global system prompt.
# OpenCode reads ~/.config/opencode/AGENTS.md (per upstream docs); unlike
# claude/agy/copilot which use pointer files, opencode reads the filename
# "AGENTS.md" natively so we copy the full SSOT (~22KB) verbatim.
AGENTS_SRC="$CURRENT_DIR/AGENTS.md"
AGENTS_DST="$HOME/.config/opencode/AGENTS.md"
if [ -f "$AGENTS_SRC" ]; then
    if [ -f "$AGENTS_DST" ] && cmp -s "$AGENTS_SRC" "$AGENTS_DST"; then
        log_info "AGENTS.md (opencode) already in sync"
    else
        cp "$AGENTS_SRC" "$AGENTS_DST"
        log_success "Deployed AGENTS.md to $AGENTS_DST"
    fi
else
    log_warning "AGENTS.md source missing at $AGENTS_SRC"
fi

# Yarn (classic) — npm-pinned global install (TERM-002 companion).
# Guarded on npm (same convention as pi). Reconcile-not-skip: a version drift
# is corrected by reinstalling the pin, since npm -g yarn upgrades are safe.
if command -v npm >/dev/null 2>&1; then
    YARN_PINNED="${YARN_VERSION:-}"
    if ! command -v yarn >/dev/null 2>&1; then
        log_info "Installing yarn (yarn@${YARN_PINNED:-latest}) via npm..."
        if npm install -g "yarn${YARN_PINNED:+@$YARN_PINNED}" >/dev/null 2>&1; then
            log_success "yarn installed: $(yarn --version 2>/dev/null)"
        else
            log_warning "yarn install failed — run: npm install -g yarn${YARN_PINNED:+@$YARN_PINNED}"
        fi
    else
        INSTALLED_YARN=$(yarn --version 2>/dev/null | head -1)
        if [ -n "$YARN_PINNED" ] && ! version_gte "$INSTALLED_YARN" "$YARN_PINNED"; then
            log_info "yarn $INSTALLED_YARN below pinned minimum $YARN_PINNED — upgrading..."
            if npm install -g "yarn@$YARN_PINNED" >/dev/null 2>&1; then
                log_success "yarn upgraded to $YARN_PINNED"
            else
                log_warning "yarn upgrade failed — run: npm install -g yarn@$YARN_PINNED"
            fi
        else
            log_info "yarn already installed: $INSTALLED_YARN"
        fi
    fi
else
    log_warning "npm not found — skipping yarn install (install Node.js, then re-run setup)"
fi

# Deploy pi coding agent config (AI-025) — mirrors the opencode block so the two
# agents are interchangeable across Linux/Windows. pi reads ~/.pi/agent/.
# Install pinned via npm (guarded); models.json gets the same deploy-time secret
# substitution as opencode (SDD-009, self-contained cross-OS); AGENTS.md is the
# same canonical SSOT; settings.json is seeded only when absent (pi mutates it).
PI_AGENT_DIR="$HOME/.pi/agent"
PI_BIN="$HOME/.local/bin/pi"
if command -v npm >/dev/null 2>&1; then
    # Install into the manager-independent ~/.local prefix — the same dir that
    # already carries `claude`/`dotf` and is on PATH for login shells AND
    # GUI/ADE processes (e.g. Orca). A bare `npm i -g` lands in nvm's
    # per-node-version tree (~/.nvm/versions/node/<v>/bin), invisible to any
    # environment running a different node. Guard on the stable location, not
    # bare `command -v pi`, so a stale nvm-version copy cannot mask a missing
    # ~/.local launcher.
    if [ ! -x "$PI_BIN" ]; then
        PI_PKG="@earendil-works/pi-coding-agent${PI_VERSION:+@$PI_VERSION}"
        log_info "Installing pi ($PI_PKG) into ~/.local ..."
        if npm install -g --ignore-scripts --prefix "$HOME/.local" "$PI_PKG" >/dev/null 2>&1; then
            log_success "pi installed -> $PI_BIN"
        else
            log_warning "pi install failed — run: npm install -g --ignore-scripts --prefix \"\$HOME/.local\" $PI_PKG"
        fi
    else
        # REFACTOR-013: the pin is a MINIMUM. Presence alone left an outdated pi
        # in place forever (it was only ever WARNed by healthcheck, #474, never
        # converged here). Upgrade when installed < pin; leave a newer pi alone.
        installed_pi=$("$PI_BIN" --version 2>/dev/null | head -1 | awk '{print $NF}')
        if [ -n "$PI_VERSION" ] && ! version_gte "$installed_pi" "$PI_VERSION"; then
            log_info "pi ${installed_pi:-unknown} below pinned minimum $PI_VERSION — upgrading..."
            if npm install -g --ignore-scripts --prefix "$HOME/.local" "@earendil-works/pi-coding-agent@$PI_VERSION" >/dev/null 2>&1; then
                log_success "pi upgraded to $PI_VERSION"
            else
                log_warning "pi upgrade failed — run: npm install -g --ignore-scripts --prefix \"\$HOME/.local\" @earendil-works/pi-coding-agent@$PI_VERSION"
            fi
        else
            log_info "pi already installed ($PI_BIN, ${installed_pi:-unknown} >= ${PI_VERSION:-any})"
        fi
    fi
else
    log_warning "npm not found — skipping pi install (install Node.js, then re-run setup)"
fi

ensure_directory "$PI_AGENT_DIR"

# pi's models.json is deployed by `dotf deploy` (CLI-039): one implementation for
# every OS, replacing the copy that lived here and its twin in setup-windows.ps1.
# ADR-020 C7 keeps this script on the thin bootstrap; staging, rendering,
# comparing and installing a config is tooling logic and belongs in the CLI.
if command -v dotf >/dev/null 2>&1; then
    dotf deploy pi || log_warning "dotf deploy pi failed -- run it again after setup, or see 'dotf doctor'"
else
    log_warning "dotf not on PATH -- skipping pi config deploy (run ./scripts/install-dotf.sh, then 'dotf deploy pi')"
fi

PI_AGENTS_DST="$PI_AGENT_DIR/AGENTS.md"
if [ -f "$AGENTS_SRC" ]; then
    if [ -f "$PI_AGENTS_DST" ] && cmp -s "$AGENTS_SRC" "$PI_AGENTS_DST"; then
        log_info "AGENTS.md (pi) already in sync"
    else
        cp "$AGENTS_SRC" "$PI_AGENTS_DST"
        log_success "Deployed AGENTS.md to $PI_AGENTS_DST"
    fi
fi

# settings.json is SEED-IF-MISSING, unlike models.json/tui.json above: pi
# rewrites this file at runtime (lastChangelogVersion, theme, the model picked
# in the TUI), so it is the one deployed config the user's own tool edits.
# It previously used the same "copy unless identical" shape as its neighbours,
# which for a self-mutating file means copy ALWAYS -- tests/pi-config.bats
# forbids lastChangelogVersion in the committed copy, so `cmp` could never
# match once pi had run, and every setup run silently reset the user's theme
# and default model. Seed it, then never touch it again.
PI_SETTINGS_SRC="$CURRENT_DIR/ai/pi/settings.json"
PI_SETTINGS_DST="$PI_AGENT_DIR/settings.json"
if [ -f "$PI_SETTINGS_SRC" ]; then
    if [ -f "$PI_SETTINGS_DST" ]; then
        log_info "pi settings.json present, preserving local edits"
    else
        cp "$PI_SETTINGS_SRC" "$PI_SETTINGS_DST"
        log_success "Seeded pi settings.json at $PI_SETTINGS_DST"
    fi
fi

# Deploy opencode TUI config (theme + keybinds incl. the display_thinking toggle).
# Plain copy — no secret substitution (DX-004): unlike opencode.jsonc this file
# carries no secrets, so it deploys verbatim. opencode reads tui.json natively.
TUI_SRC="$CURRENT_DIR/ai/opencode/tui.json"
TUI_DST="$HOME/.config/opencode/tui.json"
if [ -f "$TUI_SRC" ]; then
    if [ -f "$TUI_DST" ] && cmp -s "$TUI_SRC" "$TUI_DST"; then
        log_info "opencode tui.json already in sync"
    else
        cp "$TUI_SRC" "$TUI_DST"
        log_success "Deployed tui.json to $TUI_DST"
    fi
else
    log_warning "tui.json source missing at $TUI_SRC"
fi

# opencode commands are deployed from the vault skill records by
# `compile-harness.sh --deploy` (SDD-008): each committed record under
# harness/skills/ whose targets[] includes opencode is rendered to a
# ~/.config/opencode/commands/<n>.md command file (name: dropped — opencode
# keys commands off the filename). This replaces the former
# ai/opencode/commands/ + skills-to-opencode.sh path; the per-skill targets[]
# now expresses what was the hard-coded Claude-only skip-list.

# Post-deploy assertion — binary reachable + version reports
if [ -x "$OPENCODE_BINARY" ] || command -v opencode >/dev/null 2>&1; then
    OPENCODE_VERSION=$("$OPENCODE_BINARY" --version 2>&1 | head -1 || echo "unknown")
    log_success "opencode ready: $OPENCODE_VERSION"
    log_info "First-time use: launch \`opencode\` and run /connect to authenticate (Go subscription)"
else
    log_warning "opencode binary not reachable at $OPENCODE_BINARY after install — agent unavailable"
fi

# GitHub Copilot CLI (BUG-003: standalone agentic CLI, drops legacy gh-copilot
# extension path). Linux side is detect-and-act -- no auto-install (distros vary;
# user installs via snap/apt/curl per https://docs.github.com/copilot/how-tos/copilot-cli).
# Verification string set by AI-013 (pointer-style copilot-instructions.md).
#
# Cleanup (idempotent): the previous setup added 'eval "$(gh copilot alias -- bash)"'
# to .zshrc/.bashrc. That subcommand does not exist in the new standalone CLI, so
# the line errors silently on every shell startup. Strip it if present.
for rc in "$HOME/.zshrc" "$HOME/.bashrc"; do
    if [ -f "$rc" ] && grep -qF 'eval "$(gh copilot alias -- bash)"' "$rc"; then
        sed -i '/eval "$(gh copilot alias -- bash)"/d' "$rc"
        log_info "Removed stale gh-copilot eval line from $rc"
    fi
done

if command -v copilot >/dev/null 2>&1; then
    log_info "GitHub Copilot CLI detected, deploying configuration..."
    ensure_directory "$HOME/.copilot"
    cp -rf "$CURRENT_DIR/ai/copilot/"* "$HOME/.copilot/" 2>/dev/null || true
    if [ -f "$HOME/.copilot/copilot-instructions.md" ] && grep -q 'First, read `AGENTS.md`' "$HOME/.copilot/copilot-instructions.md"; then
        log_success "copilot-instructions.md deployed successfully (verified pointer to AGENTS.md)"
    else
        log_warning "copilot-instructions.md deployment failed verification (expected pointer to AGENTS.md)"
    fi
    log_success "GitHub Copilot CLI configured (aliases cop/cops in .zsh/aliases.zsh)"
else
    log_info "GitHub Copilot CLI not installed, skipping Copilot config (install via snap/apt/curl: https://docs.github.com/copilot/how-tos/copilot-cli)"
fi

# SDD-005 parity (.github/copilot-instructions.md vs ai/copilot/): NOT synced here.
# Setup deploys to $HOME only and MUST NEVER write into the checkout — a checkout
# write leaves `git status` dirty, and `dotf update` skips any dirty worktree,
# so a self-deploying machine silently stops updating after the first run
# (dotfiles#694). The two copilot-instructions files are kept in parity by the
# fail-loud test tests/docs-drift.bats (runs in CI), which is the correct
# enforcement point: drift blocks the merge instead of a deploy rewriting a
# committed file behind the user's back.

# BUG-004 + BUG-011: defense-in-depth wrapper around EVERY `claude <subcommand>`
# invocation in this script. The Claude Code CLI's deserialize-modify-serialize
# cycle drops fields outside its internal struct (organizationType,
# organizationRateLimitTier, projects map, onboarding flags) -- upstream bug
# anthropics/claude-code#59870 -- shrinking ~/.claude/.claude.json from ~75 KB
# to ~1.5 KB and forcing re-authentication. The bug fires on ANY subcommand
# (`plugin install`, `plugin list`, `mcp get`, `mcp add`), not just install.
#
# BUG-004 (PR #57) wrapped only `claude plugin install`. BUG-011 extends the
# guard to every other call site (MCP loop iterations + plugin list pre-fetch)
# after the user empirically observed truncation recurrence in an MCP-only path.
#
# snapshot_claude_json copies the file to a tempfile BEFORE the call;
# restore_claude_json_if_truncated restores it AFTER, iff the snapshot was
# >= 10 KB and the new file is < 50% of the snapshot size. Complementary to
# SDD-021 session-start canary in dotf mem session-start (same 10240-byte
# threshold, same upstream issue). See dotfiles#33 for the original incomplete
# trigger fix that motivated this layer.
snapshot_claude_json() {
    local claude_json="$HOME/.claude/.claude.json"
    [ -f "$claude_json" ] || return 0
    local backup
    backup=$(mktemp "${TMPDIR:-/tmp}/.claude.json.bug004.XXXXXX")
    cp -f "$claude_json" "$backup"
    printf '%s' "$backup"
}

restore_claude_json_if_truncated() {
    local backup="$1"
    [ -n "$backup" ] && [ -f "$backup" ] || return 0
    local claude_json="$HOME/.claude/.claude.json"
    if [ -f "$claude_json" ]; then
        local snapshot_size new_size half
        snapshot_size=$(stat -c %s "$backup" 2>/dev/null || echo 0)
        new_size=$(stat -c %s "$claude_json" 2>/dev/null || echo 0)
        half=$((snapshot_size / 2))
        if [ "$snapshot_size" -ge 10240 ] && [ "$new_size" -lt "$half" ]; then
            cp -f "$backup" "$claude_json"
            log_warning ".claude.json shrunk from $snapshot_size to $new_size bytes after install (upstream #59870); restored from backup"
        fi
    fi
    rm -f "$backup"
}

# Register MCP servers (requires Claude Code CLI, Node.js, jq)
# Idempotent: server list lives in mcp-servers.json (SSOT shared with Windows);
# `claude mcp get` is used to skip already-registered entries, and `add` errors
# are surfaced rather than swallowed. BUG-011: every `claude mcp {get,add}`
# invocation is wrapped with snapshot_claude_json / restore_claude_json_if_truncated
# because both subcommands hit the same #59870 truncation path as `plugin install`.
MCP_CONFIG="$DOTFILES_DIR/mcp-servers.json"
if command -v claude >/dev/null 2>&1 && command -v npx >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
    if [ ! -f "$MCP_CONFIG" ]; then
        log_warning "mcp-servers.json not found at $MCP_CONFIG, skipping MCP registration"
    else
        log_info "Registering Claude Code MCP servers from $MCP_CONFIG..."
        mcp_added=0
        mcp_skipped=0
        mcp_failed=0
        # HIVE-118: migrate a stale `uvx hive-vault` entry by REMOVING it here so
        # the skip-if-present loop below re-adds the current definition
        # (`hive client`) from mcp-servers.json. Remove-only -- the loop owns every
        # `claude mcp add` so this stays SSOT-driven (no hardcoded server add).
        if claude mcp get hive 2>/dev/null | grep -qE 'uvx|hive-vault'; then
            _snap=$(snapshot_claude_json)
            if claude mcp remove hive --scope user >/dev/null 2>&1; then
                log_info "Migrating hive MCP entry: uvx hive-vault -> hive client (via mcp-servers.json)"
            fi
            restore_claude_json_if_truncated "$_snap"
        fi
        while IFS=$'\t' read -r mcp_name mcp_transport mcp_args mcp_prereq_bin mcp_prereq_cmd; do
            # Check prerequisite binary, run prerequisite command if specified
            if [ -n "$mcp_prereq_bin" ]; then
                if ! command -v "$mcp_prereq_bin" >/dev/null 2>&1; then
                    log_warning "MCP $mcp_name: prerequisite '$mcp_prereq_bin' not found, skipping"
                    mcp_failed=$((mcp_failed + 1))
                    continue
                fi
                if [ -n "$mcp_prereq_cmd" ]; then
                    # shellcheck disable=SC2086
                    if ! $mcp_prereq_cmd >/dev/null 2>&1; then
                        log_warning "MCP $mcp_name: prerequisite command failed: $mcp_prereq_cmd"
                    fi
                fi
            fi
            # BUG-011: snapshot before both `mcp get` and `mcp add` (one snapshot
            # per iteration -- legitimate `mcp add` additions are <<50% of file
            # size, so restore only fires on the real #59870 truncation).
            _snap=$(snapshot_claude_json)
            # Idempotence: skip if `claude mcp get` already knows this name
            if claude mcp get "$mcp_name" >/dev/null 2>&1; then
                log_info "MCP $mcp_name already registered, skipping"
                mcp_skipped=$((mcp_skipped + 1))
                restore_claude_json_if_truncated "$_snap"
                continue
            fi
            # Word-split args intentionally; entries in mcp-servers.json are
            # tokenized for `claude mcp add` argv after `--`.
            # shellcheck disable=SC2086
            if mcp_err=$(claude mcp add --transport "$mcp_transport" "$mcp_name" --scope user -- $mcp_args 2>&1); then
                log_success "Registered MCP $mcp_name"
                mcp_added=$((mcp_added + 1))
            else
                log_warning "Failed to register MCP $mcp_name: $mcp_err"
                mcp_failed=$((mcp_failed + 1))
            fi
            restore_claude_json_if_truncated "$_snap"
        done < <(jq -r '.servers[] | [.name, .transport, .args, (.prerequisite_binary // ""), (.prerequisite_command // "")] | @tsv' "$MCP_CONFIG")
        log_success "MCP servers: $mcp_added added, $mcp_skipped already present, $mcp_failed failed"
    fi
else
    log_warning "Claude Code CLI, npx, or jq not found, skipping MCP server registration"
fi

# Phase C daemon supervision (HIVE-118 / hive#176). Install the supervised
# `hive serve` service now that the MCP loop's prerequisite has installed/upgraded
# the tool. Gated on hive-vault >= 1.32.0 via the package version (NOT by probing
# `hive service`, which an older hive routes to the blocking stdio server). The MCP
# entry itself is already `hive client` (re-added from mcp-servers.json above).
# Non-fatal: a missing systemd --user or any failure leaves the in-process fallback.
hive_ver=$(uv tool list 2>/dev/null | sed -n 's/^hive-vault v\?\([0-9][0-9.]*\).*/\1/p' | head -1)
if command -v hive >/dev/null 2>&1 && [ -n "$hive_ver" ] \
    && [ "$(printf '1.32.0\n%s\n' "$hive_ver" | sort -V | head -1)" = "1.32.0" ]; then
    if hive service install >/dev/null 2>&1; then
        log_success "Installed hive daemon service (systemd --user, v$hive_ver)"
    else
        log_warning "hive service install failed (non-fatal; client works via fallback)"
    fi

    # ADR-025 + HARNESS-024 (#446): the hive serve daemon is a systemd --user
    # service and does NOT source the shell paths.sh, so provision its vault path
    # via environment.d (read by the user manager for all --user services). The
    # value follows the ADR-025 cascade ($VAULT_PATH -> machine.json -> contract).
    # hive#246 hardens the daemon's own resolution; this guarantees the env is set.
    HIVE_VAULT_RESOLVED="$(dotf env path HIVE_VAULT_PATH 2>/dev/null || true)"
    HIVE_VAULT_RESOLVED="${HIVE_VAULT_RESOLVED:-${HIVE_VAULT_PATH:-${VAULT_PATH:-$HOME/Projects/knowledge}}}"
    ENV_D_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/environment.d"
    if mkdir -p "$ENV_D_DIR" 2>/dev/null; then
        printf 'HIVE_VAULT_PATH=%s\nVAULT_PATH=%s\n' "$HIVE_VAULT_RESOLVED" "$HIVE_VAULT_RESOLVED" > "$ENV_D_DIR/10-hive-vault.conf"
        export HIVE_VAULT_PATH="$HIVE_VAULT_RESOLVED"
        systemctl --user import-environment HIVE_VAULT_PATH VAULT_PATH >/dev/null 2>&1 || true
        log_success "Provisioned hive daemon vault path via environment.d ($HIVE_VAULT_RESOLVED)"
    fi

    # AI-023 / hive#176: the upgrade policy that FEEDS the daemon's
    # restart-on-upgrade. Deploy the --user timer + oneshot from the repo's
    # systemd/ SSOT and enable the timer (every 15 min). Same version gate as the
    # service above -- daemon + its feeder install together or not at all.
    if command -v systemctl >/dev/null 2>&1; then
        SYSTEMD_USER_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
        ensure_directory "$SYSTEMD_USER_DIR"
        if cp -f "$CURRENT_DIR/systemd/hive-upgrade.service" "$SYSTEMD_USER_DIR/" \
            && cp -f "$CURRENT_DIR/systemd/hive-upgrade.timer" "$SYSTEMD_USER_DIR/"; then
            systemctl --user daemon-reload >/dev/null 2>&1 || true
            if systemctl --user enable --now hive-upgrade.timer >/dev/null 2>&1; then
                log_success "Enabled hive-upgrade.timer (every 15 min, systemd --user)"
                # Single owner: retire the legacy manual cron now the timer owns
                # the upgrade policy (only after a successful enable).
                if crontab -l 2>/dev/null | grep -q 'uv tool upgrade hive-vault'; then
                    if crontab -l 2>/dev/null | grep -v 'uv tool upgrade hive-vault' | crontab -; then
                        log_info "Removed legacy 'uv tool upgrade hive-vault' cron (timer is now the single owner)"
                    fi
                fi
            else
                log_warning "hive-upgrade.timer enable failed (non-fatal; setup still upgrades hive each run)"
            fi
        else
            log_warning "Could not deploy hive-upgrade units to $SYSTEMD_USER_DIR (non-fatal)"
        fi
    else
        log_warning "systemctl not found; skipping hive-upgrade.timer (setup still upgrades hive each run)"
    fi
elif command -v hive >/dev/null 2>&1; then
    log_warning "hive ${hive_ver:-<unknown>} predates 'hive service' (need >= 1.32.0); skipping daemon supervision"
fi

# OPS-001: opt-in self-deploy timer (git pull --ff-only + idempotent setup).
# Tri-state, gated on DOTFILES_AUTODEPLOY so a normal setup never touches it:
#   1     -> deploy units + enable --now the daily --user timer
#   0     -> disable + remove the timer (clean opt-out / teardown)
#   unset -> no-op (opt-in, default OFF)
# Non-fatal: any failure leaves setup succeeding; the timer is a convenience, and
# `dotf update` itself no-ops on a dirty/diverged repo (OPS-001).
if command -v systemctl >/dev/null 2>&1; then
    SYSTEMD_USER_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
    case "${DOTFILES_AUTODEPLOY:-}" in
        1)
            ensure_directory "$SYSTEMD_USER_DIR"
            if cp -f "$CURRENT_DIR/systemd/dotfiles-selfupdate.service" "$SYSTEMD_USER_DIR/" \
                && cp -f "$CURRENT_DIR/systemd/dotfiles-selfupdate.timer" "$SYSTEMD_USER_DIR/"; then
                systemctl --user daemon-reload >/dev/null 2>&1 || true
                if systemctl --user enable --now dotfiles-selfupdate.timer >/dev/null 2>&1; then
                    log_success "Enabled dotfiles-selfupdate.timer (daily self-deploy, systemd --user)"
                else
                    log_warning "dotfiles-selfupdate.timer enable failed (non-fatal; re-run setup manually to update)"
                fi
            else
                log_warning "Could not deploy dotfiles-selfupdate units to $SYSTEMD_USER_DIR (non-fatal)"
            fi
            ;;
        0)
            systemctl --user disable --now dotfiles-selfupdate.timer >/dev/null 2>&1 || true
            rm -f "$SYSTEMD_USER_DIR/dotfiles-selfupdate.timer" "$SYSTEMD_USER_DIR/dotfiles-selfupdate.service"
            systemctl --user daemon-reload >/dev/null 2>&1 || true
            log_info "Disabled + removed dotfiles-selfupdate.timer (DOTFILES_AUTODEPLOY=0)"
            ;;
        *)
            : # unset -> no-op (opt-in, default OFF)
            ;;
    esac
fi

# Claude Code plugins (requires claude CLI).
# Idempotent: cache the installed-plugins list ONCE before the loop and skip
# entries already present. The wrapper above (BUG-004/011) catches the
# false-negative case where the idempotence guard misses a plugin and the
# resulting `claude plugin install` call truncates .claude.json. BUG-011: the
# pre-loop `claude plugin list` is now also wrapped because it goes through the
# same #59870 path.
if command -v claude >/dev/null 2>&1; then
    log_info "Installing Claude Code plugins..."
    # BUG-011: wrap the read-only `claude plugin list` pre-fetch with the
    # snapshot guard -- the CLI still rewrites .claude.json on any invocation.
    _snap=$(snapshot_claude_json)
    installed_plugins=$(claude plugin list 2>/dev/null || true)
    restore_claude_json_if_truncated "$_snap"
    plugins_added=0
    plugins_skipped=0
    for plugin in \
        "gopls-lsp@claude-plugins-official" \
        "security-guidance@claude-plugins-official" \
        "frontend-design@claude-plugins-official"; do
        if printf '%s' "$installed_plugins" | grep -qF "$plugin"; then
            plugins_skipped=$((plugins_skipped + 1))
        else
            # BUG-004: wrap the install with snapshot/restore so the upstream
            # truncation bug (#59870) cannot drop subscription state.
            _snap=$(snapshot_claude_json)
            if claude plugin install "$plugin" >/dev/null 2>&1; then
                plugins_added=$((plugins_added + 1))
            fi
            restore_claude_json_if_truncated "$_snap"
        fi
    done
    log_success "Claude Code plugins ready ($plugins_added added, $plugins_skipped already present)"
else
    log_warning "Claude Code CLI not found, skipping plugin installation"
fi

# MEM-002: retire claude-mem — one-cycle cleanup, prune after rollout.
# claude-mem (the @thedotmack conversation-memory plugin + its marketplace) is no
# longer installed (ADR-016 Q2: drop the L0 store). Converge existing machines to
# "no claude-mem" on the next setup: uninstall the plugin if the CLI is present,
# then remove any leftover plugin cache + marketplace dirs (both the GitHub repo
# name `thedotmack-claude-mem` and the legacy `thedotmack` fallback). Silent and
# idempotent — a no-op on a clean machine.
log_info "Removing retired claude-mem plugin (MEM-002, if present)..."
if command -v claude >/dev/null 2>&1; then
    _snap=$(snapshot_claude_json)
    claude plugin uninstall claude-mem@thedotmack >/dev/null 2>&1 || true
    restore_claude_json_if_truncated "$_snap"
fi
_claude_cfg="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
rm -rf "$_claude_cfg/plugins/cache/thedotmack/claude-mem" \
       "$_claude_cfg/plugins/marketplaces/thedotmack-claude-mem" \
       "$_claude_cfg/plugins/marketplaces/thedotmack" 2>/dev/null || true
# The dir removal above is undone on the next Claude start if `thedotmack` is still
# registered in settings.json — Claude re-clones the marketplace and its SessionStart
# self-heal hook re-activates claude-mem. The settings merge below is additive (it
# never strips keys), so delete the marketplace registration explicitly here. Guarded
# on jq + an existing target; idempotent (the jq -e test skips an already-clean file).
_claude_settings="$_claude_cfg/settings.json"
if command -v jq >/dev/null 2>&1 && [ -f "$_claude_settings" ] && \
   jq -e '.extraKnownMarketplaces.thedotmack' "$_claude_settings" >/dev/null 2>&1; then
    if _stripped=$(jq 'del(.extraKnownMarketplaces.thedotmack)
          | if (.extraKnownMarketplaces == {}) then del(.extraKnownMarketplaces) else . end' \
          "$_claude_settings" 2>/dev/null) && [ -n "$_stripped" ]; then
        printf '%s\n' "$_stripped" > "$_claude_settings"
        log_info "Removed retired claude-mem marketplace registration from settings.json"
    fi
fi

# Merge `ai/claude/settings.json` template into the deployed `~/.claude/settings.json`
# per the per-key policy in specs/SDD-002-settings-portability/proposal.md. Bootstrap
# when target missing. Preserves user customizations (Read paths,
# additionalDirectories, third-party hooks like GitGuardian) by only
# touching the keys declared as "ours" in the template. The template's
# __HOOK_COMMAND__ placeholder is replaced via jq --arg before any merge / write.
merge_claude_settings() {
    local template_path="$1"
    local target_path="$2"
    local hook_command="$3"
    local session_end_command="$4"

    if [ ! -f "$template_path" ]; then
        log_warning "Claude settings template not found at $template_path, skipping merge"
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        log_warning "jq not found, skipping settings merge (install jq and re-run)"
        return 0
    fi

    local template_substituted
    template_substituted=$(jq --arg cmd "$hook_command" --arg endcmd "$session_end_command" \
        '(.hooks.SessionStart[0].hooks[0].command) = $cmd
         | (.hooks.SessionEnd[0].hooks[0].command) = $endcmd
         | .permissions.allow = (.permissions.allow | unique)' \
        "$template_path" 2>/dev/null)
    if [ -z "$template_substituted" ]; then
        log_warning "Claude settings template substitution failed, skipping merge"
        return 0
    fi

    if [ ! -f "$target_path" ]; then
        log_info "Bootstrapping ~/.claude/settings.json from template (file did not exist)"
        echo "$template_substituted" > "$target_path"
        log_success "Claude settings.json bootstrapped from template"
        return 0
    fi

    # Per-key merge via single jq invocation. Policy table in proposal.md:
    # model, effortLevel: template wins. permissions.allow: UNION (deduped).
    # hooks.SessionStart: template wins (replace). enabledPlugins: object
    # merge (template wins on conflict). All other keys: existing preserved.
    local merged
    merged=$(jq --argjson tmpl "$template_substituted" '
        .model = $tmpl.model
        | .effortLevel = $tmpl.effortLevel
        | .permissions = (.permissions // {})
        | .permissions.allow = (((.permissions.allow // []) + $tmpl.permissions.allow) | unique)
        | .hooks = (.hooks // {})
        | .hooks.SessionStart = $tmpl.hooks.SessionStart
        | .hooks.SessionEnd = $tmpl.hooks.SessionEnd
        | .enabledPlugins = ((.enabledPlugins // {}) + $tmpl.enabledPlugins)
    ' "$target_path" 2>/dev/null)
    if [ -z "$merged" ]; then
        log_warning "Claude settings merge produced empty output, skipping write"
        return 0
    fi

    echo "$merged" > "$target_path"
    log_success "Claude settings.json merged from template (user customizations preserved)"
}

# SDD-002 (PR #51): single source of truth for the "dotfiles-owned" subset of
# settings.json lives at ai/claude/settings.json. Previous inline `HOOK_ENTRY`
# heredoc + `jq --argjson` is gone -- merge_claude_settings reads the template,
# substitutes __HOOK_COMMAND__, applies per-key policy, bootstraps if missing.
log_info "Applying Claude settings.json template + registering SessionStart/SessionEnd hooks..."
CLAUDE_SETTINGS="$HOME/.claude/settings.json"
CLAUDE_SETTINGS_TEMPLATE="$CURRENT_DIR/ai/claude/settings.json"
# CLI-025: both session hooks are agnostic `dotf mem` nouns, invoked directly (no
# shell-twin shim — claude-session-start.{sh,ps1} + session-handoff.{sh,ps1} are
# deleted). Absolute binary path keeps the hooks working even when ~/.local/bin is
# off the profile PATH (the env-contract owns that invariant; see #531).
EXPECTED_HOOK_COMMAND="$HOME/.local/bin/dotf mem session-start"
EXPECTED_SESSION_END_COMMAND="$HOME/.local/bin/dotf mem session-end"
merge_claude_settings "$CLAUDE_SETTINGS_TEMPLATE" "$CLAUDE_SETTINGS" "$EXPECTED_HOOK_COMMAND" "$EXPECTED_SESSION_END_COMMAND"

# Deploy auto-memory symlinks (vault → Claude Code)
# Memory lives in the knowledge vault, not in this repo (see ADR-007)
# Scans both 10_projects/ and 50_work/ for memory directories.
# VAULT_ROOT honors the ADR-025 seam ($VAULT_PATH, set by the sourced paths.sh)
# with the legacy default as fallback — parity with the agy hive-vault block above.
VAULT_ROOT="${VAULT_PATH:-$HOME/Projects/knowledge}"
VAULT_PROJECTS="$VAULT_ROOT/10_projects"
VAULT_WORK="$VAULT_ROOT/50_work"
if [ -d "$VAULT_ROOT" ]; then
    log_info "Deploying auto-memory symlinks from vault..."
    linked_memory_count=0

    # Helper: create symlink for a vault memory dir
    _link_memory() {
        local memory_source="$1" cwd_path="$2" project_name="$3"
        local encoded_path target_dir

        encoded_path=$(printf '%s' "$cwd_path" | sed 's|/|-|g')
        target_dir="$HOME/.claude/projects/$encoded_path/memory"

        ensure_directory "$HOME/.claude/projects/$encoded_path"

        if [ -L "$target_dir" ]; then
            rm "$target_dir"
        elif [ -d "$target_dir" ] && [ "$(ls -A "$target_dir" 2>/dev/null)" ]; then
            log_warning "Backing up existing memory for $project_name"
            mv "$target_dir" "${target_dir}.bak.$(date +%s)"
        elif [ -d "$target_dir" ]; then
            rmdir "$target_dir" 2>/dev/null || true
        fi

        # Count SUCCESSES, not attempts. Incrementing unconditionally made the
        # summary below report links that were never created — a counter
        # answering "how many did I try" while its message claims "how many
        # worked".
        if ln -s "$memory_source" "$target_dir"; then
            linked_memory_count=$((linked_memory_count + 1))
        else
            log_warning "could not link auto-memory for ${project_name}: ln -s failed"
        fi
    }

    # 10_projects/*: convention — repo at ~/Projects/<name>
    if [ -d "$VAULT_PROJECTS" ]; then
        for project_dir in "$VAULT_PROJECTS"/*/; do
            [ -d "$project_dir" ] || continue
            memory_source="${project_dir}memory"
            [ -d "$memory_source" ] || continue

            project_name=$(basename "$project_dir")
            cwd_path="$HOME/Projects/$project_name"
            _link_memory "$memory_source" "$cwd_path" "$project_name"
        done
    fi

    # 50_work/**/memory: work projects — CWD is the vault path itself
    if [ -d "$VAULT_WORK" ]; then
        # `find` used to run inside the process substitution, where its exit
        # status is discarded and `2>/dev/null` hid the reason. A failed scan then
        # linked nothing while setup reported success. Run it first, keep its
        # status, and say so when it fails.
        if ! work_memory_dirs=$(find "$VAULT_WORK" -type d -name "memory" 2>&1); then
            log_warning "scanning ${VAULT_WORK} for memory dirs failed: ${work_memory_dirs}"
            work_memory_dirs=""
        fi
        while read -r memory_source; do
            [ -n "$memory_source" ] || continue
            project_dir=$(dirname "$memory_source")
            project_name=$(basename "$project_dir")
            cwd_path="$project_dir"
            _link_memory "$memory_source" "$cwd_path" "$project_name"
        done < <(printf '%s\n' "$work_memory_dirs")
    fi

    if [ "$linked_memory_count" -gt 0 ]; then
        log_success "Linked auto-memory for $linked_memory_count project(s) from vault"
    fi

    # Migrate orphan memories: local Claude Code memories not yet in vault
    for claude_project in "$HOME/.claude/projects"/*/; do
        [ -d "$claude_project" ] || continue
        memory_dir="${claude_project}memory"
        # Skip if no memory, already a symlink, or empty
        [ -d "$memory_dir" ] && [ ! -L "$memory_dir" ] && [ "$(ls -A "$memory_dir" 2>/dev/null)" ] || continue

        # Extract project name from encoded path (last segment after Projects-)
        encoded_name=$(basename "$claude_project")
        project_name=$(printf '%s' "$encoded_name" | sed 's/.*-Projects-//')
        [ -n "$project_name" ] || continue

        vault_memory="$VAULT_PROJECTS/$project_name/memory"
        # Only migrate if the vault project exists but has no memory dir yet
        if [ -d "$VAULT_PROJECTS/$project_name" ] && [ ! -d "$vault_memory" ]; then
            log_info "Migrating orphan memory: $project_name → vault"
            mv "$memory_dir" "$vault_memory" && ln -s "$vault_memory" "$memory_dir" \
                && log_success "Migrated and linked: $project_name" \
                || log_error "Failed to migrate memory for $project_name"
        fi
    done
fi

# Deploy all skills from the committed records to their per-agent $HOME paths
# (SDD-008, option A). Renders each harness/skills/<n> record to ~/.claude/skills,
# ~/.config/opencode/commands, ~/.gemini/skills + ~/.gemini/prompts, and injects
# the copilot catalog into ~/.copilot/copilot-instructions.md, honoring each
# skill's targets[]. Deploy is a regular copy — never a symlink — and replaces
# any pre-existing vault symlink with a copy first, ending the BUG-100 fragility
# class. Records are committed, so this runs offline (no vault required); the
# earlier --refresh re-renders the records from the vault SSOT when present.
# Runs last so the per-agent base files (e.g. ~/.copilot/copilot-instructions.md)
# are already in place for catalog injection.
if ( cd "$CURRENT_DIR" && "$CURRENT_DIR/scripts/compile-harness.sh" --deploy ) >/dev/null; then
    log_success "Skills deployed from records (claude / opencode / agy / copilot)"
else
    log_warning "compile-harness --deploy reported issues; check skill records"
fi

# Weekly vault maintenance cron (Sundays 10:00 AM)
log_info "Setting up weekly vault maintenance cron..."
if command -v crontab >/dev/null 2>&1; then
    CRON_CMD="7 10 * * 0 $CURRENT_DIR/scripts/vault-maintenance-weekly.sh"
    if crontab -l 2>/dev/null | grep -q "vault-maintenance-weekly"; then
        log_info "Weekly vault maintenance cron already installed"
    else
        # `crontab -l` exits 1 when the user has no crontab yet. Under the
        # script's `set -euo pipefail` that kills the subshell before the echo
        # runs, and pipefail then aborts the whole setup — so a fresh node never
        # gets past this line. `|| true` keeps the empty-crontab case benign.
        (crontab -l 2>/dev/null || true; echo "$CRON_CMD # dotfiles: vault-maintenance-weekly") | crontab -
        log_success "Installed weekly vault maintenance cron (Sundays 10:07)"
    fi
else
    log_warning "crontab not available, skipping weekly maintenance cron"
fi

# Test if files are correctly linked
log_success "Installation completed! Verifying file links..."

check_deployed "$DOTFILES_DIR/.zsh/aliases.zsh" "$HOME/.zsh/aliases.zsh" "aliases.zsh"
check_deployed "$DOTFILES_DIR/.zsh/functions.zsh" "$HOME/.zsh/functions.zsh" "functions.zsh"
check_deployed "$DOTFILES_DIR/.zsh/functions.sh" "$HOME/.zsh/functions.sh" "functions.sh"
# NOTE: .zshrc intentionally NOT under strict check_deployed -- setup-linux.sh
# may legitimately modify it (e.g. stripping stale gh-copilot eval lines via
# sed -i). Same rationale as .bashrc: tool installers append PATH/init lines,
# producing legitimate drift that should not abort the bootstrap.
# NOTE: .bashrc intentionally NOT under strict check_deployed -- tool installers
# (opencode, bun, NVM, ggshield) append PATH/init lines to ~/.bashrc post-deploy,
# producing legitimate drift. Just verify existence.
[ -f "$HOME/.bashrc" ] && log_success ".bashrc exists (drift expected from installers)" || log_error ".bashrc missing (run setup-linux.sh)"
file_exists "$HOME/.bash/bash_aliases" && log_success "bash_aliases created" || log_error "bash_aliases issue"

# Check dependencies
check_dependencies "git" "zsh" "eza" "direnv" "node" "npm" "zoxide" "docker" "docker-compose" "kubectl" "helm" "terraform" "ansible" "pip"

# Deploy env-contract.json so `dotf doctor` can find it under $DOTFILES_DIR.
if [ -f "$CURRENT_DIR/env-contract.json" ]; then
    cp -f "$CURRENT_DIR/env-contract.json" "$DOTFILES_DIR/env-contract.json"
    log_success "Deployed env-contract.json to $DOTFILES_DIR/"
fi

# Render the per-machine path file (ADR-025) from the contract + the
# ~/.config/dotfiles/machine.json overrides, so the profiles source it and the
# `dotf doctor` run below sees no path drift. Sourcing it here makes this setup
# shell match what the next login shell will see.
if command -v dotf >/dev/null 2>&1; then
    # Seed DOTFILES_REPO_DIR into machine.json to the checkout setup runs from,
    # BEFORE generating the path file, so the cascade (and the generated paths.sh)
    # resolve the real repo instead of the phantom contract default — otherwise
    # `dotf update`/`mem` no-op on a fresh machine (BUG-029/#696). Idempotent and
    # preserves any other overrides (e.g. VAULT_PATH).
    if dotf env set DOTFILES_REPO_DIR "$CURRENT_DIR" >/dev/null; then
        log_success "Seeded DOTFILES_REPO_DIR=$CURRENT_DIR in machine.json"
    else
        log_warning "dotf env set DOTFILES_REPO_DIR failed (update/mem fall back to the git walk-up)"
    fi
    if dotf env generate; then
        log_success "Generated $DOTFILES_DIR/paths.sh (dotf env generate)"
        # shellcheck disable=SC1091
        [ -f "$DOTFILES_DIR/paths.sh" ] && . "$DOTFILES_DIR/paths.sh"
    else
        log_warning "dotf env generate failed (profiles fall back to inline defaults)"
    fi
fi

# Re-enforce main dotfiles at the end of setup to overwrite any third-party installer mutations.
deploy_file "$DOTFILES_DIR/.zshrc" "$HOME/.zshrc"
[ -f "$DOTFILES_DIR/.bashrc" ] && deploy_file "$DOTFILES_DIR/.bashrc" "$HOME/.bashrc"
[ -f "$DOTFILES_DIR/.profile" ] && deploy_file "$DOTFILES_DIR/.profile" "$HOME/.profile"

# Final assertion against env-contract.json -- catches drift between what
# setup just deployed and what's actually in place / on PATH / in env vars.
#
# Pre-export the REFACTOR-002 path vars so `dotf doctor` sees what the deployed
# RC files WILL set on the next shell. Without this, every fresh setup run
# reports false warnings because the running shell hasn't re-sourced .zshrc/.bashrc.
# Values must match the corresponding `export` lines in .zshrc + .bashrc.
export SCRIPTS_DIR="${SCRIPTS_DIR:-$DOTFILES_DIR/scripts}"
export GEMINI_HOME="${GEMINI_HOME:-$HOME/.gemini}"
# AGY_HOME (SSOT name per env-contract.json) lives inside GEMINI_HOME, since
# agy inherits the legacy ~/.gemini/ path for backwards compat with gemini-cli.
export AGY_HOME="${AGY_HOME:-$GEMINI_HOME/antigravity-cli}"
export COPILOT_HOME="${COPILOT_HOME:-$HOME/.copilot}"
export OPENCODE_HOME="${OPENCODE_HOME:-$HOME/.config/opencode}"
# BUG-021 (2026-05-21): pre-export DOTFILES_REPO_DIR so `dotf doctor` (which
# byte-compares the repo against the deployed copy, CLI-019) sees it set even
# when the running shell's profile hasn't been re-evaluated post-deploy.
# Mirror of the setup-windows.ps1 fix.
export DOTFILES_REPO_DIR="${DOTFILES_REPO_DIR:-$HOME/Projects/dotfiles}"

# Final consolidated diagnostics: `dotf doctor` (ADR-021, CLI-012) replaces the
# retired doctor.sh + healthcheck.sh twins in one cross-compiled sweep. Non-fatal:
# surfaces deploy gaps (env-contract drift, version/symlink/vault mismatches) but
# does NOT alter setup's exit code. Needs dotf on PATH (installed above by
# install_dotf); a clean box where that step failed skips gracefully.
if command -v dotf >/dev/null 2>&1; then
    log_info "Running post-setup diagnostics (dotf doctor)..."
    echo
    dotf doctor || log_warning "dotf doctor reported one or more FAIL items -- review output above; re-run with 'dotf doctor'"
    echo
else
    log_warning "dotf not on PATH -- skipping post-setup diagnostics (run ./scripts/install-dotf.sh, then 'dotf doctor')"
fi

log_info "To apply changes immediately, run:"
log_info "  - For Bash: source ~/.bashrc"
log_info "  - For Zsh:  source ~/.zshrc"
