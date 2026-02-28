#!/bin/bash

# setup-linux.sh: Install dotfiles and set up environment
# Usage: ./setup-linux.sh

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
    safe_copy "$CURRENT_DIR/.zshrc" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.profile" "$DOTFILES_DIR/" 2>/dev/null || true
    if [ -f "$CURRENT_DIR/.bashrc" ]; then
        safe_copy "$CURRENT_DIR/.bashrc" "$DOTFILES_DIR/" 2>/dev/null || true
    fi    
    safe_copy "$CURRENT_DIR/.gitconfig" "$DOTFILES_DIR/" 2>/dev/null || true
    cp -rf "$CURRENT_DIR/.zsh/"* "$DOTFILES_DIR/.zsh/" 2>/dev/null || true
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
cp -n "$DOTFILES_DIR/ssh/id_ed25519.pub" "$HOME/.ssh/id_ed25519.pub" 2>/dev/null || true

# Git configuration
log_info "Setting up Git configuration..."
if [ -f "$DOTFILES_DIR/.gitconfig" ]; then
    if [ -f "$HOME/.gitconfig" ]; then
        log_warning ".gitconfig already exists at $HOME/.gitconfig, skipping"
        log_info "To update manually: cp '$DOTFILES_DIR/.gitconfig' '$HOME/.gitconfig'"
    else
        cp "$DOTFILES_DIR/.gitconfig" "$HOME/.gitconfig"
        log_success "Deployed .gitconfig"
    fi
else
    log_warning ".gitconfig not found in dotfiles"
fi

# Create symbolic links for .zsh directory and utils.sh
log_info "Setting up .zsh directory and utils.sh..."
ensure_directory "$HOME/.zsh"
ln -sf "$DOTFILES_DIR/.zsh/aliases.zsh" "$HOME/.zsh/aliases.zsh"
ln -sf "$DOTFILES_DIR/.zsh/functions.zsh" "$HOME/.zsh/functions.zsh"
ln -sf "$DOTFILES_DIR/.zsh/nvm.zsh" "$HOME/.zsh/nvm.zsh"
chmod +x "$DOTFILES_DIR/scripts/utils.sh"
chmod +x "$DOTFILES_DIR/scripts/github-secrets-manager.sh"
chmod +x "$DOTFILES_DIR/scripts/age-encrypt-decrypt.sh"
chmod +x "$DOTFILES_DIR/scripts/install-precommit.sh"
chmod +x "$DOTFILES_DIR/scripts/load-secrets.sh"
chmod +x "$DOTFILES_DIR/scripts/dotfiles-sync.sh"

# Copy sensitive directory (env-mapping.conf and encrypted files)
log_info "Setting up sensitive directory..."
ensure_directory "$DOTFILES_DIR/sensitive"
if [ "$CURRENT_DIR" != "$DOTFILES_DIR" ]; then
    cp -rf "$CURRENT_DIR/sensitive/"* "$DOTFILES_DIR/sensitive/" 2>/dev/null || true
fi


# Update functions.zsh to source utils.sh if not already done
if [ -f "$DOTFILES_DIR/.zsh/functions.zsh" ]; then
    if ! grep -q "source.*utils.sh" "$DOTFILES_DIR/.zsh/functions.zsh"; then
        log_info "Updating functions.zsh to source utils.sh..."
        printf '\n# Source utility functions\n. %s/scripts/utils.sh\n' "$DOTFILES_DIR" >> "$DOTFILES_DIR/.zsh/functions.zsh"
    fi
    log_info "functions.zsh already exists, skipping creation..."
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
$(cat "$DOTFILES_DIR/.zsh/aliases.zsh" | grep "alias" | sed 's/alias /alias /g')
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

# Setup AI configuration
log_info "Setting up AI configuration..."

# Gemini CLI
ensure_directory "$HOME/.gemini"
ensure_directory "$HOME/.gemini/prompts"
cp -rf "$CURRENT_DIR/ai/gemini/"* "$HOME/.gemini/" 2>/dev/null || true
# Extract SKILL.md content as flat prompts for Gemini (strip YAML frontmatter)
for skill_dir in "$CURRENT_DIR/ai/skills/"*/; do
    if [ -d "$skill_dir" ] && [ -f "${skill_dir}SKILL.md" ]; then
        skill_name=$(basename "$skill_dir")
        # Copy SKILL.md as flat file, stripping YAML frontmatter
        sed '/^---$/,/^---$/d' "${skill_dir}SKILL.md" > "$HOME/.gemini/prompts/${skill_name}.md"
    fi
done
# Force copy master files (Neural Hive Protocol)
rm -f "$HOME/.gemini/GEMINI.md"
cp "$CURRENT_DIR/ai/gemini/GEMINI.md" "$HOME/.gemini/GEMINI.md"
if grep -q "CORE PRINCIPLE" "$HOME/.gemini/GEMINI.md"; then
    log_success "GEMINI.md deployed successfully (verified)"
else
    echo "❌ Error: GEMINI.md deployment failed verification"
fi
log_success "Gemini CLI configured"

# Claude Code
ensure_directory "$HOME/.claude"
ensure_directory "$HOME/.claude/skills"
cp -rf "$CURRENT_DIR/ai/claude/"* "$HOME/.claude/" 2>/dev/null || true
cp -f "$CURRENT_DIR/scripts/init-project.sh" "$HOME/.claude/" 2>/dev/null || true
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
if grep -q "CORE PRINCIPLE" "$HOME/.claude/CLAUDE.md"; then
    log_success "CLAUDE.md deployed successfully (verified)"
else
    echo "❌ Error: CLAUDE.md deployment failed verification"
fi
chmod +x "$HOME/.claude/init-project.sh" 2>/dev/null || true
log_success "Claude Code configured with skills"

# Create convenience symlinks for versioned-only binaries
PYTHON_DIR=$(ls -d "$HOME/Applications"/python-* 2>/dev/null | sort -V | tail -1) || true
if [ -n "$PYTHON_DIR" ] && [ -d "$PYTHON_DIR/bin" ] && [ ! -e "$PYTHON_DIR/bin/python" ]; then
    log_info "Creating python symlink..."
    ln -s python3 "$PYTHON_DIR/bin/python"
    log_success "python -> python3 symlink created"
fi

# GitHub Copilot CLI (requires gh)
if command -v gh >/dev/null 2>&1; then
    log_info "Installing GitHub Copilot CLI extension..."
    gh extension install github/gh-copilot 2>/dev/null || true
    ensure_directory "$HOME/.copilot"
    cp -rf "$CURRENT_DIR/ai/copilot/"* "$HOME/.copilot/" 2>/dev/null || true
    # Also copy to .github in home if needed, or just let gh manage it
    # cp -f "$CURRENT_DIR/ai/copilot/copilot-instructions.md" "$HOME/.copilot/copilot-instructions.md" 2>/dev/null || true
    log_success "GitHub Copilot CLI configured"
    
    # Aliases
    log_info "Adding Copilot aliases to .zshrc/.bashrc..."
    COPILOT_SUGGEST='eval "$(gh copilot alias -- bash)"'
    ensure_line_in_file "$HOME/.zshrc" "$COPILOT_SUGGEST"
    ensure_line_in_file "$HOME/.bashrc" "$COPILOT_SUGGEST"
else
    log_warning "GitHub CLI (gh) not found, skipping Copilot installation"
fi

# Register MCP servers (requires Claude Code CLI and Node.js)
if command -v claude >/dev/null 2>&1 && command -v npx >/dev/null 2>&1; then
    log_info "Registering Claude Code MCP servers..."
    claude mcp add --transport stdio drawio --scope user -- npx -y @drawio/mcp 2>/dev/null || true
    claude mcp add --transport http socket --scope user -- https://mcp.socket.dev/ 2>/dev/null || true
    claude mcp add --transport stdio sequential-thinking --scope user -- npx -y @modelcontextprotocol/server-sequential-thinking 2>/dev/null || true
    claude mcp add --transport http context7 --scope user -- https://mcp.context7.com/mcp 2>/dev/null || true
    log_success "MCP servers registered"
else
    log_warning "Claude Code CLI or npx not found, skipping MCP server registration"
fi

# Claude Code plugins (requires claude CLI)
if command -v claude >/dev/null 2>&1; then
    log_info "Installing Claude Code plugins..."
    for plugin in \
        "claude-mem@thedotmack" \
        "code-simplifier@claude-plugins-official" \
        "github@claude-plugins-official" \
        "gopls-lsp@claude-plugins-official" \
        "security-guidance@claude-plugins-official" \
        "claude-md-management@claude-plugins-official" \
        "claude-code-setup@claude-plugins-official" \
        "frontend-design@claude-plugins-official" \
        "ralph-loop@claude-plugins-official" \
        "code-review@claude-plugins-official" \
        "commit-commands@claude-plugins-official" \
        "pr-review-toolkit@claude-plugins-official"; do
        claude plugin install "$plugin" >/dev/null 2>&1 || true
    done
    log_success "Claude Code plugins installed"
else
    log_warning "Claude Code CLI not found, skipping plugin installation"
fi

# Register Claude Code SessionStart hook for vault health
CLAUDE_SETTINGS="$HOME/.claude/settings.json"
if [ -f "$CLAUDE_SETTINGS" ] && command -v jq >/dev/null 2>&1; then
    if jq -e '.hooks.SessionStart' "$CLAUDE_SETTINGS" >/dev/null 2>&1; then
        log_info "SessionStart hook already configured, skipping"
    else
        log_info "Adding SessionStart hook to Claude Code settings..."
        HOOK_ENTRY='{"matcher":"","hooks":[{"type":"command","command":"$HOME/.dotfiles/scripts/claude-session-start.sh","timeout":30}]}'
        jq --argjson hook "[$HOOK_ENTRY]" '.hooks.SessionStart = $hook' "$CLAUDE_SETTINGS" > "${CLAUDE_SETTINGS}.tmp" \
            && mv "${CLAUDE_SETTINGS}.tmp" "$CLAUDE_SETTINGS"
        log_success "SessionStart hook registered"
    fi
elif [ ! -f "$CLAUDE_SETTINGS" ]; then
    log_warning "Claude Code settings.json not found, skipping hook registration"
else
    log_warning "jq not found, skipping hook registration (install jq and re-run)"
fi

# Deploy auto-memory symlinks (vault → Claude Code)
# Memory lives in the knowledge vault, not in this repo (see ADR-007)
VAULT_PROJECTS="$HOME/Projects/knowledge/10_projects"
if [ -d "$VAULT_PROJECTS" ]; then
    log_info "Deploying auto-memory symlinks from vault..."
    for project_dir in "$VAULT_PROJECTS"/*/; do
        [ -d "$project_dir" ] || continue
        memory_source="${project_dir}memory"
        [ -d "$memory_source" ] || continue

        project_name=$(basename "$project_dir")
        project_path="$HOME/Projects/$project_name"
        encoded_path=$(printf '%s' "$project_path" | sed 's|/|-|g')
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
    done

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
            cp -r "$memory_dir" "$vault_memory"
            mv "$memory_dir" "${memory_dir}.migrated.$(date +%s)"
            ln -s "$vault_memory" "$memory_dir"
            log_success "Migrated and linked: $project_name"
        fi
    done
fi

# Add project-init alias (AI-agnostic naming)
PROJECT_INIT_LINE='alias project-init="$HOME/.claude/init-project.sh"'
ensure_line_in_file "$HOME/.zshrc" "$PROJECT_INIT_LINE" 2>/dev/null || true
ensure_line_in_file "$HOME/.bashrc" "$PROJECT_INIT_LINE" 2>/dev/null || true

log_info "Adding dotfiles scripts directory to PATH..."
PATH_LINE='export PATH=$HOME/.dotfiles/scripts:$PATH'
ensure_line_in_file "$HOME/.zshrc" "$PATH_LINE" && log_success "Added scripts to PATH in .zshrc"
ensure_line_in_file "$HOME/.bashrc" "$PATH_LINE" && log_success "Added scripts to PATH in .bashrc"

# Test if files are correctly linked
log_success "Installation completed! Verifying file links..."

verify_symlink "$HOME/.zsh/aliases.zsh" "aliases.zsh"
verify_symlink "$HOME/.zsh/functions.zsh" "functions.zsh"
verify_symlink "$HOME/.zshrc" ".zshrc"
verify_symlink "$HOME/.bashrc" ".bashrc"
file_exists "$HOME/.bash/bash_aliases" && log_success "bash_aliases created" || log_error "bash_aliases issue"

# Check dependencies
check_dependencies "git" "zsh" "eza" "direnv" "node" "npm" "zoxide" "docker" "docker-compose" "kubectl" "helm" "terraform" "ansible" "pip"
log_info "To apply changes immediately, run:"
log_info "  - For Bash: source ~/.bashrc"
log_info "  - For Zsh:  source ~/.zshrc"
