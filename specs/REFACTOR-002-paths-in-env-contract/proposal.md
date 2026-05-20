---
id: "REFACTOR-002-paths-in-env-contract"
type: spec
status: draft
created: "2026-05-19"
tags: [spec, proposal, env-contract, paths]
template_version: "1.0"
---

# REFACTOR-002-paths-in-env-contract

## Why

`env-contract.json` is the declarative SSOT for structural environment expectations, read by `doctor.{sh,ps1}` to verify (and optionally fix) drift. Today it covers only two paths: `DOTFILES_DIR` and `CLAUDE_CONFIG_DIR`. The other per-agent install locations — Gemini, Copilot (v2), OpenCode, plus the `SCRIPTS_DIR` deploy target — are hardcoded across scripts (`$HOME/.gemini`, `$HOME/.copilot`, `$HOME/.config/opencode`, `$HOME/.dotfiles/scripts`) with no single declaration. Drift between hardcoded paths is invisible until something breaks.

AUDIT-002 surfaced this as the next concrete extension of the SSOT pattern (`env-contract.json` + `doctor.{sh,ps1}` is already the gold standard for cross-OS factorisation). Adding 4 entries gets these paths under contract — doctor validates them as side-effect on every `doctor` run, and future scripts can read from the contract instead of hardcoding.

## What

After this PR merges:

1. `env-contract.json` gains 4 new entries in `env_vars[]`:
   - `SCRIPTS_DIR` — the deployed scripts dir (`$HOME/.dotfiles/scripts` on Linux, `$env:USERPROFILE\.dotfiles\scripts` on Windows)
   - `GEMINI_HOME` — `$HOME/.gemini` / `$env:USERPROFILE\.gemini`
   - `COPILOT_HOME` — `$HOME/.copilot` / `$env:USERPROFILE\.copilot`
   - `OPENCODE_HOME` — `$HOME/.config/opencode` / `$env:USERPROFILE\.config\opencode`
2. `CLAUDE_CONFIG_DIR` (existing entry) stays canonical for Claude — it's the upstream Claude Code CLI contract; adding `CLAUDE_HOME` as alias would create two-name-one-concern duplication. Naming convention documented in env-contract `_comment` and AUDIT-002.
3. RC files gain the corresponding exports:
   - `.zshrc` + `.bashrc`: `export SCRIPTS_DIR`, `GEMINI_HOME`, `COPILOT_HOME`, `OPENCODE_HOME`
   - `powershell/profile.ps1`: `$env:SCRIPTS_DIR`, `$env:GEMINI_HOME`, `$env:COPILOT_HOME`, `$env:OPENCODE_HOME`
4. New bats coverage asserts the exports exist in each RC file and that `doctor.sh --check` reports all 4 new vars as validated (matching their defaults).

## Out of scope

- Migrating existing scripts to consume the new vars (scope creep — scripts continue to hardcode in this PR; migration is a separate refactor wave).
- Adding non-path env vars to the contract (e.g. tooling-required vars, version pins — those live in `versions.conf` and `sensitive/env-mapping.conf`).
- Adding a `CLAUDE_HOME` alias to match the naming pattern (rejected; one-name-one-concern wins over cosmetic uniformity).
- Adding `setup-windows.ps1` symmetric exports (the PowerShell profile is the deploy target; the setup script doesn't itself need the vars). Verified: `powershell/profile.ps1` already exports `$env:DOTFILES_DIR` — extending follows the same pattern.

## Risks / open questions

- **Risk: drift between env-contract.json defaults and RC-file values.** If somebody updates the contract default without touching RC exports, doctor reports OK but the actual env has a stale value. **Mitigation**: bats parity test asserts the RC-file value matches the contract default literally.
- **Risk: the new vars are unused (no consumers).** This PR only ships the *declaration*. **Mitigation**: documented in "Out of scope" — consumption follows in a future refactor wave. The value of this PR is putting the paths under contract so doctor can validate them and future scripts can read them.
- **Risk: `OPENCODE_HOME` path is XDG-style (`$HOME/.config/opencode`), not the `$HOME/.X` pattern.** Inconsistent with the other 3. **Decision**: keep XDG-style because that's where OpenCode actually deploys (per `setup-linux.sh` and `opencode.jsonc` schema). Inconsistency is honest; the contract is descriptive, not prescriptive.

## Acceptance criteria

- [ ] `env-contract.json` contains 4 new `env_vars[]` entries with `name`, `required: false`, `description`, `default` (linux + windows), `validation: "path_exists"`.
- [ ] `.zshrc` exports `SCRIPTS_DIR`, `GEMINI_HOME`, `COPILOT_HOME`, `OPENCODE_HOME` (4 lines).
- [ ] `.bashrc` exports the same 4 vars.
- [ ] `powershell/profile.ps1` exports `$env:SCRIPTS_DIR`, `$env:GEMINI_HOME`, `$env:COPILOT_HOME`, `$env:OPENCODE_HOME` (PowerShell syntax).
- [ ] `env-contract.json` is valid JSON (`jq -e . env-contract.json` passes).
- [ ] `doctor.sh --check --verbose` reports the 4 new vars on a deployed system (validated via bats).
- [ ] New bats cases assert presence of each export in each RC file (`tests/env-contract.bats` or extend an existing file).
- [ ] Full bats suite green (target: 659 + new cases).
- [ ] `shellcheck --severity=error` clean on changed RC files.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` REFACTOR-002 entry.
- Vault: [[audit-002-cross-os-duplication]] — surfaced this as the natural extension of `doctor` + `env-contract` SSOT pattern.
- Vault: [[dotfiles-architecture-map]] (AUDIT-004) — locates the SSOTs at repo root.
- Sibling pattern: `mcp-servers.json` + setup scripts (also adds entries declaratively; setup scripts consume).
