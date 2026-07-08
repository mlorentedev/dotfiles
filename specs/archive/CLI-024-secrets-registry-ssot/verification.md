---
tags: [spec, verification, secrets, registry, cli, go]
created: "2026-06-26"
---

# Verification - CLI-024-secrets-registry-ssot

## Evidence

- [x] **AC1** (checkout-first reads, deployed fallback) — PASS:
  `TestResolveRegistryPathPrefersRepoCheckout` (a checkout registry wins over the
  deployed copy) and `TestResolveRegistryPathFallsBackToDeployed` (no registry in the
  checkout → the deployed path under `DOTFILES_DIR`).
- [x] **AC2** (checkout-only writes, fail loud) — PASS:
  `TestRepoRegistryPathReturnsCheckoutPath` (returns `<checkout>/secrets/registry.yaml`)
  and `TestRepoRegistryPathFailsLoudWithoutCheckout` (errors with no checkout — the write
  never falls back to the deployed copy). `migrate` consumes `registryWritePath()` and
  returns that error before the flip.
- [x] **AC3** (RepoDir resolution) — PASS: `TestRepoDirPrefersDotfilesRepoDir`,
  `TestRepoDirWalksUpForGitWhenNoEnv`, `TestRepoDirNoneFound`.
- [x] **AC4** (repo-vs-deployed coverage + suite green) — PASS: the 7 env tests above
  exercise the path the prior `useTempRegistry` path-mock could not; the existing
  `cmd` + `secrets` suites stay green with the split seams.

## Test status

- `cd cli && go test ./internal/env/ ./internal/cmd/ ./internal/secrets/ -count=1` → **ok**
  (3/3 packages).
- `go vet ./...` → clean. `go build ./...` → clean. `gofmt -l internal/env internal/cmd` → empty.

## Decisions made during implementation

- **Only `migrate` writes the registry.** `set` writes Bitwarden (an item/field), not
  `registry.yaml`, so the write seam has a single consumer — smaller blast radius than
  the issue's "migrate/set" framing implied.
- **`.git` marker (not a sentinel filename) for `RepoDir`.** It detects both a normal
  clone (`.git` dir) and a worktree (`.git` file) via `os.Stat`, so the fix works in the
  isolated worktree this change was developed in.
- **Fail-loud at the env seam, not just the command.** `RepoRegistryPath` returns the
  error; `migrate` propagates it. The env-level test pins the behavior independent of the
  command wiring.
