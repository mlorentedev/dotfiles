---
tags: [spec, tasks, secrets, cli, go]
created: "2026-06-25"
---

# Tasks - CLI-025-secrets-render

> TDD order. Likely 2 PRs: (A) add `dotf secrets render` + wire setups off the shell
> twins; (B) delete the twins + `env-mapping.conf` + the drift-guard test (closes #587).
> Could be one PR if the diff stays < ~300 LOC.

## PR-A — `dotf secrets render` + wire setups

- [x] **RED**: `cli/internal/secrets/render_test.go` — table-driven: (1) a mapped
  `{env:VAR}` is replaced with the decrypted value, (2) an unmapped `{env:HOME}` is
  left intact, (3) atomic write + `0600`, (4) no trailing-newline drift. Inject a fake
  decryptor + a fixture registry (reuse the `registry_seed_test.go` pattern).
- [x] **GREEN**: `cli/internal/secrets/render.go` — `Render(path, reg, loader, home)`:
  builds `VAR → entry` from `reg.Entries()` (env only), regex `\{env:([A-Z_][A-Z0-9_]*)\}`,
  decrypts mapped vars via `Loader.EnvFor` (skip `bw`/unmapped), atomic temp-file rewrite
  at `0600`. Lenient on undecryptable (leave intact), fail-fast on a duplicate var.
- [x] **GREEN**: `cli/internal/cmd/secrets.go` — `dotf secrets render <file>` wires
  `registryPath()` + `ageDecryptor` into `Render`. (No `--dry-run`; not needed.)
- [x] Wire `setup-linux.sh` + `setup-windows.ps1`: replace the
  `substitute_env_placeholders` / `Substitute-EnvPlaceholders` calls (opencode.jsonc,
  pi models.json) with `dotf secrets render <config>`, guarded by `command -v dotf`.
- [ ] Bump `versions.conf` is automatic (release-please). Use a `feat(secrets):` title.

## PR-B — delete the twins + env-mapping.conf (closes #587)

- [ ] Remove `substitute_env_placeholders` (`scripts/utils.sh`) +
  `Substitute-EnvPlaceholders` (`scripts/utils.ps1`) + their tests
  (`tests/sdd-009-deploy-time-secrets.bats`, `*.Tests.ps1`).
- [ ] `git rm sensitive/env-mapping.conf`; remove the registry↔env-mapping drift-guard
  test (`cli/internal/secrets/registry_seed_test.go` — or repoint it to a static fixture).
- [ ] Grep sweep: no runtime reference to `env-mapping.conf` / the twins remains.
- [ ] `gh issue close 587` (or let the closing PR's `Closes #587` do it).

## Closing

- [ ] Every AC in `proposal.md` covered by a test; `features.json` updated.
- [ ] `go test ./...`, bats, integration green; `dotf spec archive CLI-025-secrets-render`.

## Pre-flight notes (read before coding)

- `dotf secrets show` is already deployed (0.19.x) — `render` is a sibling subcommand.
- Confirm the registry rejects two secrets exposing the same env var (validation in
  `registry.go`); if not, fail-fast in `Render`.
- Match the twins' byte parity: `Set-Content -NoNewline` / `printf '%s'` (no trailing \n).
