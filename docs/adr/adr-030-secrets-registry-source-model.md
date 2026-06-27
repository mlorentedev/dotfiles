---
id: "dotfiles-adr-030-secrets-registry-source-model"
type: adr
adr: "030"
title: "Secrets registry source model — checkout-first reads, fail-loud checkout-only writes"
tags: [adr, dotfiles, secrets, registry, cli, go]
status: accepted
created: "2026-06-26"
owner: manu
relates_to: "adr-025 (cross-machine paths), adr-028 (two-tier secrets)"
---

# ADR-030: Secrets registry source model

## Context

`secrets/registry.yaml` is the mapping SSOT for `dotf secrets` (ADR-028): it declares
every secret's id, backend (`age` / `bw`), exposure, and — for migratable entries —
a dormant `bw: { item, field }` target. It is **version-controlled** and lives in the
dotfiles checkout. Setup deploys a copy to `~/.dotfiles/secrets/registry.yaml`, which
is the runtime location the shell and the CLI read from on a non-dev machine.

`dotf secrets` resolved that path through one helper:

```go
registryPath = filepath.Join(env.DotfilesDir(home), "secrets", "registry.yaml")
```

`env.DotfilesDir` returns `$DOTFILES_DIR` or `~/.dotfiles` — **always the deployed
copy, never the checkout**. Both the read path (`run`/`show`/`render`/`verify`/`sync`/
`ls`) and the single write path (`migrate`, via `FlipRegistryToBW`) used it. This was
discovered during the `dotf secrets sync ci` smoke on 2026-06-26 (#635):

- **Read drift (annoying, recoverable):** a `sync ci --dry-run` selected 0 because the
  deployed copy still carried pre-migration consumer tags while the committed checkout
  had the migrated tags. A redeploy fixed it.
- **Write durability bug (silent data loss):** `migrate` flips `backend: age→bw` in the
  **deployed** copy. The checkout SSOT stays at `age`. On the next `git pull` + setup
  redeploy, the checkout's `age` version **overwrites the deployed `bw` flip — the
  migration is silently reverted and never lands in git**. `migrate` has not run for
  real yet (everything is still `age`), so C8 (the first real migrate, #585/#612) would
  be the first to hit this.

This is the same "version-controlled SSOT written to / read from a throwaway deployed
copy" failure `env.ResolveContractPath` (ADR-025) already solved for `env-contract.json`.

## Decision

Adopt an **asymmetric-source model with a shared checkout resolver**, mirroring
`ResolveContractPath`:

1. **`env.RepoDir()`** — the shared "where is the checkout" seam: `DOTFILES_REPO_DIR`
   when it is a real directory, else walking up from cwd for a `.git` entry (a file in
   a worktree, a dir in a clone — `os.Stat` matches both). Returns `""` when neither.

2. **Reads → checkout-first, fall back to deployed** (`env.ResolveRegistryPath`). When
   a checkout is found and carries the registry, read it; else read the deployed copy
   under `DOTFILES_DIR`. On a dev machine the checkout is the fresher, authoritative
   copy, so registry edits go live on `git pull` with no redeploy footgun. On a
   non-dev machine (no checkout) it transparently falls back to the deployed copy.

3. **Writes → checkout-only, fail loud** (`env.RepoRegistryPath`). The one registry
   mutation (`migrate`'s `FlipRegistryToBW`) resolves the checkout's registry and
   **errors if no checkout is found**, rather than writing the deployed copy. A
   migration is therefore durable and committable; it can never be silently reverted
   by a redeploy.

`set` writes to **Bitwarden**, not the registry, so it is not a registry writer — the
write seam has exactly one consumer today (`migrate`), keeping the blast radius small.

### Read-side trade-off (accepted)

Checkout-first reads mean secrets resolution **follows the active git branch**: a
feature branch that edits `registry.yaml` changes what `dotf secrets run` resolves at
shell startup (since `DOTFILES_REPO_DIR` is exported in the login shell). This is the
deliberate cost of "edits go live on pull without a redeploy." It is acceptable
because (a) the registry is a mapping, not the secret values — a wrong branch resolves
to a different *declared* target, caught by `verify`, not a corrupted secret; and (b) a
stale read is recoverable, unlike the silent write loss in the deployed-copy model.

## Alternatives considered

- **Deployed-only (status quo, reads + writes):** rejected — it *is* the durability
  bug; a real `migrate` is non-reproducible.
- **Checkout-only reads (no deployed fallback):** rejected — breaks a non-dev machine
  that legitimately has only the deployed copy.
- **Asymmetric the other way (repo writes, deployed reads):** rejected — every
  registry edit would then require a redeploy before it is visible, re-introducing the
  exact footgun the smoke hit; the user chose live-on-pull reads.

## Consequences

- `migrate` is durable: the flip lands in the checkout, ready to commit; a redeploy
  cannot revert it. With no checkout it fails loud instead of writing a dead copy.
- Registry edits are live-on-pull for reads; resolution tracks the active branch
  (the accepted trade-off above).
- New shared seam `env.RepoDir()` is available for future callers that need the
  checkout root (converging with `spec.RepoRoot` / `doctor.findRepoRoot`).
- Tests now cover repo-vs-deployed resolution for both seams, including the
  fail-loud-without-checkout path that the prior path-mocking tests could not catch.

## Addendum (the values follow the mapping)

The first cut applied the checkout-first read model to the registry *path* but left the
age value store (`sensitive/`) resolving deployed-only. That re-created the split-brain
for the secret *files*: a rotation re-encrypted in the checkout was invisible to
`dotf secrets` until a redeploy, and a value freshened only in the deployed copy never
reached the committed SSOT (surfaced live during the 2026-06-26 RELEASE_TOKEN rotation).

The model therefore extends to the value store: `env.ResolveSensitiveDir()` resolves
`sensitive/` checkout-first with the same deployed fallback, and `secretLoader` uses it.
The registry and the values it maps now resolve from one source, so a repo-side rotation
is `re-encrypt → dotf secrets sync ci` with no `DOTFILES_DIR` override and no redeploy.
The age *private key* (`~/.config/age/key.txt`) is a machine secret, not in the repo, and
is intentionally unaffected.

## References

- Issue: #635. Epic: #612 (Phase C). Migration: #585.
- Code: `cli/internal/env/env.go` (`RepoDir`, `ResolveRegistryPath`, `RepoRegistryPath`,
  `ResolveSensitiveDir`), `cli/internal/cmd/secrets.go` (`registryPath`,
  `registryWritePath`, `secretLoader`), `cli/internal/cmd/secrets_migrate.go`.
- Prior art: ADR-025 (`ResolveContractPath`, the stale-deployed-copy drift fix).
