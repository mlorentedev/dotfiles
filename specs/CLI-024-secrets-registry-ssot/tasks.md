---
tags: [spec, tasks, secrets, registry, cli, go]
created: "2026-06-26"
---

# Tasks - CLI-024-secrets-registry-ssot

> TDD order. One task = one focused commit.

## Setup

- [x] Branch from main: `fix/secrets-registry-ssot` (worktree)
- [x] `proposal.md` complete; acceptance criteria testable
- [x] ADR-029 written (registry source model: checkout-first reads, fail-loud writes)

## Implementation

- [x] `env.RepoDir()` — shared checkout resolver (DOTFILES_REPO_DIR → `.git` walk-up).
- [x] `env.ResolveRegistryPath()` — read seam, checkout-first then deployed fallback.
- [x] `env.RepoRegistryPath()` — write seam, checkout-only, fail loud (no deployed write).
- [x] `secrets.go` — split seams: `registryPath` → read; `registryWritePath` → write.
- [x] `secrets_migrate.go` — flip via `registryWritePath()`, error before the mutation.
- [x] `useTempRegistry` test helper — override both seams at the fixture.

## Tests

- [x] `TestRepoDir*` — env, `.git` walk-up, none-found.
- [x] `TestResolveRegistryPath*` — repo-first, deployed fallback.
- [x] `TestRepoRegistryPath*` — checkout path, fail-loud without checkout.

## Closing

- [x] Every AC covered by ≥1 test
- [x] `features.json` carries non-vacuous verification commands
- [x] `go test ./...` green; `go vet` + `gofmt` clean; `go build ./...` ok
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec + #635
