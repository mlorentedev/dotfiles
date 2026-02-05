# Encrypted Secrets

> **🔐 BACKUP REMINDER:** Run `backup-secrets-to-usb.sh` monthly with your encrypted USB.

This folder contains age-encrypted secrets. Only `.age` files are committed to git.

```text
sensitive/
├── env-mapping.conf     # Maps ENV_VAR=filename
├── *.secret.age         # Encrypted (committed)
├── *.secret             # Plaintext (gitignored)
└── *.secret.dec         # Decrypted (gitignored)
```

## Quick Commands

```bash
secrets_list     # Show all secrets
secrets_add      # Add new secret
secrets_rotate   # Update secret
secrets_check    # Validate integrity
```

## USB Backup

```bash
# Mount encrypted USB, backup, dismount
veracrypt /dev/sdX1 /media/veracrypt1
bash ~/Projects/dotfiles/scripts/backup-secrets-to-usb.sh /media/veracrypt1
veracrypt -d /media/veracrypt1
```

## Full Documentation

See [docs/SECRETS.md](../docs/SECRETS.md) for complete setup and usage guide.
