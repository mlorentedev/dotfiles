# Starship Prompt Module

Modern, fast, and customizable shell prompt.

## Features

- Git status and branch
- Kubernetes context and namespace
- Terraform workspace
- Docker context
- Language versions (Python, Node, Rust, Go)
- Command duration
- Exit status indicator

## Installation

Using Stow:
```bash
stow starship
```

Or via bootstrap script (installs starship + config):
```bash
./bootstrap.sh --tools
```

## Requirements

- Starship prompt (`bootstrap.sh --tools` will install it)
- A Nerd Font for icons (optional but recommended)

## Customization

Edit `~/.config/starship.toml` to customize the prompt.
See https://starship.rs/config/ for all options.
