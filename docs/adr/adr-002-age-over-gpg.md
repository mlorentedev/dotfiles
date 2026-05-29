---
id: "dotfiles-adr-002-age-over-gpg"
type: adr
adr: "002"
title: Age Over GPG for Secrets Encryption
tags: [adr, dotfiles, secrets, age]
status: accepted
created: "2026-02-22"
owner: manu
---

# ADR-002: Age Over GPG for Secrets Encryption

## Context

The dotfiles needed a way to encrypt secrets (API tokens, SSH keys, kubeconfig files) and track them in git. The encrypted files must be decryptable at shell startup without user interaction.

Two encryption tools were evaluated:

| Criteria | GPG | age |
|----------|-----|-----|
| Key management | Keyring with trust model, expiry, subkeys | Single file (`key.txt`) |
| Setup complexity | `gpg --gen-key`, configure trust, export/import | `age-keygen -o key.txt` (one command) |
| Decryption UX | May prompt for passphrase (agent caching) | Silent with `-i key.txt` flag |
| Cross-platform | Works but keyring portability is painful | Single binary, key is a plain file |
| Dependencies | `gnupg` (often pre-installed but version conflicts) | Single static binary |
| Audit surface | Large (keyservers, trust model, subkeys) | Minimal (X25519 only) |

## Decision

Use [age](https://github.com/FiloSottile/age) for all secrets encryption.

Key stored at `~/.config/age/key.txt`. Encrypted files stored as `sensitive/*.secret.age` in the git repo. Decryption happens automatically at shell login via `scripts/load-secrets.sh`.

## Consequences

### Positive

- **One-command setup:** `age-keygen -o ~/.config/age/key.txt` — no keyring, no trust model, no expiry
- **Silent decryption:** `age -d -i key.txt` never prompts, enabling automated shell startup loading
- **Portable key:** Single file copies trivially to new machines or USB backup
- **Simple scripting:** All `load-secrets.sh` commands use `age -r` (encrypt) and `age -d -i` (decrypt)
- **No version conflicts:** Static binary, no dependency on system GPG version

### Negative

- **No key expiry:** age keys don't expire — rotation is manual discipline
- **No keyserver ecosystem:** Can't use web-of-trust or publish keys
- **Single recipient:** Current setup uses one key for all secrets (no per-secret access control)

### Mitigations

- Monthly USB backup of `key.txt` to VeraCrypt-encrypted drive
- `secrets_audit` command tracks all secret operations for review
- If multi-recipient is needed, age supports multiple `-r` flags
