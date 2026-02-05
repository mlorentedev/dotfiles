# Secrets Management Guide

Complete guide for managing encrypted secrets in the dotfiles. Uses [`age`](https://github.com/FiloSottile/age) encryption with automatic environment variable loading.

## Architecture

```text
~/.dotfiles/                    # Local installation (stable)
~/Projects/dotfiles/            # Development repo

Both directories sync via:
- dotfiles-sync                 # Manual full sync
- secrets_add/secrets_rotate    # Auto-sync on secret changes
```

### File Structure

```text
sensitive/
├── env-mapping.conf        # Maps ENV_VAR=filename
├── *.secret.age            # Encrypted secrets (git tracked)
├── *.secret                # Plaintext (gitignored, temporary)
├── *.secret.dec            # Decrypted cache (gitignored)
└── .secrets-audit.log      # Audit trail
```

## Quick Reference

### Add a New Secret

```bash
secrets_add VAR_NAME filename
# Example: secrets_add PYPI_TOKEN pypi.token
# Enter value when prompted
# Auto-syncs to repo if DOTFILES_REPO_DIR is set
```

### Rotate an Existing Secret

```bash
secrets_rotate VAR_NAME
# Example: secrets_rotate GITHUB_PERSONAL_ACCESS_TOKEN
```

### Sync Changes Between Local and Repo

```bash
dotfiles-sync              # Full bidirectional sync + git push/pull
dotfiles-sync --secrets-only  # Only sync sensitive/ files
```

### Upload to GitHub Secrets

```bash
# From env-mapping.conf (recommended)
github-secrets-manager.sh --from-mapping

# Specific secrets only
github-secrets-manager.sh --from-mapping --select GITHUB_PERSONAL_ACCESS_TOKEN DOCKERHUB_TOKEN

# List available secrets
github-secrets-manager.sh --list
```

## Initial Setup

### 1. Create Age Key

```bash
mkdir -p ~/.config/age
age-keygen -o ~/.config/age/key.txt
chmod 600 ~/.config/age/key.txt
```

Back up `~/.config/age/key.txt` somewhere safe.

### 2. Configure Shell

Add to `~/.zshrc` or `~/.bashrc`:

```bash
export DOTFILES_DIR="$HOME/.dotfiles"
export DOTFILES_REPO_DIR="$HOME/Projects/dotfiles"  # For repo sync
```

### 3. Install Dotfiles

```bash
git clone https://github.com/mlorentedev/dotfiles.git ~/.dotfiles
cd ~/.dotfiles
./setup-linux.sh
source ~/.zshrc
```

### 4. Clone Repo for Development (Optional)

```bash
git clone https://github.com/mlorentedev/dotfiles.git ~/Projects/dotfiles
```

## Commands Reference

### Secret Management

| Command | Description |
|---------|-------------|
| `secrets_add VAR FILE` | Add new secret interactively |
| `secrets_rotate VAR` | Update existing secret value |
| `secrets_get VAR` | Decrypt and show single secret |
| `secrets_list` | Show all secrets and their status |
| `secrets_check` | Validate mapping integrity |
| `secrets_clean` | Remove plaintext files |
| `secrets_audit [VAR]` | Show audit log |
| `secrets_refresh` | Reload all secrets |
| `secrets_help` | Show help |
| `secrets_sync` | Sync secrets to repo |

### Dotfiles Sync

| Command | Description |
|---------|-------------|
| `dotfiles-sync` | Full sync: secrets + git push/pull |
| `dotfiles-sync --secrets-only` | Only sync sensitive/ directory |

### GitHub Secrets

| Command | Description |
|---------|-------------|
| `github-secrets-manager.sh --list` | List secrets in env-mapping.conf |
| `github-secrets-manager.sh --from-mapping` | Upload all to GitHub |
| `github-secrets-manager.sh --from-mapping --select VAR1 VAR2` | Upload specific |
| `github-secrets-manager.sh /path/to/.env` | Upload from .env file |

## Workflow: Adding a New Secret

**Step-by-step deterministic workflow:**

```bash
# 1. Add the secret (auto-syncs to repo)
secrets_add MYSERVICE_TOKEN myservice.api
# Enter value when prompted

# 2. Verify it was added
secrets_list | grep MYSERVICE

# 3. Load it in current session
secrets_refresh
# Or restart terminal

# 4. Verify it's loaded
echo $MYSERVICE_TOKEN

# 5. Commit and sync to remote
cd ~/Projects/dotfiles
git add sensitive/myservice.api.secret.age sensitive/env-mapping.conf
git commit -m "feat: add MYSERVICE_TOKEN secret"
dotfiles-sync
```

## Workflow: Syncing After Changes

When you modify files directly in either location:

```bash
# Sync everything (bidirectional for secrets, push/pull for git)
dotfiles-sync

# What happens:
# 1. Compares timestamps of .age files, env-mapping.conf, audit log
# 2. Copies newer files in both directions
# 3. git push from ~/Projects/dotfiles
# 4. git pull to ~/.dotfiles
```

## Workflow: Upload to GitHub Repository Secrets

```bash
# Navigate to your project
cd ~/my-project

# List available secrets from dotfiles
github-secrets-manager.sh --list

# Upload specific secrets needed for this project
github-secrets-manager.sh --from-mapping --select GITHUB_PERSONAL_ACCESS_TOKEN DOCKERHUB_TOKEN

# Or upload all (be careful)
github-secrets-manager.sh --from-mapping
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DOTFILES_DIR` | `~/.dotfiles` | Local installation directory |
| `DOTFILES_REPO_DIR` | (none) | Repo directory for sync |
| `AGE_KEY_PATH` | `~/.config/age/key.txt` | Age private key location |

## Mapping File Format

`sensitive/env-mapping.conf`:

```conf
# Format: ENV_VAR_NAME=filename (without .secret.age extension)
GITHUB_PERSONAL_ACCESS_TOKEN=github.token
DOCKERHUB_TOKEN=dockerhub.token
PYPI_TOKEN=pypi.token
```

Each line maps:
- `GITHUB_PERSONAL_ACCESS_TOKEN` = environment variable name
- `github.token` = file `sensitive/github.token.secret.age`

## Testing

Run the test suite to verify everything works:

```bash
./scripts/test.sh
```

### Quick Manual Tests

```bash
# 1. Test secrets loading
secrets_list

# 2. Test secrets check
secrets_check

# 3. Test sync (dry run - just check output)
dotfiles-sync --secrets-only

# 4. Test GitHub secrets list
github-secrets-manager.sh --list

# 5. Test add/rotate (use a test secret)
secrets_add TEST_SECRET test.secret
secrets_rotate TEST_SECRET
# Then clean up:
rm ~/.dotfiles/sensitive/test.secret.secret.age
rm ~/Projects/dotfiles/sensitive/test.secret.secret.age
# Remove from env-mapping.conf manually
```

## Troubleshooting

### Secret not loading

```bash
# Check the secret exists and is mapped
secrets_list | grep VAR_NAME
secrets_check

# Check key is accessible
ls -la ~/.config/age/key.txt
age-keygen -y ~/.config/age/key.txt  # Should show public key
```

### Sync not working

```bash
# Verify both directories exist
ls -la ~/.dotfiles/sensitive/
ls -la ~/Projects/dotfiles/sensitive/

# Check DOTFILES_REPO_DIR is set
echo $DOTFILES_REPO_DIR
```

### GitHub upload failing

```bash
# Check authentication
gh auth status

# Check you're in a git repo
gh repo view

# Try listing first
github-secrets-manager.sh --list
```

### Key not found

```bash
# Check default location
ls -la ~/.config/age/key.txt

# Or set custom location
export AGE_KEY_PATH=/path/to/key.txt
```

## Security Best Practices

1. **Never commit plaintext** - `.secret` and `.dec` files are gitignored
2. **Back up your age key** - Store in password manager or encrypted backup
3. **Use `secrets_clean`** - Remove decrypted files after use
4. **Review `secrets_audit`** - Check for unexpected changes
5. **Rotate compromised secrets** - Use `secrets_rotate` immediately
6. **Limit GitHub uploads** - Use `--select` to upload only needed secrets
7. **USB backup monthly** - Run `backup-secrets-to-usb.sh` at least once a month

## Physical Backup (USB + VeraCrypt)

Keep an offline backup of your secrets on an encrypted USB drive.

### Initial Setup (one-time)

#### 1. Install VeraCrypt

```bash
# Ubuntu/Debian
sudo add-apt-repository ppa:unit193/encryption
sudo apt update
sudo apt install veracrypt

# Or download from https://veracrypt.fr/en/Downloads.html
```

#### 2. Format USB with VeraCrypt

1. Open VeraCrypt GUI: `veracrypt`
2. **Tools → Volume Creation Wizard**
3. **Create a volume within a partition/drive** → Next
4. **Standard VeraCrypt volume** → Next
5. **Select Device** → choose your USB (e.g., `/dev/sdb1`)
6. Encryption: **AES** + **SHA-512** (defaults) → Next
7. Set a strong **password** → Next
8. Filesystem: **FAT** (max compatibility with Windows/Linux) → Next
9. Move mouse for entropy → **Format**

#### 3. Copy age key to USB

```bash
# Mount the encrypted USB
veracrypt /dev/sdb1 /media/veracrypt1

# Copy your private key
cp ~/.config/age/key.txt /media/veracrypt1/
```

### Regular Backup

**Run this monthly** (or after adding important secrets):

```bash
# 1. Mount encrypted USB
veracrypt /dev/sdb1 /media/veracrypt1

# 2. Backup secrets
bash ~/Projects/dotfiles/scripts/backup-secrets-to-usb.sh /media/veracrypt1

# 3. Dismount
veracrypt -d /media/veracrypt1
```

### USB Contents After Backup

```text
/media/veracrypt1/
├── key.txt              # Age private key
├── age-standalone.sh    # Standalone decrypt script
└── secrets/             # All .secret and .secret.age files
```

### Restore on Any Linux System

```bash
# 1. Install required tools
sudo apt install age veracrypt

# 2. Mount USB
sudo mkdir -p /media/secrets
veracrypt /dev/sdX1 /media/secrets

# 3. Decrypt secrets
cd /media/secrets
./age-standalone.sh decrypt

# 4. Your secrets are now in secrets/*.dec
```

### VeraCrypt Quick Reference

| Action | Command |
|--------|---------|
| Mount | `veracrypt /dev/sdX1 /media/veracrypt1` |
| Dismount | `veracrypt -d /media/veracrypt1` |
| Dismount all | `veracrypt -d` |
| List mounted | `veracrypt -l` |

### Windows Compatibility

1. Download VeraCrypt from https://veracrypt.fr
2. Install or use portable version
3. Select Device → Mount → Enter password
4. USB appears as a new drive letter

## Files Reference

| File | Purpose | Git Tracked |
|------|---------|-------------|
| `sensitive/env-mapping.conf` | Variable-to-file mapping | Yes |
| `sensitive/*.secret.age` | Encrypted secrets | Yes |
| `sensitive/*.secret` | Plaintext (temporary) | No |
| `sensitive/*.secret.dec` | Decrypted cache | No |
| `sensitive/.secrets-audit.log` | Audit trail | Yes |
| `~/.config/age/key.txt` | Age private key | No (never!) |
