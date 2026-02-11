#!/bin/bash

# setup-linux.sh: Install dotfiles and set up environment
# Usage: ./setup-linux.sh

set -e

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
    safe_copy "$CURRENT_DIR/.zshrc" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.profile" "$DOTFILES_DIR/" 2>/dev/null || true
    if [ -f "$CURRENT_DIR/.bashrc" ]; then
        safe_copy "$CURRENT_DIR/.bashrc" "$DOTFILES_DIR/" 2>/dev/null || true
    fi    
    cp -rf "$CURRENT_DIR/.zsh/"* "$DOTFILES_DIR/.zsh/" 2>/dev/null || true
    cp -rf "$CURRENT_DIR/scripts/"* "$DOTFILES_DIR/scripts/" 2>/dev/null || true
else
    log_info "Already in dotfiles directory, skipping copy..."
fi

# Create symbolic links to main dotfiles
log_info "Creating symbolic links for main dotfiles..."
ln -sf "$DOTFILES_DIR/.zshrc" "$HOME/.zshrc"
ln -sf "$DOTFILES_DIR/.profile" "$HOME/.profile" 2>/dev/null || true

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
chmod +x "$HOME/.claude/init-project.sh" 2>/dev/null || true
log_success "Claude Code configured with skills"

# Register MCP servers (requires Claude Code CLI and Node.js)
if command -v claude >/dev/null 2>&1 && command -v npx >/dev/null 2>&1; then
    log_info "Registering Claude Code MCP servers..."
    claude mcp add --transport stdio drawio --scope user -- npx -y @drawio/mcp 2>/dev/null || true
    claude mcp add --transport http socket --scope user -- https://mcp.socket.dev/ 2>/dev/null || true
    log_success "MCP servers registered"
else
    log_warning "Claude Code CLI or npx not found, skipping MCP server registration"
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