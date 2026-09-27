# Encrypted Secrets

Bitwarden holds the secrets; `secrets/registry.yaml` (repo root) maps each one to
the env var or file it becomes (ADR-028). This folder keeps only what must survive
without Bitwarden:

```text
sensitive/
├── id_ed25519.secret.age      # the SSH key, the one age-offline secret (committed)
└── dr/bitwarden-export.age    # the whole-vault DR escrow (committed, ADR-028 §5)
```

A `*.secret.age` that no age-backed registry entry claims is refused: a test fails
on one committed here, and `dotf doctor` fails on one left in the `~/.dotfiles`
mirror (`dotf doctor --fix` prunes it). The per-secret blobs ADR-002 left were
retired in CLI-036.

## Quick Commands

```bash
dotf secrets ls          # List registry secret ids, plane, and exposed vars
dotf secrets show VAR    # Show a single secret's decrypted value
dotf secrets set VAR     # Add or rotate an env var secret
dotf secrets run -- CMD  # Inject secrets into a child process only (no ambient env)
```

## Offline copy (USB)

The encrypted USB holds the age key and a copy of `dr/`, and carries its own
`README.md`. The procedure is being folded into `dotf secrets usb` (#1770).

## Full Documentation

Full secrets management runbook: [`docs/runbooks/secrets-management.md`](../docs/runbooks/secrets-management.md).
