---
id: "dotfiles-troubleshoot-secrets"
type: troubleshooting
status: active
tags: [troubleshooting, dotfiles, secrets, age, bitwarden]
created: "2026-02-22"
owner: manu
---

# Troubleshooting: Secrets

## A secret does not resolve

```bash
dotf secrets verify              # every registry secret: OK / MISSING / FAILED, no values
dotf secrets drift               # where the vault disagrees with the registry
dotf secrets probe VAR           # what bw serve answered: status, lengths, fingerprints
```

**Common causes:**
- The vault is locked (see the next section): `verify` reports FAILED for every bw-backed secret at once.
- The item or field was renamed or moved in the app. `drift` names it, and `dotf secrets reconcile` converges it back (runbook: CONVERGE).
- The id is not in `secrets/registry.yaml`. `dotf secrets ls` lists the mapped ids.
- The daemon answers from a stale cache after an edit in another client. `dotf secrets rotate` syncs for you; after any other edit, run `bw sync`.

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

## Age key not found

The age identity decrypts the floor (`SSH_KEY`) and the DR escrow. Bitwarden-backed secrets do not need it.

```bash
ls -la ~/.config/age/key.txt                 # or $AGE_KEY_PATH
age-keygen -y ~/.config/age/key.txt          # must equal AGE_KEY_PERSONAL's recipient
dotf secrets verify                          # compares the two, and names both on a mismatch
```

**Common causes:**
- The key was never restored on this machine. Restore it from the offline USB (runbook: RECOVER, step 1). Do not generate a new one: nothing in the repo is encrypted to it.
- A custom `AGE_KEY_PATH` is not exported in the shell config.
- Wrong permissions: `chmod 600 ~/.config/age/key.txt`.

## File secret not materializing

A file secret (`expose: { file: … }`) is written to its `path` with its `mode` when the secret resolves, and its `var` points at that path.

```bash
dotf secrets run --only KUBECONFIG -- sh -c 'ls -la "$KUBECONFIG"'
```

**Common causes:**
- The secret itself does not resolve: see the first section.
- The entry declares `env` rather than `file`. `dotf secrets ls` shows the exposed form.

## Agent wrapper refuses to launch (vault locked)

**Symptom.** `pi` or `opencode` exits at once with
*Error: bw resolve nan-api-key/api-key: bitwarden vault is locked: no bw serve
daemon is running — run `dotf secrets unlock`*.

**Cause.** The wrappers (`.zshrc`, `.bashrc`, `powershell/profile.ps1`) are
`dotf secrets run --only NAN_API_KEY,… -- <agent>`: the keys are resolved
through the `bw serve` daemon before the agent process exists, and a locked
vault fails the resolution (ADR-028 "not always exposed": nothing is in the
ambient shell).

**Fix.** `dotf secrets unlock` once per boot. The daemon is detached from the
terminal that started it on every OS (Linux `Setsid`; Windows `DETACHED_PROCESS`
since #1304 — before that, closing the unlocking terminal killed it and every
wrapper in every other terminal failed together, WIN-012). `dotf doctor` WARNs
with this remedy under `[Agent configs from secrets]` while the keys are bw-backed and no
unlocked daemon answers.

## Related

- [Runbook: Secrets Governance](../runbooks/guide-secrets-governance.md)
- [ADR-002: Age Over GPG](../adr/adr-002-age-over-gpg.md)
