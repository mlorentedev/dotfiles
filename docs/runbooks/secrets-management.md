---
id: "dotfiles-runbook-secrets-management"
type: runbook
status: active
tags: [runbook, dotfiles, secrets, age]
created: "2026-02-22"
owner: manu
---

# Secrets Management

> ⚠️ **Out of date — pending rewrite ([#600](https://github.com/mlorentedev/dotfiles/issues/600)).**
> The `sensitive/env-mapping.conf` workflow below was retired in #587. Secrets are now
> mapped in **`secrets/registry.yaml`** (ADR-028) and accessed via `dotf secrets {run,show,render}`.
> Treat the `env-mapping.conf` steps here as historical until this runbook is refreshed.

Complete guide for managing encrypted secrets in the dotfiles. Uses [age](https://github.com/FiloSottile/age) encryption with automatic environment variable loading.

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

## Quick Reference

| Command | Description |
|---------|-------------|
| `secrets_add VAR FILE` | Add new env var secret interactively |
| `secrets_add_file VAR FILE DEST` | Add new file secret (kubeconfig, SSH keys, certs) |
| `secrets_rotate VAR` | Update existing secret value |
| `secrets_show VAR` | Show secret value (memory/disk, `--raw` for .age decrypt) |
| `secrets_list` | Show all secrets and their status |
| `secrets_check` | Validate mapping integrity |
| `secrets_clean` | Remove plaintext files |
| `secrets_audit [VAR]` | Show audit log |
| `secrets_refresh` | Reload all secrets |
| `secrets_sync` | Sync secrets to repo |
| `dotfiles-sync` | Full sync: secrets + git push/pull |
| `dotfiles-sync --secrets-only` | Only sync sensitive/ directory |

### GitHub Secrets

`dotf secrets sync ci` materializes a repo's CI secrets to its GitHub Actions secrets,
backend-agnostically (age **or** Bitwarden). A secret feeds a repo's CI when its
`consumers:` contains `ci:<owner>/<repo>`.

| Command | Description |
|---------|-------------|
| `dotf secrets ls` | List registry secret ids, plane, and exposed vars (no values) |
| `dotf secrets sync ci --repo OWNER/REPO --dry-run` | Preview the VAR→repo set (no values, no upload) |
| `dotf secrets sync ci --repo OWNER/REPO` | Upload the repo's `ci:<owner>/<repo>` secrets to its Actions secrets |
| `dotf secrets sync ci` | Same, targeting the current repo's origin |

## Adding a New Env Var Secret

```bash
# 1. Add the secret (auto-syncs to repo)
secrets_add MYSERVICE_TOKEN myservice.api
# Enter value when prompted

# 2. Verify it was added
secrets_list | grep MYSERVICE

# 3. Load it in current session
secrets_refresh

# 4. Verify it's loaded
echo $MYSERVICE_TOKEN

# 5. Commit and sync to remote
cd ~/Projects/dotfiles
git add sensitive/myservice.api.secret.age sensitive/env-mapping.conf
git commit -m "feat: add MYSERVICE_TOKEN secret"
dotfiles-sync
```

## Adding a File Secret

For multiline secrets (kubeconfig, SSH keys, certificates) that need to be deployed as files.

```bash
# Option A: Interactive (recommended)
secrets_add_file KUBECONFIG kubelab.kubeconfig ~/.kube/kubelab.config
# Enter source file path when prompted
# Auto-encrypts, adds mapping, syncs to repo

# Option B: Manual
# 1. Encrypt the file
age -r $(age-keygen -y ~/.config/age/key.txt) -o sensitive/kubelab.kubeconfig.secret.age ~/.kube/config

# 2. Add mapping to env-mapping.conf
echo '@KUBECONFIG=kubelab.kubeconfig>~/.kube/kubelab.config' >> sensitive/env-mapping.conf

# 3. Deploy
secrets_refresh
echo $KUBECONFIG  # → ~/.kube/kubelab.config

# 4. Commit
git add sensitive/kubelab.kubeconfig.secret.age sensitive/env-mapping.conf
git commit -m "feat: add kubeconfig file secret"
dotfiles-sync
```

### Mapping Format

`sensitive/env-mapping.conf`:

```conf
# Env var secrets: ENV_VAR_NAME=filename (without .secret.age extension)
GITHUB_PERSONAL_ACCESS_TOKEN=github.token
DOCKERHUB_TOKEN=dockerhub.token

# File secrets: @VAR_NAME=filename>dest_path
@KUBECONFIG=kubelab.kubeconfig>~/.kube/kubelab.config
```

File secret behavior:
- `@` prefix marks a file secret (stripped from env var name)
- `>` separates encrypted filename from destination path
- `~` is expanded to `$HOME` at runtime
- Env var points to deployed file path
- Files deployed with `chmod 600`
- Caching: skips re-decrypt if dest is newer than `.age` source
- `secrets_refresh` removes deployed file to force re-decrypt
- File secrets are skipped by `dotf secrets sync ci` (not GitHub Actions secrets)

## Syncing After Changes

```bash
# Sync everything (bidirectional for secrets, push/pull for git)
dotfiles-sync

# What happens:
# 1. Compares timestamps of .age files, env-mapping.conf, audit log
# 2. Copies newer files in both directions
# 3. git push from ~/Projects/dotfiles
# 4. git pull to ~/.dotfiles
```

## SSH Setup (New Machine)

SSH config and public key live in `ssh/` (plain, not encrypted). The private key is a file secret in `sensitive/id_ed25519.secret.age`.

### Linux / macOS

```bash
./setup-linux.sh        # symlinks ssh/config, copies pub key
source ~/.zshrc          # file secret deploys private key to ~/.ssh/id_ed25519
ssh rpi4                 # test
```

### Windows

```powershell
.\setup-windows.ps1     # copies ssh/config and pub key to %USERPROFILE%\.ssh\

# Private key requires manual decryption (one-time):
age -d -i "$env:USERPROFILE\.config\age\key.txt" sensitive\id_ed25519.secret.age > "$env:USERPROFILE\.ssh\id_ed25519"
```

## Physical Backup (USB + VeraCrypt)

### Initial Setup (one-time)

```bash
# 1. Install VeraCrypt
sudo add-apt-repository ppa:unit193/encryption
sudo apt update && sudo apt install veracrypt

# 2. Format USB with VeraCrypt (GUI)
# Tools → Volume Creation Wizard → Create a volume within a partition/drive
# Standard volume → Select USB (/dev/sdb1) → AES + SHA-512 → FAT → Format

# 3. Copy age key to USB (device varies — use lsblk to find it)
veracrypt /dev/sdX /media/veracrypt1
cp ~/.config/age/key.txt /media/veracrypt1/
```

### Regular Backup (monthly)

```bash
veracrypt /dev/sdX /media/veracrypt1   # use lsblk to find USB device
bash ~/Projects/dotfiles/scripts/backup-secrets-to-usb.sh /media/veracrypt1
veracrypt -d /media/veracrypt1
```

### USB Contents After Backup

```text
/media/veracrypt1/
├── key.txt              # Age private key (human/workstation)
├── ci-age-key.txt       # Age private key (CI/GitHub Actions)
├── age-standalone.sh    # Standalone decrypt script
└── secrets/             # All .secret and .secret.age files
```

### Restore on Any Linux System

```bash
sudo apt install age veracrypt
veracrypt /dev/sdX1 /media/secrets
cd /media/secrets && ./age-standalone.sh decrypt
# Secrets are now in secrets/*.dec
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DOTFILES_DIR` | `~/.dotfiles` | Local installation directory |
| `DOTFILES_REPO_DIR` | (none) | Repo directory for sync |
| `AGE_KEY_PATH` | `~/.config/age/key.txt` | Age private key location |

## Security Best Practices

1. **Never commit plaintext** — `.secret` and `.dec` files are gitignored
2. **Back up your age key** — Store in password manager or encrypted backup
3. **Use `secrets_clean`** — Remove decrypted files after use
4. **Review `secrets_audit`** — Check for unexpected changes
5. **Rotate compromised secrets** — Use `secrets_rotate` immediately
6. **Limit GitHub uploads** — Use `--select` to upload only needed secrets
7. **USB backup monthly** — Run `backup-secrets-to-usb.sh` at least once a month

## Related

- [Troubleshooting: Secrets](../troubleshooting/secrets.md) — Common secrets issues
- [ADR-002](../adr/adr-002-age-over-gpg.md) — Why age over GPG
- [ADR-005](../adr/adr-005-two-directory-sync.md) — Why two-directory sync
- Project overview — see the repo `README.md` (strategic context lives in the maintainer's knowledge store)
