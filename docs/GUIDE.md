# Dotfiles Guide

Complete guide for using and customizing the dotfiles.

## Features

### Shell Improvements

- **Aliases** for commands I use constantly
- **Smart directory jumping** with zoxide  
- **Automatic environment loading** with direnv
- **Decent prompts** for both shells
- **Colors everywhere**

### Development Setup

- **Java, Maven, Python, Ruby, Go paths**
- **Docker and Kubernetes shortcuts**
- **Quick commands** for Terraform, Ansible, Helm
- **Node version management**
- **GitHub CLI integration**

### Security Features

- **Encrypt/decrypt files** with age
- **Upload .env files** to GitHub secrets
- **Sensitive files** automatically gitignored

## Scripts

All scripts in `scripts/` are automatically added to your PATH.

### `utils.sh`

Common functions library with colored output, path helpers, dependency checking, safe file operations, and progress tracking.

### `age-encrypt-decrypt.sh`

Encrypt/decrypt files using age.

```bash
age-encrypt-decrypt.sh encrypt    # encrypt all *.secret files
age-encrypt-decrypt.sh decrypt    # decrypt all *.age files
age-encrypt-decrypt.sh encrypt /some/path
```

**Requires:** `age` and private key at `~/.config/age/key.txt`

### `github-secrets-manager.sh`

Upload .env files to GitHub repository secrets.

```bash
github-secrets-manager.sh                 # use default .env
github-secrets-manager.sh /path/to/.env   # use specific file
```

**Requires:** `gh` (GitHub CLI) and authentication

### `install-precommit.sh`

Install pre-commit hooks for current repository.

```bash
install-precommit.sh
```

**Requires:** Python with pip and git repository

### `install.sh`

Main installer - creates directories, symlinks, sets up shells, and configures AI tools.

```bash
./install.sh
```

## Customization

### Aliases

Edit `~/.zsh/aliases.zsh` or add to shell config:

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

Add to `.zshrc`/`.bashrc` for shell-specific, or `.profile` for global.

### AI Configuration

Both Gemini and Claude CLIs are configured with aliases and custom prompts.

**Gemini:**

- Alias: `g` → `gemini`
- Prompt function: `gp <prompt-name> <args>`
- Config: `~/.gemini/GEMINI.md`, `~/.gemini/settings.json`
- Prompts: `~/.gemini/prompts/`

**Claude:**

- Alias: `c` → `claude`
- Prompt function: `cp <prompt-name> <args>`
- Config: `~/.claude/CLAUDE.md`, `~/.claude/settings.json`
- Prompts: `~/.claude/prompts/`

**Using custom prompts:**

1. Create a markdown file in `~/.gemini/prompts/` or `~/.claude/prompts/`
2. Invoke with the prompt function:

```bash
gp architect "design a microservices system"
cp architect "design a microservices system"
```

The prompt file content is prepended as system instructions.
