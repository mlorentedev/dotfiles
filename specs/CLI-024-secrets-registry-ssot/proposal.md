---
id: "CLI-024-secrets-registry-ssot"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-26"
issue: "mlorentedev/dotfiles#635"
tags: [spec, proposal, secrets, registry, cli, go]
template_version: "1.0"
---

# CLI-024-secrets-registry-ssot

## Why

`dotf secrets` resolved `secrets/registry.yaml` through `env.DotfilesDir` — always the
**deployed** copy under `~/.dotfiles`, never the version-controlled checkout. Both the
read path and the one write path (`migrate` → `FlipRegistryToBW`) used it. The smoke of
`dotf secrets sync ci` (#635) surfaced two failures:

- **Write durability bug:** `migrate` flips `backend: age→bw` in the deployed copy; the
  checkout SSOT stays `age`; the next redeploy silently reverts the flip and it never
  reaches git. C8 (the first real migrate) would hit this.
- **Read drift:** resolution read a stale deployed copy that diverged from the committed
  checkout (a `--dry-run` selected 0).

This is the same failure `env.ResolveContractPath` (ADR-025) already solved for
`env-contract.json`.

## What

Per **ADR-030**, an asymmetric source model over a shared checkout resolver:

- `env.RepoDir()` — `DOTFILES_REPO_DIR` (when a real dir) else walk up cwd for `.git`
  (file or dir); `""` when neither.
- `env.ResolveRegistryPath()` — **reads** checkout-first, fall back to the deployed copy.
- `env.RepoRegistryPath()` — **writes** resolve the checkout's registry and **fail loud**
  if no checkout is found (never write the deployed copy).
- `secrets.go`: read seam `registryPath = env.ResolveRegistryPath`; new write seam
  `registryWritePath = env.RepoRegistryPath`.
- `secrets_migrate.go`: flip via `registryWritePath()`, erroring before the mutation when
  no checkout exists.

`set` writes Bitwarden, not the registry, so it is not a registry writer — one write
consumer (`migrate`).

## Acceptance criteria

- **AC1** — Reads prefer the checkout registry, fall back to the deployed copy when no
  checkout / no file there.
- **AC2** — `migrate` writes the checkout SSOT and fails loud when no checkout is found.
- **AC3** — `RepoDir` resolves via `DOTFILES_REPO_DIR` and via `.git` walk-up, and is
  empty when neither.
- **AC4** — Tests cover repo-vs-deployed resolution for both seams (the prior path-mock
  tests could not catch this); suite green, vet + fmt clean, module builds.

## Out of scope

- The `sync ci` "0 selected" clearer message + lessons (separate follow-up per #635).
- C8 (the first real age→bw migrate) — needs Windows/bw; validated after this lands.
