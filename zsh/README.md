# Zsh Module

Modern Zsh configuration with Oh My Zsh, custom aliases, functions, and tool integrations.

## Features

- **Oh My Zsh**: Pre-configured with useful plugins
- **Enhanced History**: Large history with deduplication and sharing
- **Smart Completion**: Context-aware completions for kubectl, helm, etc.
- **Custom Prompt**: Shows git branch, k8s context, terraform workspace
- **Tool Integration**: direnv, zoxide, fzf, starship
- **DevOps Aliases**: Shortcuts for daily DevOps tasks
- **Auto Correction**: Typo correction for commands and directories

## Included Plugins

- git - Git aliases and functions
- docker - Docker command completion
- kubectl - Kubernetes completion
- terraform - Terraform completion
- ansible - Ansible completion
- colored-man-pages - Colorized man pages
- command-not-found - Suggests packages for missing commands
- sudo - Press ESC twice to prepend sudo
- history-substring-search - Better history search

## Installation

Using Stow from the dotfiles directory:
```bash
stow zsh
```

Or via bootstrap script:
```bash
./bootstrap.sh --minimal
```

## File Structure

- `.zshrc` - Main zsh configuration
- `.zsh/aliases.zsh` - Custom command aliases
- `.zsh/functions.zsh` - Custom shell functions
- `.zsh/nvm.zsh` - Node version manager setup

## Customization

Add local customizations to `~/.zshrc.local` (not tracked by git):
```zsh
# Custom environment variables
export MY_VAR="value"

# Custom aliases
alias myproject='cd ~/work/my-project'

# Custom functions
myfunc() {
    echo "Hello from custom function"
}
```

## Requirements

- Zsh 5.0+
- Optional: Oh My Zsh (auto-installed by bootstrap)
- Optional: eza, bat, zoxide, direnv, fzf, starship

## Compatibility

- Ubuntu 20.04+ ✓
- Debian 10+ ✓
- WSL2 ✓
