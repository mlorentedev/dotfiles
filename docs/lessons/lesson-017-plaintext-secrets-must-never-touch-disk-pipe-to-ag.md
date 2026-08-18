---
id: lesson-017-plaintext-secrets-must-never-touch-disk-pipe-to-ag
type: lesson
status: active
created: "2026-03-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 017: Plaintext secrets must never touch disk — pipe to age directly

**Context**: `secrets_add` and `secrets_rotate` in `load-secrets.sh` wrote the secret value to a plaintext file, encrypted it with age, then deleted the plaintext.

**Problem**: Between write and delete, the secret exists unencrypted on disk. On SSDs with wear leveling or systems with filesystem journaling, `rm` doesn't guarantee data erasure. An interrupted script leaves the plaintext file permanently.

**Solution**: Piped the value directly to age via stdin: `printf '%s' "$value" | age_encrypt "$encrypted_file" "$key_path"`. No temporary plaintext file is ever created.

**Rule**: Never write secrets to disk before encryption. Always pipe directly to the encryption tool's stdin. This eliminates both the crash-window vulnerability and the data-remanence risk.
