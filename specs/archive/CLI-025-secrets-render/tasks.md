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
  pi models.json) with `dotf secrets render <config>`, gated on the subcommand
  *succeeding* (Linux: `command -v dotf && dotf secrets render` in the `if` condition,
  which exempts it from `set -e`; Windows: `Get-Command dotf` + `$LASTEXITCODE` check),
  with the twin as fallback so a stale dotf never aborts setup.
- [ ] Bump `versions.conf` is automatic (release-please). Use a `feat(secrets):` title.

## PR-B — delete the twins + env-mapping.conf (closes #587)

- [x] Remove `substitute_env_placeholders` (`scripts/utils.sh`) +
  `Substitute-EnvPlaceholders` (`scripts/utils.ps1`) + their tests
  (`tests/sdd-009-deploy-time-secrets.bats`, `*.Tests.ps1`); simplify the setup
  fallback to render-or-literal (no twin).
- [x] `git rm sensitive/env-mapping.conf`; remove `ParseMapping` + the
  registry↔env-mapping drift-guard test (`registry_seed_test.go`) + `TestParseMapping`;
  the round-trip test repointed to a static expected.
- [x] **Out-of-plan: env-mapping.conf had 3 more live consumers.** Migrated the doctor
  (`checks_pat.githubPATSecrets`, `checks_deploy.checkSecrets`) and
  `github-secrets-manager.sh` (`--list`/`--from-mapping`) to the registry; added
  `dotf secrets ls --pairs` (env `VAR<TAB>age-source`) as the enabler; dropped
  env-mapping.conf from `dotfiles-sync.{sh,ps1}` and the setup-windows deploy step.
- [x] Grep sweep: no runtime reference to `env-mapping.conf` / the twins remains
  (only descriptive comments + historical spec records).
- [x] `Closes #587` on the PR. Runbook/troubleshooting full rewrite tracked in #600.

## Closing

- [x] Every AC in `proposal.md` covered by a test; PR-A + PR-B green.
- [x] `go test ./...` green (11 ok), shellcheck + PSScriptAnalyzer parse clean.
- [ ] On merge: `dotf spec archive CLI-025-secrets-render`.

## Pre-flight notes (read before coding)

- `dotf secrets show` is already deployed (0.19.x) — `render` is a sibling subcommand.
- Confirm the registry rejects two secrets exposing the same env var (validation in
  `registry.go`); if not, fail-fast in `Render`.
- Match the twins' byte parity: `Set-Content -NoNewline` / `printf '%s'` (no trailing \n).
