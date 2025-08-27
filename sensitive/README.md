# Encrypted Files

This folder contains my sensitive files encrypted with [`age`](https://github.com/FiloSottile/age). Only the encrypted `.age` files get committed to git.

## How It Works

```text
sensitive/
├── *.secret     # Original files (local only)
├── *.secret.age # Encrypted versions (committed to git)
└── *.secret.dec # Decrypted files (local only)
```

## Setup

First you need to create a key pair:

```bash
mkdir -p ~/.config/age
age-keygen -o ~/.config/age/key.txt
```

**Important:** Back up that key file somewhere safe. If you lose it, your encrypted files are gone forever.

I keep mine in:

- My password manager Bitwarden
- An encrypted USB drive  

## Using the Scripts

### Encrypt your files

```bash
# Turn all *.secret files into *.secret.age files
age-encrypt-decrypt.sh encrypt
```

### Decrypt them back

```bash
# Turn all *.age files back into *.dec files
age-encrypt-decrypt.sh decrypt
```

### Work with a different folder

```bash
# If your files are somewhere else
age-encrypt-decrypt.sh encrypt /path/to/other/directory
age-encrypt-decrypt.sh decrypt /path/to/other/directory
```

## Manual Commands

### Get your public key

```bash
age-keygen -y ~/.config/age/key.txt
```

### Encrypt one file

```bash
age -r $(age-keygen -y ~/.config/age/key.txt) -o file.secret.age file.secret
```

### Decrypt one file

```bash
age -d -i ~/.config/age/key.txt -o file.secret.dec file.secret.age
```

## Git Setup

The `.gitignore` is set up to keep sensitive files out:

```gitignore
# Sensitive data
*.secret      # Original sensitive files
*.dec         # Decrypted files
```

Only the `.age` encrypted files get committed.

## Setting Up on a New Machine

1. Install `age`
2. Copy your private key to `~/.config/age/key.txt`
3. Fix the permissions: `chmod 600 ~/.config/age/key.txt`
4. Decrypt whatever you need

## Security Notes

- Don't commit `.secret` or `.dec` files
- Encrypt things before committing
- Back up your key regularly
- Use strong content in sensitive files

## If You Lose Your Key

Check these places:

- Your secure backups (USB drive, password manager)
- `~/.config/age/key.txt` on your other machines
- If you can't find it anywhere, those encrypted files are gone for good
