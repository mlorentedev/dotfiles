---
id: "HARNESS-027-adr025-hardening"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-19"
issue: "mlorentedev/dotfiles#457"
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-027-adr025-hardening

> Close the cross-machine path seam end-to-end — the consumers and tooling that
> ADR-025 shipped the mechanism for, but never wired to it.

## Why

<!-- from issue #457 -->

Deploying ADR-025 on a real Windows machine (vault relocated to `~/Projects/Workspace/`)
surfaced a cluster of **pre-existing** gaps: the path-resolution mechanism worked, but
several consumers and tools still read hardcoded paths or the wrong variable, and one
real bug blocked `dotf` from self-upgrading. None are regressions from ADR-025 — they
are deuda the rigorous deploy made visible (the "linterna" effect). This hardens the
seam so the mechanism is authoritative, not merely present.

## What

- **`scripts/install-dotf.ps1` + `.sh`** — `dotf version` prints to **stderr**; the
  idempotence check read stdout (empty), and under PowerShell StrictMode `[-1]` on the
  empty array threw, aborting the install (dotf stuck at 0.6.0 vs pinned 0.7.0). Capture
  both streams (`2>&1`) + regex-extract the semver; parity fix in the `.sh`.
- **`scripts/healthcheck.ps1` + `scripts/vault-health.sh`** — read `VAULT_PATH` (the
  ADR-025 seam) instead of the never-existed `VAULT_DIR` / a hardcoded default.
- **`scripts/init-project.ps1`** — VAULT_PATH fallback uses the contract default (not
  this machine's relocated `Workspace` path); the generated context template emits the
  real `$ProjectRoot` instead of a baked-in `~/Projects/Workspace/...` layout.
- **`powershell/profile.ps1` + `.bashrc` + `.zshrc`** — auto-run `dotf env generate`
  when `paths.{sh,ps1}` is missing and dotf is on PATH (zero-touch: a fresh machine
  self-configures with no manual step).
- **`cli/internal/env/env.go`** — `ResolveContractPath` prefers the repo's
  `env-contract.json` when `DOTFILES_REPO_DIR` is a valid checkout, eliminating the
  stale-deployed-copy drift where a relocated repo generated from an out-of-date
  `~/.dotfiles` contract.
- **`cli/internal/doctor/`** — OS-gate the POSIX-only checks (zsh/direnv, shell-rc
  symlinks, tmux, `~/Applications` version dirs, Linux tool-home vars) behind a
  mockable `System.GOOS` seam so they SKIP on Windows instead of false failures.
- **`README.md`** — a "Cross-machine paths" section: where paths are changed
  (`machine.json` vs `env-contract.json`) + `dotf env generate`.

## Out of scope

- **Vault→CLI template re-vendor drift** (`TestEmbeddedTemplatesMatchVault`) — a
  separate pre-existing drift; its own ticket.
- **Skills / AGENTS.md / guardrails audit** — the recurring-session-error thread; its
  own dedicated session.
- **hive overhaul** (#450) and hive self-contained install (hive repo).

## Acceptance criteria

- [x] install-dotf idempotence reads the version robustly (stderr/stdout); StrictMode-safe;
      `.sh` parity. bats `.ps1` green.
- [x] healthcheck / vault-health / init-project resolve the vault via `VAULT_PATH`; no
      hardcoded machine layout in generated output.
- [x] Shell profiles auto-generate `paths.{sh,ps1}` on a fresh machine.
- [x] `ResolveContractPath` prefers a valid repo checkout; covered by tests.
- [x] `dotf doctor` SKIPs POSIX-only checks on Windows via a mockable `GOOS` seam;
      Linux behavior unchanged (existing tests pass); new Windows-path tests added.
- [x] README documents the path-change workflow.

## References

- ADR-025 (`docs/adr/adr-025-cross-machine-path-resolution.md`)
- Surfaced during WIN-006 (#451) deploy · work-gate: `mlorentedev/dotfiles#457`
