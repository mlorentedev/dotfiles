# Scripts Module

Utility scripts for development and DevOps workflows.

## Installation

Using Stow (installs to `~/.local/bin`):
```bash
stow scripts
```

## Included Scripts

### `utils`
Common utility functions library. Provides:
- Colored logging (log_info, log_success, log_warning, log_error)
- File operations (backup_file, safe_copy, safe_move)
- Dependency checking
- Environment file loading
- Progress tracking

Used by other scripts. Can also be sourced in your own scripts:
```bash
source ~/.local/bin/utils
log_info "Starting task..."
```

### `age-encrypt-decrypt`
Encrypt/decrypt sensitive files using age encryption.

```bash
age-encrypt-decrypt encrypt              # Encrypt all *.secret files
age-encrypt-decrypt decrypt              # Decrypt all *.age files
age-encrypt-decrypt encrypt /path/dir    # Encrypt files in specific directory
```

**Requirements**: age, age key in `~/.config/age/key.txt`

### `github-secrets-manager`
Upload environment variables from .env files to GitHub repository secrets.

```bash
github-secrets-manager                   # Use default .env files
github-secrets-manager /path/to/.env     # Use specific file
```

**Requirements**: gh (GitHub CLI), authenticated

### `install-precommit`
Install and configure pre-commit hooks for git repositories.

```bash
install-precommit
```

**Requirements**: Python, pip, git repository

## Usage

After installation, all scripts are available in your PATH:
```bash
utils --help
age-encrypt-decrypt encrypt
github-secrets-manager
install-precommit
```

## Requirements

- Bash 4.0+
- Individual script requirements vary (see above)
