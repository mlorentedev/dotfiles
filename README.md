# Dotfiles - Modernized with GNU Stow

[![Tests](https://github.com/mlorentedev/dotfiles/workflows/Test%20Dotfiles/badge.svg)](https://github.com/mlorentedev/dotfiles/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Modern, tested, and automated dotfiles for DevOps engineers. Built with GNU Stow for modular configuration management.

## Features

- **Modular Structure**: GNU Stow-based modules for easy management
- **Automated Installation**: One-command setup with tool installation
- **Comprehensive Testing**: Unit, integration, and Docker-based tests
- **Secure Secrets**: Age encryption with smart wrapper and pre-commit hooks
- **DevOps Ready**: Pre-configured for Docker, Kubernetes, Terraform, Ansible
- **Cross-Platform**: Ubuntu 20.04+, Debian 10+, WSL2
- **Modern Shell**: Starship prompt, eza, bat, fzf, zoxide integrations
- **CI/CD**: GitHub Actions for automated testing

## Quick Start

```bash
# Clone repository
git clone https://github.com/mlorentedev/dotfiles ~/.dotfiles
cd ~/.dotfiles

# Minimal installation (just configs)
make minimal

# Or install with DevOps tools
make tools

# Or install everything
make all
```

## Installation Options

### Option 1: Using Makefile (Recommended)

```bash
make minimal        # Install configs only
make tools          # Install configs + DevOps tools
make all            # Install everything including secrets
make check          # Check dependencies
make test           # Run all tests
```

### Option 2: Using Bootstrap Script

```bash
./bootstrap.sh --minimal    # Configs only
./bootstrap.sh --tools      # Configs + tools
./bootstrap.sh --all        # Everything
```

### Option 3: Manual Stow

```bash
# Install GNU Stow
sudo apt-get install stow

# Stow individual modules
cd ~/.dotfiles
stow bash zsh git shell-common scripts
```

## What's Included

### Core Modules

- **bash/**: Modern Bash configuration with aliases and functions
- **zsh/**: Zsh with Oh My Zsh, plugins, and custom config
- **git/**: Git configuration and aliases
- **shell-common/**: Shared shell configuration
- **scripts/**: Utility scripts in `~/.local/bin`
- **starship/**: Modern cross-shell prompt

### DevOps Tools (Installed with `--tools`)

**Shell Enhancements:**
- eza (modern ls)
- bat (cat with syntax highlighting)
- fzf (fuzzy finder)
- ripgrep (fast grep)
- zoxide (smart cd)
- starship (prompt)
- direnv (environment switcher)
- age (encryption)

**Container Tools:**
- docker-compose
- lazydocker (Docker TUI)

**Kubernetes:**
- kubectl
- k9s (Kubernetes TUI)
- helm
- kubectx/kubens
- stern (log tailing)

**Infrastructure as Code:**
- terraform
- ansible
- ansible-lint

## Directory Structure

```
dotfiles/
├── bash/              # Bash configuration
│   ├── .bashrc
│   ├── .bash_profile
│   └── README.md
├── zsh/               # Zsh configuration
│   ├── .zshrc
│   ├── .zsh/
│   │   ├── aliases.zsh
│   │   ├── functions.zsh
│   │   └── nvm.zsh
│   └── README.md
├── git/               # Git configuration
│   ├── .gitconfig
│   └── README.md
├── shell-common/      # Common shell configs
│   ├── .profile
│   └── README.md
├── scripts/           # Utility scripts → ~/.local/bin
│   ├── .local/bin/
│   │   ├── utils
│   │   ├── age-encrypt-decrypt
│   │   ├── github-secrets-manager
│   │   ├── secrets-wrapper
│   │   └── install-precommit
│   └── README.md
├── starship/          # Starship prompt config
│   ├── .config/starship.toml
│   └── README.md
├── test/              # Test suite
│   ├── run-all-tests.sh
│   ├── test-stow.sh
│   ├── test-aliases.sh
│   ├── test-path.sh
│   └── docker/
│       ├── Dockerfile.ubuntu-20.04
│       ├── Dockerfile.ubuntu-22.04
│       └── Dockerfile.ubuntu-24.04
├── tools/             # Tool installation scripts
│   ├── install-shell.sh
│   ├── install-containers.sh
│   ├── install-kubernetes.sh
│   └── install-iac.sh
├── .github/workflows/
│   └── test.yml       # CI/CD pipeline
├── bootstrap.sh       # Main installation script
├── Makefile           # Common operations
├── README.md          # This file
└── MIGRATION.md       # Migration guide
```

## Usage Examples

### Shell Enhancements

```bash
# Smart cd with zoxide
z dotfiles          # Jump to frequently used directories

# Better ls with eza
ls                  # Auto-aliased to eza
ll                  # Long listing with git status
tree                # Tree view with eza

# Fuzzy finding with fzf
Ctrl+R              # Search command history
Ctrl+T              # Search files
Alt+C               # Search directories

# Syntax highlighting with bat
cat file.json       # Auto-aliased to bat
```

### DevOps Aliases

```bash
# Kubernetes
k get pods          # kubectl shorthand
k9s                 # Kubernetes TUI
kubectx             # Switch context
kubens              # Switch namespace

# Docker
d ps                # docker shorthand
dc up -d            # docker-compose shorthand
lazydocker          # Docker TUI

# Terraform
tf plan             # terraform shorthand
tf apply            #

# Ansible
a -m ping           # ansible shorthand
ap playbook.yml     # ansible-playbook shorthand
```

### Secrets Management

```bash
# Encrypt secrets
secrets-wrapper encrypt

# Decrypt secrets
secrets-wrapper decrypt

# Create backup
secrets-wrapper backup

# Validate (check for leaks)
secrets-wrapper validate

# Setup direnv integration
secrets-wrapper setup-direnv

# List secrets
secrets-wrapper list
```

## Testing

### Local Testing

```bash
# Run all tests
make test

# Test individual components
make test-stow
make test-aliases
make test-path

# Run shellcheck
make lint
```

### Docker Testing

```bash
# Test on Ubuntu 22.04
make test-docker

# Test on all Ubuntu versions (20.04, 22.04, 24.04)
make test-docker-all
```

### CI/CD

Tests run automatically on:
- Push to main/master/develop
- Pull requests
- Manual workflow dispatch

## Customization

### Local Overrides

Create these files for machine-specific configs (not tracked by git):

```bash
~/.bashrc.local     # Bash-specific
~/.zshrc.local      # Zsh-specific
~/.gitconfig.local  # Git-specific
```

### Adding Aliases

Edit `~/.zsh/aliases.zsh` or add to `~/.bashrc.local`:

```bash
alias myproject='cd ~/work/my-project'
alias deploy='./scripts/deploy.sh'
```

### Adding Functions

Edit `~/.zsh/functions.zsh`:

```bash
# Quick backup
backup() {
    cp "$1" "$1.backup-$(date +%Y%m%d)"
}
```

## Common Tasks

### Update Dotfiles

```bash
make update         # Pull latest and reinstall
```

### Backup Existing Config

```bash
make backup         # Backup to ~/.dotfiles-backup-TIMESTAMP
```

### Uninstall

```bash
make clean          # Remove symlinks
make uninstall      # Remove symlinks (alias for clean)
```

### Check Dependencies

```bash
make check          # Show installed/missing dependencies
```

## Troubleshooting

### Stow Conflicts

If stow reports conflicts:

```bash
# Backup and remove conflicting files
make backup
rm ~/.bashrc ~/.zshrc  # etc

# Then reinstall
make minimal
```

### Shell Not Loading Config

```bash
# For Bash
source ~/.bashrc

# For Zsh
source ~/.zshrc

# Or restart shell
exec $SHELL
```

### Path Not Updated

Add to your shell config:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then restart shell or source the config.

## Requirements

### Minimal

- Bash 4.0+ or Zsh 5.0+
- git
- curl or wget

### Recommended

- GNU Stow (auto-installed)
- zsh (for best experience)
- Python 3 + pip (for pre-commit)

### Optional

All optional tools can be installed with `make tools`:
- eza, bat, fzf, ripgrep, zoxide, starship, direnv, age
- docker, docker-compose, lazydocker
- kubectl, k9s, helm, kubectx, stern
- terraform, ansible

## Migration

Migrating from old dotfiles structure? See [MIGRATION.md](MIGRATION.md) for step-by-step guide.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

## CI/CD

The repository includes GitHub Actions workflows that:
- Run tests on Ubuntu 20.04, 22.04, and latest
- Test minimal installation
- Run shellcheck on all scripts
- Test in Docker containers
- Verify stow functionality

## License

MIT License - See [LICENSE](LICENSE) for details.

## Related Projects

- [Boilerplates](https://github.com/mlorentedev/boilerplates) - Project templates
- [Cheatsheets](https://github.com/mlorentedev/cheat-sheets) - Quick references

## Author

**Miguel Lorente**
- Website: [mlorente.dev](https://mlorente.dev)
- GitHub: [@mlorentedev](https://github.com/mlorentedev)
