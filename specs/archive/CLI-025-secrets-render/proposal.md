---
id: "CLI-025-secrets-render"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-25"
issue: "mlorentedev/dotfiles#587"
tags: [spec, proposal, secrets, cli, go]
template_version: "1.0"
---

# CLI-025-secrets-render

## Why

Deploy-time config materialization (SDD-009) — substituting `{env:VAR}` placeholders
in `opencode.jsonc` / pi `models.json` with the decrypted secret at setup time — is
done today by a **twin pair**: `substitute_env_placeholders` (`scripts/utils.sh`) and
`Substitute-EnvPlaceholders` (`scripts/utils.ps1`). Both age-decrypt `{env:VAR}` by
reading **`sensitive/env-mapping.conf`** directly. They are:

1. **The last live consumers of `env-mapping.conf`** — it cannot be deleted (closing
   #587) until they stop reading it.
2. **An ADR-020 twin pair** — the same logic in two OS dialects, which already drifted
   once (their default `SecretsDir` resolution differs).

A `dotf secrets render` subcommand collapses both into one Go implementation over the
`secrets/registry.yaml` SSOT, deletes the twins + `env-mapping.conf`, and closes #587.

## What

`dotf secrets render <file>` rewrites `<file>` in place, substituting every
`{env:VAR}` placeholder whose `VAR` is a registry-exposed env secret with that
secret's decrypted value; placeholders with no registry mapping are **left intact**
(opencode/pi resolve those at runtime). Atomic write, `0600`.

`setup-{linux,windows}` call `dotf secrets render <config>` for `opencode.jsonc` and
pi `models.json` instead of the shell twins. The `substitute_env_placeholders` /
`Substitute-EnvPlaceholders` functions + their tests are deleted, `env-mapping.conf`
is removed, and the registry↔env-mapping drift-guard test is retired.

## Out of scope

- The Bitwarden (`bw`) backend — Phase 3 / #585. `render` reads the age backend only,
  via `Secret.AgeBacked()` (same gate as `show`/`run`).
- Unmapped placeholders (no registry entry — e.g. `{env:HOME}`, `{env:OLLAMA_API_KEY}`)
  — left intact, even if the name looks secret: the decision is registry-based, not value-based.
- The agy `mcp_config.json` materialization (a jq-merge, not a `{env:VAR}`
  substitution; already on `dotf secrets show` since B3) — untouched.

## Risks / open questions

- **VAR → value resolution.** The registry maps `id → expose.env` (a var, a list, or a
  per-var map). Reuse `secrets.Registry.Entries()` (already flattens to `{Var, age}`
  entries for `run`) to build a `VAR → ageSource` map; decrypt via the existing
  `ageDecryptor` seam. If two secrets expose the same `VAR`, the registry validation
  should already reject it — confirm, else fail-fast in `render`.
- **Atomic + mode parity.** Match the twins: stage to a temp file, `0600`, rename over
  the target; preserve byte parity (no trailing newline added).
- **Unmapped placeholders must NOT error** — leave `{env:VAR}` intact and continue, so
  `{env:HOME}` etc. survive for the runtime resolver (matches the twins).
- **Cross-OS verification gap.** setup-windows isn't run at runtime in CI — unit-test
  the render core in Go (table-driven) so correctness doesn't depend on a setup run.

## Acceptance criteria

- [ ] **AC1** — `dotf secrets render <file>` substitutes `{env:VAR}` for each registry
  env secret whose exposed var is `VAR` (decrypted value), and leaves unmapped
  placeholders intact. *Verify:* table-driven Go test on the render core + a fixture.
- [ ] **AC2** — atomic write, `0600`, no trailing-newline drift. *Verify:* Go test.
- [ ] **AC3** — `setup-{linux,windows}` materialize `opencode.jsonc` / pi `models.json`
  via `dotf secrets render`; the `substitute_env_placeholders` twins + their tests are
  gone. *Verify:* grep clean + bats.
- [ ] **AC4** — `sensitive/env-mapping.conf` deleted; the registry↔env-mapping
  drift-guard test removed; no runtime reference remains. *Verify:* grep + `go test`.
- [ ] **AC5** — bats + integration green cross-OS; **#587 closes**.

## References

- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`; ADR-020 (twin convergence).
- Issue: `mlorentedev/dotfiles#587` (this is its final, env-mapping-deleting step).
- Reuse: `cli/internal/secrets/registry.go` (`Entries()`), `cli/internal/cmd/secrets.go`
  (`ageDecryptor` seam, `registryPath()`), `scripts/utils.{sh,ps1}` (twin behaviour to match).
- Prior: spec `CLI-024-secrets-retire-loadsecrets` (B1/B2/B3/PR-C, all merged).
