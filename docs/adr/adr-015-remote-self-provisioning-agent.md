---
id: "ADR-015-remote-self-provisioning-agent"
type: adr
status: accepted
owner: manu
date: "2026-05-31"
extends: [adr-009-multi-agent-runtime, adr-012-deploy-strategy-copy-with-drift-assertion]
tags: [architecture, decision, agents, hermes, remote-agent, provisioning, vault-ssot, dotfiles, hermes-001]
created: "2026-05-31"
---

# ADR-015: Remote self-provisioning agent (Hermes)

> Closes `HERMES-001`. Builds on ADR-009 (AGENTS.md cross-agent SSOT) and diverges deliberately from the local-deploy and age-secrets models.

## Status

Accepted

## Date

2026-05-31

## Context

Every agent integrated so far (Claude Code, OpenCode, Copilot, Antigravity) runs on Manu's own machines, where the dotfiles repo is present. Their integration is a per-agent overlay that *delegates* to the root `AGENTS.md` (ADR-009), and their config is deployed by `setup-linux.sh` / `setup-windows.ps1`.

Hermes breaks three of those assumptions at once:

1. **It does not clone dotfiles.** It runs on remote NaN infrastructure (Debian 13, Telegram-fronted) and can only reach the *vault* (via a git clone + Hive MCP), never the dotfiles repo.
2. **It is low-capability and autonomous.** A junior/remote agent cannot be trusted to honor instruction-only boundaries (a bug or prompt-injection could push anywhere).
3. **It has neither the dotfiles repo nor the age key**, so the repo's age-encrypted secrets model cannot reach it.

A box probe (2026-05-31) further showed the real environment diverged from assumptions: product config at `/hermes-home/config.yaml` (not `~/.hermes/`), a native `hermes mcp add` CLI, a vault clone in ephemeral `/tmp`, and the vault token embedded in the remote URL.

## Decision Drivers

- Hermes must be operable and recoverable without the dotfiles repo on the box.
- A junior agent's boundary must hold even when it ignores instructions.
- Secrets must reach the box without the age key.
- Parity with the ADR-009 overlay model where it still applies (a per-agent file in `ai/hermes/`).

## Considered Options

1. **Remote self-provisioning via a curled `setup.sh` + vault SSOT + mechanical git-hook boundary** (CHOSEN).
2. **Treat Hermes like the local agents** (deploy via setup-linux). Rejected — the box has no dotfiles repo and the local-deploy surface must stay untouched (AC5).
3. **Instruction-only write boundary** (tell Hermes to write only in its workspace). Rejected — a junior/remote agent can ignore instructions; the boundary must be mechanical.

## Decision

Adopt a **remote self-provisioning agent** model for Hermes, with four parts:

1. **Provisioning = a curled idempotent `ai/hermes/setup.sh`**, never wired into `setup-linux.sh`. It clones the vault to a *persistent* path (`$HERMES_VAULT_PATH`, default `~/.local/state/hermes/vault` — never `/tmp`), installs Hive MCP via the native `hermes mcp add`, and sets up the sync mechanism (cron `pull --rebase` wrapper that aborts on conflict + `post-commit` auto-push).

2. **Knowledge = vault SSOT, full-clone read + write-only workspace.** Hermes reads the whole vault (the cross-project brain is its value) but writes only within `80_agents/hermes-nan/`. Its live operating law is the vault constitution `80_agents/hermes-nan/AGENTS.md`, a self-contained subset that *defers to the dotfiles root `AGENTS.md` as canonical authority* (it cannot read the root directly).

3. **Boundary = mechanical, not instructional.** Because the probe confirmed Hermes commits via the git CLI, the write-zone is enforced by local git hooks in its clone (untracked, never cloned): `pre-commit` rejects staged paths outside the workspace and token-like content; `pre-push` rejects non-fast-forward (force) pushes.

4. **Secrets diverge from the age model by design.** `GITHUB_TOKEN_KNOWLEDGE` lives in `$HERMES_HOME/.env` (chmod 600) and is supplied to git through a credential helper, so it never lands in `.git/config`. The box has neither the dotfiles repo nor the age key, so the age-secrets model does not apply.

## Consequences

- **Positive:** Hermes survives a box loss (state is in the vault); the boundary holds mechanically; the local-deploy surface is untouched; the model generalizes to future remote agents.
- **Negative / debt:** a **three-layer drift risk** — root `AGENTS.md` ↔ `ai/hermes/AGENTS.md` overlay ↔ vault constitution — mitigated only by "defer to root", not enforced. A future improvement (HERMES-002 territory) would deploy the root to the vault so Hermes reads the canonical SSOT via Hive.
- The self-authored vault docs drifted from reality and had to be curated against the shipped `setup.sh` (the AC6 / Track B work). Onboarding a remote agent's docs as *claims to verify* is now a recorded lesson.
