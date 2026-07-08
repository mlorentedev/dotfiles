# Encrypted Secrets

> **🔐 BACKUP REMINDER:** Run `backup-secrets-to-usb.sh` monthly with your encrypted USB.

This folder contains age-encrypted secrets. Only `.age` files are committed to git.

```text
sensitive/
├── *.secret.age         # Encrypted (committed)
├── *.secret             # Plaintext (gitignored)
└── *.secret.dec         # Decrypted (gitignored)
```

`secrets/registry.yaml` (repo root) is the mapping SSOT — env var / file dest to
age source, per ADR-028.

## Quick Commands

```bash
dotf secrets ls          # List registry secret ids, plane, and exposed vars
dotf secrets show VAR    # Show a single secret's decrypted value
dotf secrets set VAR     # Add or rotate an env var secret
dotf secrets run -- CMD  # Inject secrets into a child process only (no ambient env)
```

## USB Backup

```bash
# Mount encrypted USB, backup, dismount
veracrypt /dev/sdX1 /media/veracrypt1
bash ~/Projects/dotfiles/scripts/backup-secrets-to-usb.sh /media/veracrypt1
veracrypt -d /media/veracrypt1
```

## Full Documentation

Full secrets management runbook: [`docs/runbooks/secrets-management.md`](../docs/runbooks/secrets-management.md).
