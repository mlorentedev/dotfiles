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

- **Automatic secrets loading** from encrypted age files
- **Encrypt/decrypt files** with age
- **Upload secrets** to GitHub repositories
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

### `load-secrets.sh`

Manages encrypted secrets as environment variables. Sourced automatically.

```bash
secrets_list      # Show all secrets and status
secrets_add VAR FILE   # Add new secret
secrets_rotate VAR     # Update secret value
secrets_check          # Validate integrity
secrets_clean          # Remove plaintext files
```

**Requires:** `age` and private key at `~/.config/age/key.txt`

See [SECRETS.md](SECRETS.md) for full documentation.

### `install-precommit.sh`

Install pre-commit hooks for current repository.

```bash
install-precommit.sh
```

**Requires:** Python with pip and git repository

### `test.sh`

Run the test suite for all utility functions.

```bash
./scripts/test.sh
```

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

Both Gemini and Claude CLIs are configured with aliases and custom skills.

**Gemini:**

- Alias: `g` → `gemini`
- Prompt function: `gp <skill-name> <args>`
- Config: `~/.gemini/GEMINI.md`
- Skills: `~/.gemini/prompts/` (flat markdown, YAML stripped)

**Claude:**

- Alias: `c` → `claude`
- Skills: `~/.claude/skills/` (SKILL.md with YAML frontmatter)
- Config: `~/.claude/CLAUDE.md`
- Init script: `~/.claude/init-project.sh`

**Using skills:**

```bash
# Claude Code - use slash commands in session
claude
> /audit src/auth.py
> /refactor this function

# Gemini - use gp function
gp audit "$(cat src/auth.py)"
gp refactor "$(cat src/utils.py)"
```

See [AI.md](AI.md) for full documentation.
