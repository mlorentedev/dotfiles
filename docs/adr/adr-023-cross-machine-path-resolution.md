---
id: adr-023-cross-machine-path-resolution
type: adr
status: accepted
date: 2026-06-18
related: [adr-022-dotf-init-flagship, adr-020-tooling-cli-go-convergence, adr-012-deploy-strategy-copy-with-drift-assertion, adr-018-de-vault-task-placement, adr-007-mcp-persistence-and-auto-memory]
tags: [paths, env-contract, cross-os, cross-machine, dotf, resolver, render-at-setup, knowledge-placement]
---

# ADR-023 — Cross-machine path resolution: per-machine overrides + render-at-setup

## Status

Accepted 2026-06-18. Implemented the same day in one pass: `cli/internal/env`
(resolver + generator + tests), `dotf env generate` (+ `--check`), the
`dotf doctor` drift check, the `vault.ResolveVault` rewrite, the shell/PS/hook
consumers, the setup-script wiring, and `~/.config/dotfiles/machine.json`.
`go build`/`go vet`/`go test ./...` green; `dotf env generate` + `dotf doctor`
verified end-to-end. Pending: commit + a setup run to deploy on each machine.

## Context

The same repos (`dotfiles`, the `knowledge` vault, future ones) now live on **multiple machines, different OSes, different absolute roots**. The trigger: the `knowledge` vault and the `dotfiles` repo were both relocated from `~/Projects/<x>` to `~/Projects/Workspace/<x>`. Session start immediately reported drift, and a verification pass confirmed the root cause is structural, not a one-off typo:

| Layer | Config/memory says | On-disk reality | Drift |
|---|---|---|---|
| Vault root | `~/Projects/knowledge` (hardcoded in hooks + Go fallback) | `~/Projects/Workspace/knowledge` | YES |
| `VAULT_PATH` | (should point at the vault) | **empty — never exported** | YES |
| `DOTFILES_REPO_DIR` | `~/Projects/dotfiles` | dead path; real repo at `…/Workspace/dotfiles` | YES |
| `HIVE_VAULT_PATH` | (hive daemon vault) | empty | YES |

The diagnosis: **the env-var seams exist but are never activated.** `cli/internal/vault/vault.go::ResolveVault` honors `$VAULT_PATH` then falls back to a hardcoded `~/Projects/knowledge`; the shell profiles hardcode `DOTFILES_REPO_DIR=$HOME/Projects/dotfiles` (`.bashrc:49`, `.zshrc:21`, `powershell/profile.ps1:214`); and the session hooks **ignore the seam entirely**, hardcoding the vault path (`scripts/claude-session-start.sh:121`, `scripts/claude-session-start.ps1:36`). The system only "worked" because the hardcoded fallback happened to match reality. Moving the folder broke the coincidence.

We need a **single source of truth for per-machine path values** that does not require editing committed files on each machine, resolved by **one mechanism** consumed by all three surfaces: shells (bash/zsh), PowerShell, and the `dotf` Go CLI.

## Evidence — industry audit (Regla del 3, per ADR-015)

Five references audited against their canonical docs for the "same repos, N machines, different OS/paths" problem:

| Reference | Mechanism | Per-machine source | Committed? | OS-aware | Precedence | Resolver |
|---|---|---|---|---|---|---|
| **XDG Base Dir** | env var + known default | the env var | no | no | `env-if-absolute → default` | each app |
| **chezmoi** | source-state + Go templating | `~/.config/chezmoi/chezmoi.toml` (data) | **no, machine-local** | yes (`.chezmoi.os/.arch/.hostname`) | one data file/machine → render | `chezmoi apply` (deploy-time) |
| **git config** | layered files + `includeIf` | local config + conditional includes by `gitdir:` | global yes / local no | no (path-based) | system→global→local→worktree, last-wins | git native |
| **direnv** | per-dir `.envrc` + shell hook | `.envrc` in the tree | optional | no | nearest `.envrc` walking up | shell hook |
| **12-factor** | config in env vars | the environment | no | n/a | env only | the app |

**Divergence log:**

- **Convergent (adopted):**
  1. The seam is an **env var with a documented default** (XDG, 12-factor, chezmoi data, git env overrides). The repo already does this; the bug is non-activation, not the pattern.
  2. Per-machine values live in **ONE local, uncommitted file** (chezmoi `~/.config/chezmoi/chezmoi.toml`; git local config). This is the missing layer.
  3. Resolution is a **cascade**: explicit override → local config → committed default (git's last-wins made explicit).
  4. **OS-awareness via a built-in variable** (chezmoi `.chezmoi.os`), replacing the manual `{linux, windows}` forks in `env-contract.json` with `runtime.GOOS`.
- **Divergent (rejected as not applicable):** direnv's per-CWD `.envrc` solves *per-project* env, not *per-machine root*, and adds a runtime dependency on every machine; git's `onbranch:`/`hasconfig:` are git-specific; chezmoi's full render-at-deploy engine would *replace* the existing deploy engine (ADR-012/013).

**Conclusion:** the repo already has 2 of the 4 convergent pieces (env-var seam + `env-contract.json` declarative contract). This is an *extension*, not a rewrite — adding the per-machine override layer and a single resolver.

## Constraints (evaluated against)

| # | Constraint | Origin |
|---|---|---|
| C1 | Same repos on N machines, different OS (win/linux/mac), different absolute roots | user |
| C2 | One per-machine source of truth; **zero editing of committed files per machine** | user ("algún fichero de config") |
| C3 | A **single resolver** consumed by shells (bash/zsh), PowerShell, and the `dotf` Go CLI | user |
| C4 | Cross-OS parity: all three consumers resolve identically | repo reality |
| C5 | No startup fragility: shell startup must not hard-depend on a binary that may be absent | engineering |
| C6 | Drift-detectable: `dotf doctor` must assert resolution is correct (extends ADR-012) | repo reality |
| C7 | Backward-compatible with existing seams (`VAULT_PATH`, `DOTFILES_REPO_DIR`) | repo reality |
| C8 | Durable/scalable: new machine = one local file + setup; new path = one line in the contract | user ("escalable y duradera") |

## Decision

**Render-at-setup hybrid** (Option C). A single resolver, materialized as generated env files so there is zero runtime cost and no startup chicken-and-egg.

### Inputs

1. **`env-contract.json`** (committed) — the declarative contract that already lists every structural path and its per-OS default. Stays the SSOT for *defaults*. The `default` block gains a `darwin` key (or falls back to the `linux` POSIX value when absent).
2. **`~/.config/dotfiles/machine.json`** (per-machine, **gitignored, outside the repo tree** — XDG-style, `%USERPROFILE%\.config\dotfiles\machine.json` on Windows). The chezmoi `chezmoi.toml` analog: holds only the keys this machine overrides. Absent file ⇒ pure contract defaults (a machine where the defaults are already correct needs no file).

```jsonc
// ~/.config/dotfiles/machine.json
{
  "paths": {
    "DOTFILES_REPO_DIR": "C:\\Users\\mlorente\\Projects\\Workspace\\dotfiles",
    "VAULT_PATH":        "C:\\Users\\mlorente\\Projects\\Workspace\\knowledge"
  }
}
```

### Resolution cascade (per path key)

1. **Explicit env var already set** in the process environment — highest (honors 12-factor, CI, manual override).
2. **`machine.json`** value for that key — per-machine override.
3. **`env-contract.json`** default for the current OS (`runtime.GOOS`) — committed fallback.

### Generator

`dotf env generate` resolves the cascade once and renders two deployable files into `$DOTFILES_DIR` (`~/.dotfiles`):

- `paths.sh` — `export KEY="${KEY:-<resolved>}"` per key (the `:-` preserves cascade rule #1).
- `paths.ps1` — `if (-not $env:KEY) { $env:KEY = '<resolved>' }` per key.

The generator runs as a step of `setup-{linux,windows}` and on demand (`dotf env generate`).

### Consumers

- **Shells:** `.bashrc`/`.zshrc` `source "$DOTFILES_DIR/paths.sh"`; `profile.ps1` dot-sources `paths.ps1`. The hardcoded `export DOTFILES_REPO_DIR=…` lines are **deleted** — the generated file is the only writer.
- **Hooks:** `claude-session-start.{sh,ps1}` read `$VAULT_PATH` / `$HIVE_VAULT_PATH` from the environment instead of hardcoding `~/Projects/knowledge` (fixes C3 at the hook layer).
- **Go CLI:** `dotf` reads `env-contract.json` + `machine.json` natively (it *is* the generator), so `ResolveVault` and friends share one cascade implementation.

### Drift guard

`dotf doctor` gains a check asserting the deployed `paths.sh`/`paths.ps1` match a fresh in-memory resolution of contract+override (the ADR-012 drift-assertion pattern). Edit the contract or `machine.json` without re-running `dotf env generate` ⇒ loud doctor FAIL.

### Options considered

| Option | C2 | C3 | C4 | C5 | C6 | C8 | Verdict |
|---|---|---|---|---|---|---|---|
| **A** — each consumer parses contract+override independently (jq / ConvertFrom-Json / encoding/json) | ok | **gap** (3 cascade impls) | ok | ok | ok | ok | rejected (violates C3) |
| **B** — `dotf` resolves at runtime; shells call `dotf path X` on every startup | ok | ok | ok | **gap** (binary must precede profiles; per-prompt process spawn) | ok | ok | rejected (violates C5) |
| **C** — `dotf env generate` renders `paths.{sh,ps1}` at setup | ok | ok | ok | ok | ok | ok | **selected** |

### Rejection list (do not re-debate without a changed trigger)

- **Adopt chezmoi wholesale** — replaces the deploy engine decided in ADR-012/013; the repo already owns a copy-with-drift-assertion deploy model. Reopen only if maintaining the generator proves costlier than a chezmoi migration.
- **direnv per-repo `.envrc`** — solves per-project env, not per-machine roots; adds a runtime dependency + shell hook + `allow` step on every machine. Reopen only if a genuine *repo-local* override need appears.
- **Hand-set env vars in committed profiles** — the current broken state; every machine edits committed files ⇒ merge conflicts. Does not scale to N machines (violates C2/C8).

## Build sequence (TDD-ordered, one spec / PR per step)

1. **`machine.json` schema + cascade resolver** in `cli/internal/` (extend the `env-contract` reader): `Resolve(key)` implementing env → machine.json → contract-default(`runtime.GOOS`). Table-driven `go test` covering all three precedence layers + missing-file + missing-key. Add the `darwin` default key (or POSIX fallback) to `env-contract.json`.
2. **`dotf env generate`** — render `paths.sh` + `paths.ps1` (idempotent: skip if `cmp`-equal). Golden-file tests per OS.
3. **`dotf doctor` drift check** — assert deployed `paths.*` match a fresh resolution (ADR-012 pattern).
4. **Wire consumers** — `setup-{linux,windows}` call `dotf env generate`; profiles `source paths.sh` / dot-source `paths.ps1` and **delete** the hardcoded `DOTFILES_REPO_DIR` exports; `claude-session-start.{sh,ps1}` read `$VAULT_PATH`/`$HIVE_VAULT_PATH` instead of hardcoding. Guard-grep: no hardcoded `Projects/knowledge` or `Projects/dotfiles` outside `env-contract.json` defaults + docs.
5. **Docs** — update `env-contract.json` `_comment`, the "Vault path moved" row in `docs/runbooks/guide-knowledge-distillation.md:160` (now: "edit `~/.config/dotfiles/machine.json`, run `dotf env generate`"), and the vault-root references in `AGENTS.md` to point at the seam, not a literal path.

## Consequences

### Positive
- Moving a repo on any machine becomes a **one-line edit** in `~/.config/dotfiles/machine.json` + `dotf env generate` — no committed-file edits, no merge conflicts across machines (C2, C8).
- The seams (`VAULT_PATH`, `DOTFILES_REPO_DIR`, `HIVE_VAULT_PATH`) are finally **actually set**, fixing the hook/Go inconsistency at the root (C3, C7).
- Zero runtime cost and no startup binary dependency — generated plain files (C5).
- `runtime.GOOS` resolution retires the manual `{linux, windows}` path forks as they migrate to the cascade.
- `doctor` drift assertion makes a forgotten `dotf env generate` loud, not silent (C6).

### Negative
- A generated artifact can drift if the contract/override changes without regeneration — mitigated by the C6 doctor check (same mitigation proven in ADR-012).
- One more concept in the bootstrap (`machine.json`) to document for new-machine onboarding.
- Until step 4 lands, the current machine stays on the manual-env stopgap (see follow-up).

### Neutral
- `machine.json` is JSON for parity with `env-contract.json`; TOML (chezmoi's choice) was not adopted to avoid a second parser.
- Existing explicit `VAULT_PATH` exports (if any user sets them) keep working — cascade rule #1 preserves them.

## References

- Trigger + diagnosis: session 2026-06-18 (vault + dotfiles relocated to `~/Projects/Workspace/`)
- Pattern reuse: `docs/adr/adr-012-deploy-strategy-copy-with-drift-assertion.md` (drift assertion), `docs/adr/adr-020-tooling-cli-go-convergence.md` (Go owns logic, shell owns bootstrap), `docs/adr/adr-022-dotf-init-flagship.md` (`env-contract.json` is the format `dotf` consumes)
- Code touched: `cli/internal/vault/vault.go` (`ResolveVault`), `env-contract.json`, `.bashrc`/`.zshrc`/`powershell/profile.ps1`, `scripts/claude-session-start.{sh,ps1}`, `scripts/dotfiles-{selfupdate,sync}.{sh,ps1}`
- Industry audit: chezmoi (manage-machine-to-machine-differences), XDG Base Directory Spec, git-config conditional includes, direnv, 12-factor §III
- Follow-up: `/spec init` for build-sequence step 1; a bitácora ticket tracks the epic (pending — this ADR is the "why" upstream)
