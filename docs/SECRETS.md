# Secrets Management Guide

Complete guide for managing encrypted secrets in the dotfiles. Uses [`age`](https://github.com/FiloSottile/age) encryption with automatic environment variable loading.

## Overview

```text
sensitive/
├── env-mapping.conf     # Maps ENV_VAR=filename (no extension)
├── *.secret             # Original plaintext (local only, gitignored)
├── *.secret.age         # Encrypted versions (committed to git)
├── *.secret.dec         # Decrypted files (local only, gitignored)
└── .secrets-audit.log   # Audit trail of changes
```

## Quick Start

### Initial Setup

1. Create an age key pair:

     ```bash
     mkdir -p ~/.config/age
     age-keygen -o ~/.config/age/key.txt
     chmod 600 ~/.config/age/key.txt
     ```

2. Back up the key somewhere safe (password manager, encrypted USB).

3. Run the dotfiles installer:

     ```bash
     ./install.sh
     ```

4. Restart your terminal - secrets load automatically.

### Adding a New Secret

```bash
secrets_add API_KEY myservice.api
# Enter value when prompted
secrets_refresh  # or restart terminal
```

### Rotating a Secret

```bash
secrets_rotate API_KEY
# Enter new value when prompted
secrets_refresh
```

## Commands Reference

After sourcing `.zshrc`/`.bashrc`, these commands are available:

| Command | Description |
|---------|-------------|
| `secrets_help` | Show all commands and usage |
| `secrets_list` | Show mapped variables and load status |
| `secrets_refresh` | Force re-decrypt and reload all secrets |
| `secrets_get VAR` | Decrypt single secret on-demand |
| `secrets_add VAR FILE` | Add a new secret interactively |
| `secrets_rotate VAR` | Update an existing secret's value |
| `secrets_check` | Validate mapping integrity |
| `secrets_clean` | Remove plaintext .dec and .secret files |
| `secrets_audit [VAR]` | Show audit log of secret changes |

### Command Details

#### `secrets_list`

Shows all mapped secrets and their current status:

```text
Secret mappings (/path/to/sensitive/env-mapping.conf):

  * GITHUB_PERSONAL_ACCESS_TOKEN     # loaded
  o DOCKERHUB_TOKEN                  # not loaded (file exists)
  x MISSING_TOKEN (missing: file.secret.age)  # file not found
```

#### `secrets_check`

Validates the integrity of your secrets setup:

```bash
secrets_check
# Output:
# Checking secrets integrity...
#   ✓ GITHUB_PERSONAL_ACCESS_TOKEN
#   ✗ BROKEN_VAR (missing: nonexistent.secret.age)
#
# Checking for orphaned .age files...
#   ? orphan.secret.age (no mapping)
#
# Summary:
#   Valid:    5
#   Missing:  1
#   Orphaned: 1
```

#### `secrets_clean`

Removes plaintext files that shouldn't be kept:

```bash
secrets_clean --dry-run  # preview what would be deleted
secrets_clean            # actually delete
```

#### `secrets_audit`

Shows when secrets were added or rotated:

```bash
secrets_audit                     # all entries
secrets_audit GITHUB_TOKEN        # specific variable
```

## Mapping File Format

The `sensitive/env-mapping.conf` file maps environment variables to encrypted files:

```conf
# Format: ENV_VAR_NAME=filename (without .secret.age extension)
GITHUB_PERSONAL_ACCESS_TOKEN=github.token
DOCKERHUB_TOKEN=dockerhub.token
CLOUDFLARE_API_TOKEN=cloudflare.api-token

# Comments start with #
# Blank lines are ignored
```

Each line maps an environment variable to a file in `sensitive/`:

- `GITHUB_PERSONAL_ACCESS_TOKEN=github.token` means:
  - File: `sensitive/github.token.secret.age`
  - Variable: `$GITHUB_PERSONAL_ACCESS_TOKEN`

## Manual Encryption

### Encrypt/Decrypt Scripts

```bash
# Encrypt all *.secret files in sensitive/
age-encrypt-decrypt.sh encrypt

# Decrypt all *.age files
age-encrypt-decrypt.sh decrypt

# Work with a different directory
age-encrypt-decrypt.sh encrypt /path/to/directory
```

### Direct age Commands

```bash
# Get your public key
age-keygen -y ~/.config/age/key.txt

# Encrypt one file
age -r $(age-keygen -y ~/.config/age/key.txt) -o file.secret.age file.secret

# Decrypt one file
age -d -i ~/.config/age/key.txt -o file.secret.dec file.secret.age
```

## GitHub Secrets Sync

Upload secrets to GitHub repository:

```bash
# Upload from .env file
github-secrets-manager.sh /path/to/.env

# Uses current directory's .env by default
github-secrets-manager.sh
```

**Requires:** `gh` CLI authenticated (`gh auth login`)

## Setting Up on a New Machine

1. Install `age`:

```bash
brew install age          # macOS
sudo apt install age      # Ubuntu
```

1. Copy your private key:

```bash
mkdir -p ~/.config/age
# Copy key.txt from backup
chmod 600 ~/.config/age/key.txt
```

1. Clone and install dotfiles:

```bash
git clone https://github.com/mlorentedev/dotfiles.git ~/.dotfiles
cd ~/.dotfiles
./install.sh
```

1. Secrets load automatically on terminal start.

## Security Best Practices

- Never commit `.secret` or `.dec` files (gitignored by default)
- Encrypt files before committing changes
- Back up your age key in multiple secure locations
- Use `secrets_clean` after working with plaintext files
- Review `secrets_audit` periodically
- Rotate secrets if you suspect compromise

## Troubleshooting

### Secrets not loading

1. Check key exists: `ls -la ~/.config/age/key.txt`
2. Check permissions: `chmod 600 ~/.config/age/key.txt`
3. Check mapping: `secrets_list`
4. Check integrity: `secrets_check`

### Bad substitution error

This indicates a shell compatibility issue. Ensure you're using:

- Bash 4+ or Zsh 5+

### Missing age command

Install age. See [TOOLS.md](TOOLS.md#age-encryption) for installation instructions.

### Key not found

The default key path is `~/.config/age/key.txt`. Override with:

```bash
export AGE_KEY_PATH=/custom/path/to/key.txt
```

## Architecture

### File Loading Flow

```text
Terminal Start
     │
     ▼
source .zshrc/.bashrc
     │
     ▼
source load-secrets.sh
     │
     ▼
secrets_load()
     │
     ├─► Read env-mapping.conf
     │
     ├─► For each mapping:
     │      └─► age decrypt → export ENV_VAR
     │
     └─► Secrets available as env vars
```

### Files Involved

| File | Purpose |
|------|---------|
| `scripts/load-secrets.sh` | Main secrets management script |
| `scripts/utils.sh` | Shared utility functions |
| `sensitive/env-mapping.conf` | Variable-to-file mapping |
| `sensitive/*.secret.age` | Encrypted secret files |
| `~/.config/age/key.txt` | Age private key |
