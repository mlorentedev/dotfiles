---
id: "CLI-016-dotf-env"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-18"
issue: "mlorentedev/dotfiles#445"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-016-dotf-env

> **Naming**: file lives at `<repo>/specs/CLI-016-dotf-env/proposal.md`. `CLI-016-dotf-env` is `AREA-NNN-slug`.

## Why

<!-- from issue #445: cross-machine path resolution — dotf env generate + per-machine machine.json -->

The same repos (`dotfiles`, the `knowledge` vault) live on multiple machines with different OSes and different absolute roots. The env-var seams (`VAULT_PATH`, `DOTFILES_REPO_DIR`, `HIVE_VAULT_PATH`) already existed but were **never activated**: the shell profiles hardcoded `DOTFILES_REPO_DIR`, `VAULT_PATH` was never exported, and the session hooks hardcoded `~/Projects/knowledge` instead of reading the seam. The system only "worked" because the hardcoded fallback happened to match reality — relocating the workspace to `~/Projects/Workspace/` broke vault/hive/selfupdate silently. The codebase also carried **three names for one concept** (`VAULT_PATH` in Go/skills, `VAULT_DIR` in doctor, hardcoded `~/Projects/knowledge` in hooks). ADR-023 is the decision; this spec is its implementation.

## What

A single per-machine path-resolution mechanism — the **render-at-setup hybrid** (ADR-023 Option C), consumed identically by shells, PowerShell, and the `dotf` CLI:

- **`cli/internal/env`** — a self-contained package: loads `env-contract.json` (defaults) + `~/.config/dotfiles/machine.json` (per-machine overrides), resolves the cascade `env → machine.json → contract default[GOOS]`, and renders `paths.sh` / `paths.ps1`. Deliberately does **not** import `doctor` (the dependency runs doctor→env), so there is no import cycle.
- **`dotf env generate`** (`+ --check`, `--stdout`) — renders the OS-appropriate path file into `<DOTFILES_DIR>/paths.{sh,ps1}`; idempotent; `--check` reports drift without writing.
- **`dotf doctor` drift check** — asserts the deployed path file matches a fresh resolution (the ADR-012 copy-with-drift-assertion discipline); `checkVault` realigned from the orphan `VAULT_DIR` name to the canonical `VAULT_PATH`.
- **`vault.ResolveVault`** — rewired to resolve `VAULT_PATH` through the `env` cascade, dropping the hardcoded `~/Projects/knowledge` fallback.
- **Consumers wired** — `.bashrc` / `.zshrc` / `profile.ps1` source the generated file (with a bootstrap fallback when it is not yet generated); `claude-session-start.{sh,ps1}` read `$VAULT_PATH`; `setup-{linux,windows}` run `dotf env generate` after deploying the contract.
- **Per-machine override** — `~/.config/dotfiles/machine.json` (XDG-style, outside any repo); `machine.json.example` is the committed reference.

The cascade's rule #1 (an explicit env var wins) is enforced by the generated file itself (`${VAR:-default}` / `if (-not $env:VAR)`), so nothing clobbers a deliberately-set value.

## Out of scope

- **Making `dotf doctor` fully cross-OS.** `checkContractEnvVars` / `checkContractPath` are Linux-first (they read `Default["linux"]`); on Windows `doctor.ps1` is still the path (ADR-022 defers the Windows `dotf doctor`). Running `dotf doctor` on Windows surfaces mixed-separator defaults for unset vars — pre-existing, tracked in the WIN queue, not this spec.
- **Migrating `env-contract.json` to YAML** (#227) — unrelated format migration.
- **Adopting chezmoi wholesale / direnv** — evaluated and rejected in ADR-023's rejection list.
- **Removing the per-machine User-level stopgap env vars** set during the incident — a one-time cleanup after deploy, not a code change.

## Risks / open questions

- **A generated artifact can drift** if the contract/override changes without a re-`generate`. Mitigation: the `dotf doctor` drift check (ADR-012 pattern) makes a forgotten regenerate a loud FAIL.
- **Bootstrap chicken-and-egg** (the path file locates itself via `DOTFILES_DIR`). Mitigation: `DOTFILES_DIR` is bootstrapped first (fixed `~/.dotfiles` default), then the file is sourced; an inline fallback covers a fresh machine before the first `generate`.
- **Optional vault on vault-less machines** — `VAULT_PATH`/`HIVE_VAULT_PATH` are declared without `path_exists` validation so an absent vault is not a false FAIL; the dedicated `checkVault` owns vault existence.
- **Focused contract view duplicated in `env`** (name + default only) vs doctor's richer `Contract`. Accepted to avoid a doctor refactor + import cycle; unify later if it grows.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] The cascade resolves `env → machine.json → contract default[GOOS]`; a machine.json override beats the default; vars without a default (HOME/USERPROFILE) are never emitted; darwin falls back to the linux default. → `env` unit tests.
- [x] `dotf env generate` writes the OS-appropriate `paths.{sh,ps1}`, is idempotent (no rewrite on identical content), and `--check` reports drift non-zero. → `env` tests + CLI smoke.
- [x] Generated lines preserve an already-set value (`${VAR:-…}` / `if (-not $env:VAR)`). → render tests + `dotf doctor --verbose` showing seams OK.
- [x] `dotf doctor` reports the generated path file's drift status (PASS when up to date, FAIL when stale). → `checkPathFiles`; verified end-to-end.
- [x] `vault.ResolveVault` honors `VAULT_PATH` then machine.json then the contract default, with no hardcoded `~/Projects/knowledge`. → `vault` tests green; `ResolvePath` path.
- [x] Profiles + hooks read the seam; `setup-*` run `dotf env generate`. → diff; bootstrap-fallback retained.
- [x] `go build` / `go vet` / `go test ./...` green; `gofmt -l` clean.

## References

- ADR: `docs/adr/adr-023-cross-machine-path-resolution.md` (the decision + industry audit)
- Pattern reuse: `docs/adr/adr-012-deploy-strategy-copy-with-drift-assertion.md` (drift assertion) · `docs/adr/adr-020-tooling-cli-go-convergence.md` (Go owns logic) · `docs/adr/adr-022-dotf-init-flagship.md` (`env-contract.json` is the format dotf consumes)
- Work-gate: `mlorentedev/dotfiles#445`
- Precedent: `cli/internal/spec/drift_test.go` / `cli/internal/vault/vault.go` (embed + resolver idioms)
