# Encrypted Secrets

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

## Full Documentation

See [docs/SECRETS.md](../docs/SECRETS.md) for complete setup and usage guide.
