---
id: "CLI-058-env-persist"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-28"
issue: "mlorentedev/dotfiles#1324"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, env-contract, adr-025, windows, copilot]
template_version: "1.0"
---

# CLI-058-env-persist

## Why

The ADR-025 cascade (`env-contract.json` defaults + `machine.json` overrides)
reaches a process only through the rc files: `paths.ps1` is dot-sourced by
`profile.ps1`, `paths.sh` by `.bashrc`/`.zshrc`. A process started without a
profile inherits none of it — and Copilot runs every tool call as `pwsh
-NoProfile -NoLogo` by default. Measured on the Windows work box on 2026-08-27:
`DOTFILES_REPO_DIR`, `DOTFILES_DIR`, `VAULT_PATH` and `COPILOT_HOME` were empty at
User scope while every shell had them; inside a Copilot tool call `dotf`
resolved (User PATH) but `dotf harness mirror` could not locate the checkout and
the instructions' fallback `$DOTFILES_REPO_DIR/AGENTS.md` was unresolvable.
Issue #1324.

## What

`dotf env persist` resolves every structural variable exactly as `dotf env
generate` does and writes each into the OS's per-user persistent scope —
`HKCU\Environment` on Windows, the scope `[Environment]::SetEnvironmentVariable(...,
'User')` writes and every new process reads — touching only values that differ,
and broadcasting `WM_SETTINGCHANGE` so a terminal opened afterwards sees them.
`--check` reports drift without writing. Where the OS has no such scope (Linux,
macOS) the command is a no-op that says so: the rc files source `paths.sh` and
unit files carry their own environment. `setup-windows.ps1` calls it right after
`dotf env generate`. `dotf doctor` gains a "Persisted environment (user scope)"
section that WARNs, naming the variables, when the persisted values are missing
or stale.

## Out of scope

- Setting `powershellFlags: []` in Copilot's config (would load the interactive
  profile on every tool call — the ticket names it as the wrong fix).
- Machine scope (`HKLM`): per-user is what the contract describes and needs no
  elevation.
- Linux `environment.d`: no profile-less consumer on the Linux boxes reads it
  today; the command is an explicit no-op there so setup scripts stay symmetric.

## Risks / open questions

- A value the user set by hand at User scope is overwritten by the contract's
  resolution — the same rule the cascade already applies (`machine.json` is the
  override surface, not the registry).
- `AGE_KEY_PATH` / `SOPS_AGE_KEY_FILE` are contract variables and are persisted
  too: they are paths to the key file, never the key.

## Acceptance criteria

- [x] AC1 — `dotf env persist` writes every resolved contract variable at user
  scope and a second run changes nothing (only values that differ are written).
- [x] AC2 — `--check` names missing/stale variables and exits non-zero; exits zero
  when all are persisted.
- [x] AC3 — on an OS without a per-user persistent scope the command is a no-op
  that says so and exits zero.
- [x] AC4 — `dotf doctor` reports the persisted scope: PASS when all variables
  match, WARN naming the missing/stale ones with the remedy, WARN when the store
  is unreadable, no section where the OS has no such scope.
- [x] AC5 — `setup-windows.ps1` calls `dotf env persist` after `dotf env generate`.
- [x] AC6 — on the Windows work box: after `dotf env persist`, a process started
  with a fresh environment built from the registry (`Start-Process
  -UseNewEnvironment`, no profile, no inheritance) sees the variables.

## References

- Bitácora board: #1324
- ADR-025 (paths resolve at setup), CLI-039 (`dotf env generate`), #1323 (Copilot instructions outside repos)

<!-- archived 2026-08-28 — PR: https://github.com/mlorentedev/dotfiles/pull/1362 -->
