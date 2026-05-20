#!/bin/bash

# setup-linux.sh: Install dotfiles and set up environment
# Usage: ./setup-linux.sh
#
# Philosophy: opportunistic configuration. The script attempts to set up
# every supported tool and warns when something is not available, rather
# than requiring opt-in flags. Heavy / disk-intensive installs (e.g.
# future Ollama, Ghostty) may add explicit flags case by case.

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
    safe_copy "$CURRENT_DIR/mcp-servers.json" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.zshrc" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.profile" "$DOTFILES_DIR/" 2>/dev/null || true
    if [ -f "$CURRENT_DIR/.bashrc" ]; then
        safe_copy "$CURRENT_DIR/.bashrc" "$DOTFILES_DIR/" 2>/dev/null || true
    fi    
    safe_copy "$CURRENT_DIR/.gitconfig" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/tmux.conf" "$DOTFILES_DIR/" 2>/dev/null || true
    cp -rf "$CURRENT_DIR/.zsh/." "$DOTFILES_DIR/.zsh/" 2>/dev/null || true
    ensure_directory "$DOTFILES_DIR/ssh"
    cp -rf "$CURRENT_DIR/ssh/"* "$DOTFILES_DIR/ssh/" 2>/dev/null || true
    cp -rf "$CURRENT_DIR/scripts/"* "$DOTFILES_DIR/scripts/" 2>/dev/null || true
else
    log_info "Already in dotfiles directory, skipping copy..."
fi

# Create symbolic links to main dotfiles
log_info "Creating symbolic links for main dotfiles..."
ln -sf "$DOTFILES_DIR/.zshrc" "$HOME/.zshrc"
ln -sf "$DOTFILES_DIR/.profile" "$HOME/.profile" 2>/dev/null || true

# SSH config and public key
log_info "Setting up SSH config..."
ensure_directory "$HOME/.ssh"
ln -sf "$DOTFILES_DIR/ssh/config" "$HOME/.ssh/config"
chmod 600 "$DOTFILES_DIR/ssh/config"
cp "$DOTFILES_DIR/ssh/id_ed25519.pub" "$HOME/.ssh/id_ed25519.pub" 2>/dev/null || true

# Git configuration
log_info "Setting up Git configuration..."
if [ -f "$DOTFILES_DIR/.gitconfig" ]; then
    ln -sf "$DOTFILES_DIR/.gitconfig" "$HOME/.gitconfig"
    log_success "Linked: .gitconfig → ~/.gitconfig"
else
    log_warning ".gitconfig not found in dotfiles"
fi

# Create symbolic links for .zsh directory and utils.sh
log_info "Setting up .zsh directory and utils.sh..."
ensure_directory "$HOME/.zsh"
ln -sf "$DOTFILES_DIR/.zsh/aliases.zsh" "$HOME/.zsh/aliases.zsh"
ln -sf "$DOTFILES_DIR/.zsh/functions.zsh" "$HOME/.zsh/functions.zsh"
ln -sf "$DOTFILES_DIR/.zsh/nvm.zsh" "$HOME/.zsh/nvm.zsh"
ln -sf "$DOTFILES_DIR/tmux.conf" "$HOME/.tmux.conf"
chmod +x "$DOTFILES_DIR/scripts/utils.sh"
chmod +x "$DOTFILES_DIR/scripts/github-secrets-manager.sh"
chmod +x "$DOTFILES_DIR/scripts/age-encrypt-decrypt.sh"
chmod +x "$DOTFILES_DIR/scripts/install-precommit.sh"
chmod +x "$DOTFILES_DIR/scripts/load-secrets.sh"
chmod +x "$DOTFILES_DIR/scripts/dotfiles-sync.sh"
chmod +x "$DOTFILES_DIR/scripts/claude-session-start.sh"
chmod +x "$DOTFILES_DIR/scripts/vault-health.sh"
chmod +x "$DOTFILES_DIR/scripts/knowledge-crystallize.sh"
chmod +x "$DOTFILES_DIR/scripts/doctor.sh"

# Copy sensitive directory (env-mapping.conf and encrypted files)
log_info "Setting up sensitive directory..."
ensure_directory "$DOTFILES_DIR/sensitive"
if [ "$CURRENT_DIR" != "$DOTFILES_DIR" ]; then
    cp -rf "$CURRENT_DIR/sensitive/"* "$DOTFILES_DIR/sensitive/" 2>/dev/null || true
fi


# Update functions.zsh to source utils.sh if not already done
if [ -f "$DOTFILES_DIR/.zsh/functions.zsh" ]; then
    if ! grep -q "source.*utils.sh" "$DOTFILES_DIR/.zsh/functions.zsh"; then
        log_info "Appending utils.sh source to existing functions.zsh..."
        printf '\n# Source utility functions\n. %s/scripts/utils.sh\n' "$DOTFILES_DIR" >> "$DOTFILES_DIR/.zsh/functions.zsh"
    else
        log_info "functions.zsh already sources utils.sh, skipping"
    fi
else
    log_info "Creating functions.zsh file..."
    cat > "$DOTFILES_DIR/.zsh/functions.zsh" << EOF

# Source the utils.sh and setup-gh-secrets.sh scripts
. "$DOTFILES_DIR/scripts/utils.sh"

EOF
fi

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

# Link .bashrc
ln -sf "$DOTFILES_DIR/.bashrc" "$HOME/.bashrc"

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
    curl -Lo /tmp/shellcheck.tar.xz "https://github.com/koalaman/ShellCheck/releases/latest/download/shellcheck-stable.linux.x86_64.tar.xz" 2>/dev/null \
        && tar xJf /tmp/shellcheck.tar.xz -C /tmp \
        && cp /tmp/shellcheck-stable/shellcheck "$HOME/.local/bin/" \
        && rm -rf /tmp/shellcheck.tar.xz /tmp/shellcheck-stable \
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

# Setup AI configuration
log_info "Setting up AI configuration..."

# Gemini CLI config (config deploy is always safe, even without gemini installed)
ensure_directory "$HOME/.gemini"
ensure_directory "$HOME/.gemini/prompts"
cp -rf "$CURRENT_DIR/ai/gemini/"* "$HOME/.gemini/" 2>/dev/null || true
# Sync Gemini prompts: remove stale skill-derived prompts not in source
for target_file in "$HOME/.gemini/prompts/"*.md; do
    [ -f "$target_file" ] || continue
    prompt_name=$(basename "$target_file" .md)
    [ -d "$CURRENT_DIR/ai/skills/$prompt_name" ] || rm -f "$target_file"
done
# Extract SKILL.md content as flat prompts for Gemini (strip YAML frontmatter)
for skill_dir in "$CURRENT_DIR/ai/skills/"*/; do
    if [ -d "$skill_dir" ] && [ -f "${skill_dir}SKILL.md" ]; then
        skill_name=$(basename "$skill_dir")
        # Copy SKILL.md as flat file, stripping YAML frontmatter
        tr -d '\r' < "${skill_dir}SKILL.md" | sed '/^---$/,/^---$/d' > "$HOME/.gemini/prompts/${skill_name}.md"
    fi
done
# Force copy master files (Neural Hive Protocol)
rm -f "$HOME/.gemini/GEMINI.md"
cp "$CURRENT_DIR/ai/gemini/GEMINI.md" "$HOME/.gemini/GEMINI.md"
if grep -q 'First, read `AGENTS.md`' "$HOME/.gemini/GEMINI.md"; then
    log_success "GEMINI.md deployed successfully (verified pointer to AGENTS.md)"
else
    echo "❌ Error: GEMINI.md deployment failed verification"
fi
log_success "Gemini CLI configured"

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
cp -f "$CURRENT_DIR/scripts/init-project.sh" "$HOME/.claude/" 2>/dev/null || true
# Sync Claude skills: remove stale skill directories not in source.
# CRITICAL: For symlinks (vault-hosted skills, see link_vault_skills below),
# unlink only — never rm -rf through a symlink with trailing slash, which
# POSIX rm follows and would delete vault content. The vault-skills loop
# below re-creates the symlink if the source still exists.
for target_dir in "$HOME/.claude/skills/"*/; do
    target_path="${target_dir%/}"
    [ -e "$target_path" ] || [ -L "$target_path" ] || continue
    skill_name=$(basename "$target_path")
    if [ -L "$target_path" ]; then
        unlink "$target_path"
    elif [ ! -d "$CURRENT_DIR/ai/skills/$skill_name" ]; then
        rm -rf "$target_path"
    fi
done
# Copy skill directories (each skill has its own dir with SKILL.md)
for skill_dir in "$CURRENT_DIR/ai/skills/"*/; do
    if [ -d "$skill_dir" ]; then
        skill_name=$(basename "$skill_dir")
        ensure_directory "$HOME/.claude/skills/$skill_name"
        cp -rf "$skill_dir"* "$HOME/.claude/skills/$skill_name/" 2>/dev/null || true
    fi
done
# Force copy master files (Neural Hive Protocol)
rm -f "$HOME/.claude/CLAUDE.md"
cp "$CURRENT_DIR/ai/claude/CLAUDE.md" "$HOME/.claude/CLAUDE.md"
if grep -q 'First, read `AGENTS.md`' "$HOME/.claude/CLAUDE.md"; then
    log_success "CLAUDE.md deployed successfully (verified pointer to AGENTS.md)"
else
    echo "❌ Error: CLAUDE.md deployment failed verification"
fi
chmod +x "$HOME/.claude/init-project.sh" 2>/dev/null || true
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

# OpenCode (secondary AI coding agent — see ADR-009 and specs/AI-011-opencode-bootstrap)
# Idempotent per pattern-setup-script-idempotence:
#   - install only if absent (no forced re-install on every run)
#   - reconcile config (not skip-if-exists) so source-of-truth wins on drift
#   - no silenced errors; failures surface as log_warning
log_info "Setting up OpenCode configuration..."
OPENCODE_BIN_DIR="$HOME/.opencode/bin"
OPENCODE_BINARY="$OPENCODE_BIN_DIR/opencode"

if ! command -v opencode >/dev/null 2>&1 && [ ! -x "$OPENCODE_BINARY" ]; then
    log_info "Installing opencode via official install script..."
    if curl -fsSL https://opencode.ai/install | bash; then
        log_success "opencode installed to $OPENCODE_BIN_DIR"
    else
        log_warning "opencode install failed — re-run setup or install manually (https://opencode.ai/docs/)"
    fi
else
    log_info "opencode already installed"
fi

# Ensure $HOME/.opencode/bin on PATH (defensive; install script also tries this)
OPENCODE_PATH_LINE='export PATH="$HOME/.opencode/bin:$PATH"'
[ -f "$HOME/.zshrc" ] && ensure_line_in_file "$HOME/.zshrc" "$OPENCODE_PATH_LINE"
[ -f "$HOME/.bashrc" ] && ensure_line_in_file "$HOME/.bashrc" "$OPENCODE_PATH_LINE"

# Deploy opencode.jsonc — reconcile-not-skip per pattern-setup-script-idempotence
ensure_directory "$HOME/.config/opencode"
OPENCODE_CONFIG_SRC="$CURRENT_DIR/ai/opencode/opencode.jsonc"
OPENCODE_CONFIG_DST="$HOME/.config/opencode/opencode.jsonc"
if [ -f "$OPENCODE_CONFIG_SRC" ]; then
    if [ -f "$OPENCODE_CONFIG_DST" ] && cmp -s "$OPENCODE_CONFIG_SRC" "$OPENCODE_CONFIG_DST"; then
        log_info "opencode.jsonc already in sync"
    else
        cp "$OPENCODE_CONFIG_SRC" "$OPENCODE_CONFIG_DST"
        log_success "Deployed opencode.jsonc to $OPENCODE_CONFIG_DST"
    fi
else
    log_warning "opencode.jsonc source missing: $OPENCODE_CONFIG_SRC"
fi

# Deploy opencode commands (AI-012: skills→commands port).
# Source: ai/opencode/commands/*.md (kept in sync with ai/skills/ via
# scripts/skills-to-opencode.sh; CI gate enforces parity). Target: the
# OpenCode global commands directory, so /audit, /test, /writing-plans,
# etc. are available from any repo where the user launches `oc`.
OPENCODE_CMDS_SRC="$CURRENT_DIR/ai/opencode/commands"
OPENCODE_CMDS_DST="$HOME/.config/opencode/commands"
if [ -d "$OPENCODE_CMDS_SRC" ]; then
    ensure_directory "$OPENCODE_CMDS_DST"
    cmds_added=0
    cmds_skipped=0
    cmds_removed=0
    for src in "$OPENCODE_CMDS_SRC"/*.md; do
        [ -f "$src" ] || continue
        dst="$OPENCODE_CMDS_DST/$(basename "$src")"
        if [ -f "$dst" ] && cmp -s "$src" "$dst"; then
            cmds_skipped=$((cmds_skipped + 1))
        else
            cp "$src" "$dst"
            cmds_added=$((cmds_added + 1))
        fi
    done
    # Remove orphan commands on disk that no longer exist in the repo
    # (e.g. a skill was added to the skip-list and its command should
    # disappear from the user's deployed set).
    for dst in "$OPENCODE_CMDS_DST"/*.md; do
        [ -f "$dst" ] || continue
        if [ ! -f "$OPENCODE_CMDS_SRC/$(basename "$dst")" ]; then
            rm -f "$dst"
            cmds_removed=$((cmds_removed + 1))
        fi
    done
    log_success "opencode commands: $cmds_added added, $cmds_skipped already in sync, $cmds_removed orphans removed"
else
    log_info "opencode commands source missing: $OPENCODE_CMDS_SRC (skipping; run scripts/skills-to-opencode.sh to generate)"
fi

# Post-deploy assertion — binary reachable + version reports
if [ -x "$OPENCODE_BINARY" ] || command -v opencode >/dev/null 2>&1; then
    OPENCODE_VERSION=$("$OPENCODE_BINARY" --version 2>&1 | head -1 || echo "unknown")
    log_success "opencode ready: $OPENCODE_VERSION"
    log_info "First-time use: launch \`opencode\` and run /connect to authenticate (Go subscription)"
else
    log_warning "opencode binary not reachable at $OPENCODE_BINARY after install — agent unavailable"
fi

# Ghostty terminal emulator (TERM-001 — Linux only, Ubuntu universe).
# Detect-and-act with warn-not-fail: this script avoids sudo (same convention
# as tmux / xclip), so apt install is NOT attempted here -- the user runs the
# one-liner once per machine. Config is deployed unconditionally though, so
# the moment the user runs `sudo apt install -y ghostty` the canonical config
# is already in place.
GHOSTTY_VERSION_PINNED="${GHOSTTY_VERSION:-1.3.0}"
if ! command -v ghostty >/dev/null 2>&1; then
    log_warning "ghostty not installed. Run: sudo apt install -y ghostty  (Ubuntu 26.04+ universe; needed for the recommended opencode TUI host)"
else
    INSTALLED_GHOSTTY=$(ghostty --version 2>&1 | head -1 | awk '{print $2}' | sed 's/-.*//')
    if [ "$INSTALLED_GHOSTTY" = "$GHOSTTY_VERSION_PINNED" ]; then
        log_info "ghostty installed: $(ghostty --version 2>&1 | head -1)"
    else
        log_warning "ghostty version drift: installed=$INSTALLED_GHOSTTY pinned=$GHOSTTY_VERSION_PINNED (sudo apt install --only-upgrade ghostty if you want to track the pin)"
    fi
fi

# Deploy ghostty config — reconcile-not-skip pattern (same as opencode.jsonc).
# Inert if ghostty is not installed: the file lives at ~/.config/ghostty/
# regardless and gets read the moment the binary is.
ensure_directory "$HOME/.config/ghostty"
GHOSTTY_CONFIG_SRC="$CURRENT_DIR/terminals/ghostty/config"
GHOSTTY_CONFIG_DST="$HOME/.config/ghostty/config"
if [ -f "$GHOSTTY_CONFIG_SRC" ]; then
    if [ -f "$GHOSTTY_CONFIG_DST" ] && cmp -s "$GHOSTTY_CONFIG_SRC" "$GHOSTTY_CONFIG_DST"; then
        log_info "ghostty config already in sync"
    else
        cp "$GHOSTTY_CONFIG_SRC" "$GHOSTTY_CONFIG_DST"
        log_success "Deployed ghostty config to $GHOSTTY_CONFIG_DST"
    fi
else
    log_warning "ghostty config source missing: $GHOSTTY_CONFIG_SRC"
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

# Register MCP servers (requires Claude Code CLI, Node.js, jq)
# Idempotent: server list lives in mcp-servers.json (SSOT shared with Windows);
# `claude mcp get` is used to skip already-registered entries, and `add` errors
# are surfaced rather than swallowed.
MCP_CONFIG="$DOTFILES_DIR/mcp-servers.json"
if command -v claude >/dev/null 2>&1 && command -v npx >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
    if [ ! -f "$MCP_CONFIG" ]; then
        log_warning "mcp-servers.json not found at $MCP_CONFIG, skipping MCP registration"
    else
        log_info "Registering Claude Code MCP servers from $MCP_CONFIG..."
        mcp_added=0
        mcp_skipped=0
        mcp_failed=0
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
            # Idempotence: skip if `claude mcp get` already knows this name
            if claude mcp get "$mcp_name" >/dev/null 2>&1; then
                log_info "MCP $mcp_name already registered, skipping"
                mcp_skipped=$((mcp_skipped + 1))
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
        done < <(jq -r '.servers[] | [.name, .transport, .args, (.prerequisite_binary // ""), (.prerequisite_command // "")] | @tsv' "$MCP_CONFIG")
        log_success "MCP servers: $mcp_added added, $mcp_skipped already present, $mcp_failed failed"
    fi
else
    log_warning "Claude Code CLI, npx, or jq not found, skipping MCP server registration"
fi

# BUG-004: defense-in-depth wrapper around `claude plugin install`. The bash
# idempotence guard below (`grep -qF "$plugin"` against `claude plugin list`)
# yields a false negative for claude-mem@thedotmack -- it never appears in
# that listing (different marketplace, `@thedotmack` vs `@claude-plugins-official`).
# Every setup run installs claude-mem again, which triggers upstream
# anthropics/claude-code#59870: the CLI's deserialize-modify-serialize cycle
# drops fields outside its internal struct (organizationType,
# organizationRateLimitTier, projects map, onboarding flags), shrinking
# ~/.claude/.claude.json from ~75 KB to ~1.5 KB and forcing re-authentication.
# snapshot_claude_json copies the file to a tempfile BEFORE the install;
# restore_claude_json_if_truncated restores it AFTER, iff the snapshot was
# >= 10 KB and the new file is < 50% of the snapshot size. Complementary to
# SDD-021 session-start canary in claude-session-start.sh (same 10240-byte
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

# Claude Code plugins (requires claude CLI).
# Idempotent: cache the installed-plugins list ONCE before the loop and skip
# entries already present. The wrapper above (BUG-004) catches the
# false-negative case where the idempotence guard misses a plugin (e.g.
# claude-mem@thedotmack) and the resulting `claude plugin install` call
# truncates .claude.json. Same idempotence pattern as MCP registration (line 447).
if command -v claude >/dev/null 2>&1; then
    log_info "Installing Claude Code plugins..."
    installed_plugins=$(claude plugin list 2>/dev/null || true)
    plugins_added=0
    plugins_skipped=0
    for plugin in \
        "claude-mem@thedotmack" \
        "code-simplifier@claude-plugins-official" \
        "gopls-lsp@claude-plugins-official" \
        "security-guidance@claude-plugins-official" \
        "claude-md-management@claude-plugins-official" \
        "claude-code-setup@claude-plugins-official" \
        "frontend-design@claude-plugins-official" \
        "ralph-loop@claude-plugins-official" \
        "code-review@claude-plugins-official" \
        "commit-commands@claude-plugins-official" \
        "pr-review-toolkit@claude-plugins-official"; do
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

# Merge `ai/claude/settings.json` template into the deployed `~/.claude/settings.json`
# per the per-key policy in specs/SDD-002-settings-portability/proposal.md. Bootstrap
# when target missing. Preserves user customizations (Read paths,
# additionalDirectories, third-party hooks like claude-mem / GitGuardian) by only
# touching the keys declared as "ours" in the template. The template's
# __HOOK_COMMAND__ placeholder is replaced via jq --arg before any merge / write.
merge_claude_settings() {
    local template_path="$1"
    local target_path="$2"
    local hook_command="$3"

    if [ ! -f "$template_path" ]; then
        log_warning "Claude settings template not found at $template_path, skipping merge"
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        log_warning "jq not found, skipping settings merge (install jq and re-run)"
        return 0
    fi

    local template_substituted
    template_substituted=$(jq --arg cmd "$hook_command" \
        '(.hooks.SessionStart[0].hooks[0].command) = $cmd' \
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
log_info "Applying Claude settings.json template + registering SessionStart hook..."
CLAUDE_SETTINGS="$HOME/.claude/settings.json"
CLAUDE_SETTINGS_TEMPLATE="$CURRENT_DIR/ai/claude/settings.json"
EXPECTED_HOOK_COMMAND="$HOME/.dotfiles/scripts/claude-session-start.sh"
merge_claude_settings "$CLAUDE_SETTINGS_TEMPLATE" "$CLAUDE_SETTINGS" "$EXPECTED_HOOK_COMMAND"

# Deploy auto-memory symlinks (vault → Claude Code)
# Memory lives in the knowledge vault, not in this repo (see ADR-007)
# Scans both 10_projects/ and 50_work/ for memory directories.
VAULT_ROOT="$HOME/Projects/knowledge"
VAULT_PROJECTS="$VAULT_ROOT/10_projects"
VAULT_WORK="$VAULT_ROOT/50_work"
if [ -d "$VAULT_ROOT" ]; then
    log_info "Deploying auto-memory symlinks from vault..."

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

        ln -s "$memory_source" "$target_dir"
        log_success "Linked auto-memory: $project_name"
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
        find "$VAULT_WORK" -type d -name "memory" 2>/dev/null | while read -r memory_source; do
            project_dir=$(dirname "$memory_source")
            project_name=$(basename "$project_dir")
            cwd_path="$project_dir"
            _link_memory "$memory_source" "$cwd_path" "$project_name"
        done
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

# Deploy vault-hosted skill symlinks (vault → Claude Code)
# Vault skills live in $VAULT_ROOT/00_meta/skills/<name>/SKILL.md per pattern-spec-driven-development.
# Symlink each into ~/.claude/skills/ for Claude Code discovery. Idempotent.
VAULT_SKILLS_DIR="$VAULT_ROOT/00_meta/skills"
if [ -d "$VAULT_SKILLS_DIR" ]; then
    log_info "Linking vault-hosted skills to Claude Code..."
    for skill_dir in "$VAULT_SKILLS_DIR"/*/; do
        [ -d "$skill_dir" ] || continue
        skill_source="${skill_dir%/}"
        skill_name=$(basename "$skill_source")
        target="$HOME/.claude/skills/$skill_name"

        if [ -L "$target" ]; then
            current_target=$(readlink "$target")
            if [ "$current_target" != "$skill_source" ]; then
                rm "$target"
                ln -s "$skill_source" "$target"
                log_success "Re-linked vault skill: $skill_name"
            fi
        elif [ -d "$target" ] && [ -n "$(ls -A "$target" 2>/dev/null)" ]; then
            log_warning "$skill_name exists at $target — backing up"
            mv "$target" "${target}.bak.$(date +%s)"
            ln -s "$skill_source" "$target"
            log_success "Linked vault skill: $skill_name"
        elif [ -d "$target" ]; then
            rmdir "$target" 2>/dev/null || true
            ln -s "$skill_source" "$target"
            log_success "Linked vault skill: $skill_name"
        else
            ln -s "$skill_source" "$target"
            log_success "Linked vault skill: $skill_name"
        fi
    done
fi

# Add project-init alias (AI-agnostic naming)
PROJECT_INIT_LINE='alias project-init="$HOME/.claude/init-project.sh"'
[ -f "$HOME/.zshrc" ] && ensure_line_in_file "$HOME/.zshrc" "$PROJECT_INIT_LINE" 2>/dev/null || true
[ -f "$HOME/.bashrc" ] && ensure_line_in_file "$HOME/.bashrc" "$PROJECT_INIT_LINE" 2>/dev/null || true

# Weekly vault maintenance cron (Sundays 10:00 AM)
log_info "Setting up weekly vault maintenance cron..."
if command -v crontab >/dev/null 2>&1; then
    CRON_CMD="7 10 * * 0 $CURRENT_DIR/scripts/vault-maintenance-weekly.sh"
    if crontab -l 2>/dev/null | grep -q "vault-maintenance-weekly"; then
        log_info "Weekly vault maintenance cron already installed"
    else
        (crontab -l 2>/dev/null; echo "$CRON_CMD # dotfiles: vault-maintenance-weekly") | crontab -
        log_success "Installed weekly vault maintenance cron (Sundays 10:07)"
    fi
else
    log_warning "crontab not available, skipping weekly maintenance cron"
fi

log_info "Adding dotfiles scripts directory to PATH..."
PATH_LINE='export PATH=$HOME/.dotfiles/scripts:$PATH'
[ -f "$HOME/.zshrc" ] && ensure_line_in_file "$HOME/.zshrc" "$PATH_LINE" && log_success "Added scripts to PATH in .zshrc"
[ -f "$HOME/.bashrc" ] && ensure_line_in_file "$HOME/.bashrc" "$PATH_LINE" && log_success "Added scripts to PATH in .bashrc"

# Test if files are correctly linked
log_success "Installation completed! Verifying file links..."

verify_symlink "$HOME/.zsh/aliases.zsh" "aliases.zsh"
verify_symlink "$HOME/.zsh/functions.zsh" "functions.zsh"
verify_symlink "$HOME/.zshrc" ".zshrc"
verify_symlink "$HOME/.bashrc" ".bashrc"
file_exists "$HOME/.bash/bash_aliases" && log_success "bash_aliases created" || log_error "bash_aliases issue"

# Check dependencies
check_dependencies "git" "zsh" "eza" "direnv" "node" "npm" "zoxide" "docker" "docker-compose" "kubectl" "helm" "terraform" "ansible" "pip"

# Deploy env-contract.json so doctor.sh can find it under $DOTFILES_DIR.
if [ -f "$CURRENT_DIR/env-contract.json" ]; then
    cp -f "$CURRENT_DIR/env-contract.json" "$DOTFILES_DIR/env-contract.json"
    log_success "Deployed env-contract.json to $DOTFILES_DIR/"
fi

# Final assertion against env-contract.json -- catches drift between what
# setup just deployed and what's actually in place / on PATH / in env vars.
DOCTOR_SCRIPT="$DOTFILES_DIR/scripts/doctor.sh"
if [ -x "$DOCTOR_SCRIPT" ] && command -v jq >/dev/null 2>&1; then
    log_info "Running post-setup doctor check..."
    echo
    bash "$DOCTOR_SCRIPT" || log_warning "doctor reported one or more required items missing -- review output above"
    echo
elif [ -f "$DOCTOR_SCRIPT" ]; then
    log_warning "doctor.sh present but not executable or jq missing, skipping post-setup check"
fi

log_info "To apply changes immediately, run:"
log_info "  - For Bash: source ~/.bashrc"
log_info "  - For Zsh:  source ~/.zshrc"
