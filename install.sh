#!/bin/bash

# Directory where the dotfiles are located
DOTFILES_DIR=$(pwd)

# Create symbolic links to dotfiles in the home directory
ln -sf "$DOTFILES_DIR/.bashrc" "$HOME/.bashrc"
ln -sf "$DOTFILES_DIR/.zshrc" "$HOME/.zshrc"
ln -sf "$DOTFILES_DIR/.gitconfig" "$HOME/.gitconfig"
ln -sf "$DOTFILES_DIR/.profile" "$HOME/.profile"

echo "Dotfiles and configuration installed successfully!"
