# Dotfiles

My personal configuration files for development tools and shell environments. I use these across different machines to keep my setup consistent. They work with both Bash and Zsh, include aliases for common tasks, and have some handy scripts for managing secrets and GitHub repositories.

## Getting Started

You'll need these basics before installing:

- `git`
- `curl` or `wget`
- `bash` or `zsh`

### Installation

```bash
git clone https://github.com/mlorentedev/dotfiles.git ~/.dotfiles
cd ~/.dotfiles
chmod +x install.sh
./install.sh
```

This copies everything to `~/.dotfiles`, creates symlinks to your home directory, sets up both shell configs, and makes the scripts executable.

After installing, restart your shell or run:

```bash
# Zsh
source ~/.zshrc

# Bash
source ~/.bashrc
```

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
├── scripts/             # Helper scripts
│   ├── utils.sh                    # Shared functions
│   ├── age-encrypt-decrypt.sh      # File encryption
│   ├── github-secrets-manager.sh   # GitHub secrets
│   └── install-precommit.sh        # Git hooks
├── sensitive/           # Encrypted files
└── install.sh           # Installer
```

## Tools

I use quite a few tools in my daily workflow. Check [TOOLS.md](TOOLS.md) for installation instructions.

**Required:**

- Git
- Bash 4+ or Zsh 5+

**Nice to have:**

- Oh My Zsh (themes and plugins)
- eza (better `ls` with colors)
- zoxide (smart `cd`)
- direnv (per-directory env vars)
- age (file encryption)

## What It Does

### Shell Improvements

- Aliases for commands I use constantly
- Smart directory jumping with zoxide  
- Automatic environment loading with direnv
- Decent prompts for both shells
- Colors everywhere

### Development Setup

- Java, Maven, Python, Ruby, Go paths
- Docker and Kubernetes shortcuts
- Quick commands for Terraform, Ansible, Helm
- Node version management
- GitHub CLI integration

### Security Stuff  

- Encrypt/decrypt files with age
- Upload .env files to GitHub secrets
- Keeps sensitive files out of git

### Helper Scripts

- Install pre-commit hooks
- Debug PATH issues
- Show terminal colors
- Safe file operations

## Scripts

The scripts in `scripts/` get added to your PATH automatically:

### `utils.sh`

Library of common functions that other scripts use. Has colored output, path helpers, dependency checking, safe file operations, and progress tracking. Gets loaded automatically by other scripts.

### `age-encrypt-decrypt.sh`

Encrypts and decrypts files using the age tool. Takes all `*.secret` files and makes `*.secret.age` versions, or does the reverse.

```bash
age-encrypt-decrypt.sh encrypt    # encrypt all *.secret files
age-encrypt-decrypt.sh decrypt    # decrypt all *.age files
age-encrypt-decrypt.sh encrypt /some/path  # work with different directory
```

Needs `age` installed and a private key at `~/.config/age/key.txt`.

### `github-secrets-manager.sh`

Reads .env files and uploads the variables as GitHub repository secrets. Handles multiple env files, decodes base64 SSH keys automatically, and skips duplicates.

```bash
github-secrets-manager.sh                 # use default .env files
github-secrets-manager.sh /path/to/.env   # use specific file
```

Needs `gh` (GitHub CLI) and you need to be logged in.

### `install-precommit.sh`

Installs pre-commit hooks for the current git repository. Installs the pre-commit tool if needed, then sets up the hooks.

```bash
install-precommit.sh
```

Needs Python with pip and a git repository.

### `install.sh`

The main installer. Creates directories, copies files, makes symlinks, sets up both shells, adds scripts to PATH, and checks everything worked.

```bash
./install.sh
```

What it does:

1. Creates `~/.dotfiles` structure
2. Copies config files  
3. Makes symlinks to home directory
4. Sets up shell configs
5. Adds scripts to PATH
6. Checks everything

## Customizing

### Aliases

Edit `~/.zsh/aliases.zsh` or add to your shell config:

```bash
alias ll="ls -la"
alias myproject="cd ~/work/important-project"
```

### Functions  

Add to `~/.zsh/functions.zsh`:

```bash
function backup() {
    cp "$1" "$1.backup-$(date +%Y%m%d)"
}
```

### Environment Variables

Add to `.zshrc`/`.bashrc` for shell-specific stuff, or `.profile` for everything.

## Security

- Use `age-encrypt-decrypt.sh` to encrypt sensitive files
- Use `github-secrets-manager.sh` to sync secrets to GitHub
- Sensitive files are automatically gitignored

## Contributing

Fork it, make changes, submit a pull request. Pretty standard stuff.

## More Stuff

- [Boilerplates](https://github.com/mlorentedev/boilerplates) - Project templates
- [Cheatsheets](https://github.com/mlorentedev/cheat-sheets) - Quick references  
- [My list](https://mlorente.dev) - More detailed explanations

## License

MIT License - use it however you want.
