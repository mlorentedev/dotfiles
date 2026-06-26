---
id: "dotfiles-troubleshoot-secrets"
type: troubleshooting
status: active
tags: [troubleshooting, dotfiles, secrets, age]
created: "2026-02-22"
owner: manu
---

# Troubleshooting: Secrets

> ⚠️ **Partly out of date — pending rewrite ([#600](https://github.com/mlorentedev/dotfiles/issues/600)).**
> `sensitive/env-mapping.conf` was retired in #587; the var→file mapping now lives in
> **`secrets/registry.yaml`** (ADR-028). Commands below that grep `env-mapping.conf` should
> target the registry (`dotf secrets ls` lists the mapped ids). The age-decrypt steps still apply.

## Secret not loading

```bash
# Check the secret exists and is mapped
secrets_list | grep VAR_NAME
secrets_check

# Check key is accessible
ls -la ~/.config/age/key.txt
age-keygen -y ~/.config/age/key.txt  # Should show public key
```

**Common causes:**
- Missing entry in `sensitive/env-mapping.conf`
- `.secret.age` file encrypted with a different key
- Shell not sourcing `load-secrets.sh` (check `.zshrc` / `.bashrc`)

## Sync not working

```bash
# Verify both directories exist
ls -la ~/.dotfiles/sensitive/
ls -la ~/Projects/dotfiles/sensitive/

# Check DOTFILES_REPO_DIR is set
echo $DOTFILES_REPO_DIR
```

**Common causes:**
- `DOTFILES_REPO_DIR` not exported in shell config
- One of the two directories doesn't exist yet (clone or run setup)
- File permissions preventing copy

## GitHub upload failing

```bash
# Check authentication
gh auth status

# Check you're in a git repo
gh repo view

# Preview the VAR->repo set first (no values, no upload)
dotf secrets sync ci --repo OWNER/REPO --dry-run
```

**Common causes:**
- Not authenticated with `gh` (`gh auth login`)
- Not inside a git repository
- Repository doesn't have GitHub Actions enabled

## Key not found

```bash
# Check default location
ls -la ~/.config/age/key.txt

# Or set custom location
export AGE_KEY_PATH=/path/to/key.txt
```

**Common causes:**
- Key file doesn't exist (run `age-keygen -o ~/.config/age/key.txt`)
- Wrong permissions (`chmod 600 ~/.config/age/key.txt`)
- Custom `AGE_KEY_PATH` not set in shell config

## File secret not deploying

```bash
# Check mapping format (must have @ prefix and > separator)
grep "^@" sensitive/env-mapping.conf

# Force re-deploy
secrets_refresh
echo $KUBECONFIG  # Should show dest path
```

**Common causes:**
- Missing `@` prefix in env-mapping.conf
- Missing `>` separator between filename and dest path
- Destination directory doesn't exist (e.g., `~/.kube/` not created)
- Dest file is newer than `.age` source (caching) — use `secrets_refresh`

## Related

- [Runbook: Secrets Management](../runbooks/secrets-management.md)
- [ADR-002: Age Over GPG](../adr/adr-002-age-over-gpg.md)
