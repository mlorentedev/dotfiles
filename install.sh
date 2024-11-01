#!/bin/bash

DOTFILES_DIR=pwd

# Crear enlaces simbólicos para los dotfiles
ln -sf "$DOTFILES_DIR/.bashrc" "$HOME/.bashrc"
ln -sf "$DOTFILES_DIR/.zshrc" "$HOME/.zshrc"
ln -sf "$DOTFILES_DIR/.gitconfig" "$HOME/.gitconfig"

echo "Dotfiles installed successfully!"
