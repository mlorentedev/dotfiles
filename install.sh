#!/bin/bash

# install.sh: Install dotfiles and set up environment
# Usage: ./install.sh

set -e

# Load utility functions for logging
if [ -f ./scripts/utils.sh ]; then
    source ./scripts/utils.sh
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


# Update functions.zsh to source utils.sh if not already done
if [ -f "$DOTFILES_DIR/.zsh/functions.zsh" ]; then
    if ! grep -q "source.*utils.sh" "$DOTFILES_DIR/.zsh/functions.zsh"; then
        log_info "Updating functions.zsh to source utils.sh..."
        echo -e "\n# Source utility functions\nsource $DOTFILES_DIR/scripts/utils.sh" >> "$DOTFILES_DIR/.zsh/functions.zsh"
    fi
    log_info "functions.zsh already exists, skipping creation..."
else
    log_info "Creating functions.zsh file..."
    cat > "$DOTFILES_DIR/.zsh/functions.zsh" << EOF

# Source the utils.sh and setup-gh-secrets.sh scripts
source "$DOTFILES_DIR/scripts/utils.sh"

EOF
fi

# Create a bash_aliases file for bash
log_info "Creating bash_aliases file..."
ensure_directory "$HOME/.bash"
cat > "$HOME/.bash/bash_aliases" << EOF
# Bash Aliases
$(grep "alias" "$DOTFILES_DIR/.zsh/aliases.zsh" | sed 's/alias /alias /g')
EOF

# Create a basic .bashrc if it doesn't exist
if [ ! -f "$DOTFILES_DIR/.bashrc" ]; then
    log_info "Creating a basic .bashrc file..."
    cat > "$DOTFILES_DIR/.bashrc" << EOF
# ~/.bashrc: executed by bash(1) for non-login shells.

# Source aliases
if [ -f ~/.bash/bash_aliases ]; then
    source ~/.bash/bash_aliases
fi

# Source utility functions
if [ -f "$DOTFILES_DIR/scripts/utils.sh" ]; then
    source "$DOTFILES_DIR/scripts/utils.sh"
fi
EOF
fi

# Link .bashrc
ln -sf "$DOTFILES_DIR/.bashrc" "$HOME/.bashrc"

log_info "Adding dotfiles scripts directory to PATH..."
if ! grep -q "export PATH=\$DOTFILES_DIR/scripts:\$PATH" "$HOME/.zshrc"; then
    echo "export PATH=\$HOME/.dotfiles/scripts:\$PATH" >> "$HOME/.zshrc"
fi
if ! grep -q "export PATH=\$DOTFILES_DIR/scripts:\$PATH" "$HOME/.bashrc"; then
    echo "export PATH=\$HOME/.dotfiles/scripts:\$PATH" >> "$HOME/.bashrc"
fi

# Test if files are correctly linked
log_success "Installation completed! Verifying file links..."
if [ -f "$HOME/.zsh/aliases.zsh" ]; then
    log_success "aliases.zsh linked successfully!"
else
    log_error "aliases.zsh not linked correctly!"
fi

if [ -f "$HOME/.bash/bash_aliases" ]; then
    log_success "bash_aliases created successfully!"
else
    log_error "bash_aliases not created correctly!"
fi

if [ -f "$HOME/.zsh/functions.zsh" ]; then
    log_success "functions.zsh linked successfully!"
else
    log_error "functions.zsh not linked correctly!"
fi

# Check dependencies
check_dependencies "git" "zsh" "eza" "direnv" "node" "npm" "zoxide" "docker" "docker-compose" "kubectl" "helm" "terraform" "ansible" "pip"
log_info "To apply changes immediately, run:"
log_info "  - For Bash: source ~/.bashrc"
log_info "  - For Zsh:  source ~/.zshrc"