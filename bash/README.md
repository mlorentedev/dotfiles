# Bash Module

Bash shell configuration with aliases, functions, and prompt customization.

## Features

- **Enhanced History**: Large history size with duplicate removal
- **Smart Completion**: Directory name auto-correction
- **Git-aware Prompt**: Shows current git branch
- **K8s Context**: Displays current kubectl context
- **Color Support**: Syntax highlighting for common commands
- **Tool Integration**: zoxide, direnv, fzf, starship
- **DevOps Aliases**: Quick shortcuts for kubectl, docker, terraform, etc.

## Installation

Using Stow from the dotfiles directory:
```bash
stow bash
```

Or via bootstrap script:
```bash
./bootstrap.sh --minimal
```

## Customization

Add local customizations to `~/.bashrc.local` (not tracked by git):
```bash
# Custom aliases
alias myproject='cd ~/work/my-project'

# Custom functions
myfunc() {
    echo "Hello from custom function"
}
```

## Requirements

- Bash 4.0+
- Optional: eza, bat, zoxide, direnv, fzf, starship

## Compatibility

- Ubuntu 20.04+ ✓
- Debian 10+ ✓
- WSL2 ✓
