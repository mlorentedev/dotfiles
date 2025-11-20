# Dotfiles

My personal configuration files for development tools and shell environments. I use these across different machines to keep my setup consistent. They work with both Bash and Zsh, include aliases for common tasks, and have some handy scripts for managing secrets and GitHub repositories.

## Quick Start

```bash
git clone https://github.com/mlorentedev/dotfiles.git ~/.dotfiles
cd ~/.dotfiles
chmod +x install.sh
./install.sh
```

After installing, restart your shell or run:

```bash
source ~/.zshrc  # for Zsh
source ~/.bashrc # for Bash
```

## Prerequisites

- `git`
- `curl` or `wget`
- `bash` or `zsh`

## What's Inside

```text
├── .bashrc              # Bash configuration
├── .zshrc               # Zsh configuration  
├── .profile             # Profile settings
├── .gitconfig           # Git setup
├── .gitignore           # What to ignore
├── .zsh/                # Zsh files
│   ├── aliases.zsh      # Command shortcuts
│   ├── functions.zsh    # Custom functions
│   └── nvm.zsh          # Node version management
├── ai/                  # AI configuration
│   ├── gemini/          # Gemini settings
│   ├── claude/          # Claude settings
│   └── prompts/         # Custom prompts
├── docs/                # Documentation
├── scripts/             # Helper scripts
│   ├── utils.sh                    # Shared functions
│   ├── age-encrypt-decrypt.sh      # File encryption
│   ├── github-secrets-manager.sh   # GitHub secrets
│   └── install-precommit.sh        # Git hooks
├── sensitive/           # Encrypted files
└── install.sh           # Installer
```

## Documentation

- **[GUIDE.md](docs/GUIDE.md)** - Complete usage and customization guide
- **[TOOLS.md](docs/TOOLS.md)** - Tool installation instructions

## Features

Shell improvements, development setup, security features, and AI integration. See [GUIDE.md](docs/GUIDE.md) for details.

## Tools

**Required:**

- Git
- Bash 4+ or Zsh 5+

**Recommended:**

- Oh My Zsh, eza, zoxide, direnv, age

See [TOOLS.md](docs/TOOLS.md) for installation instructions.

## Security

- Use `age-encrypt-decrypt.sh` to encrypt sensitive files
- Use `github-secrets-manager.sh` to sync secrets to GitHub
- Sensitive files are automatically gitignored

## Contributing

Fork it, make changes, submit a pull request.

## Related Projects

- [Boilerplates](https://github.com/mlorentedev/boilerplates) - Project templates
- [Cheatsheets](https://github.com/mlorentedev/cheat-sheets) - Quick references  
- [My list](https://mlorente.dev) - More detailed explanations

## License

MIT License - use it however you want.
