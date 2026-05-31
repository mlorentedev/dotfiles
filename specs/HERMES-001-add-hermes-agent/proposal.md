---
id: "HERMES-001-add-hermes-agent"
status: draft # draft | implementing | verifying | archived
created: "2026-05-31"
tags: [spec, proposal]
template_version: "1.0"
---

# HERMES-001: Add Hermes agent

> **Naming**: file lives at `<repo>/specs/HERMES-001-add-hermes-agent/proposal.md`.

## Why

<!-- from 11-tasks.md: *(P1, queued 2026-05-31, GH [#193](https://github.com/mlorentedev/dotfiles/issues/193), SDD-required — spec to be defined together next session)*: Add **Hermes Agent** (remote ops agent on NaN infra, via Telegram — <https://hermes-agent.nousresearch.com/docs>) as a new agent in the dotfiles ecosystem. **Proposed deliverables:** (a) `ai/hermes/AGENTS.md` — thin pointer to root `AGENTS.md` (per ADR-009); explains Hermes is a REMOTE infra agent (Debian 13, full system access, built-in skills github/email/mcp/cron), uses Hive MCP `uvx hive-vault`, writes to `vault/80_agents/Hermes-NaN/`, speaks Spanish-from-Spain. (b) `ai/hermes/setup.sh` — minimal, idempotent, non-interactive, NO sudo/apt, does NOT touch `setup-linux.sh`/`mcp-servers.json`: verify `uv` + `$GITHUB_TOKEN_KNOWLEDGE` (exit 1 if missing), `uv tool install --upgrade hive-vault`, clone/pull vault to `/tmp/hermes-vault`, ensure `80_agents/Hermes-NaN/_index.md`, generate `~/.hermes/config.yaml` (MCP hive, `HIVE_VAULT_PATH` env, no hardcoded path) + `~/.hermes/.env` (token). **Fit analysis (2026-05-31):** ✓ strong fit — (1) cross-agent SSOT pointer pattern (ADR-009); (2) **FIRST concrete consumer of the `80_agents/` autonomous-agent commit-policy shipped in PR #189** (autonomous agents commit only in `80_agents/`) — validates that policy with a real case; (3) reuses `hive-vault` like claude/agy. **NEW pattern:** Hermes is REMOTE + SELF-provisioning (runs its own setup.sh on remote infra), deliberately NOT wired into `setup-linux` — a coherent extension, not a local-deploy agent. **Open design Qs for the spec:** (1) **vault trust boundary** — cloning the FULL private vault to remote `/tmp` exposes all AI-memory/lessons to NaN infra; decide full-clone vs scoped/`80_agents`-only access; (2) **secrets divergence** — token-in-`.env` vs the age-secrets model (justified for remote; document it); (3) idempotence test approach (bats? remote-only?) + confirm `ai/hermes/` location. **Process:** define spec together next session → `specs/HERMES-001-add-hermes-agent/` → implement. Part of the multi-agent runtime (ADR-009 / HARNESS-001 [#162](https://github.com/mlorentedev/dotfiles/issues/162) family). -->

Hermes is a remote ops agent (Debian 13 on NaN infrastructure, Telegram gateway, 24/7) that already operates and self-documents its state in the vault under `80_agents/hermes-nan/`. Today its provisioning exists only as prose the agent wrote about itself, and that prose has already drifted from reality (a `validate.sh` that checks filenames the folder no longer uses, a constitution file referenced everywhere but absent, a vault clone living in ephemeral `/tmp` whose loss takes the auto-push hook with it). A fresh box therefore cannot be brought up reproducibly. This spec makes the agent reproducible and self-replicating: a versioned, idempotent `ai/hermes/setup.sh` plus a thin `ai/hermes/AGENTS.md` pointer in dotfiles, backed by a curated, internally consistent SSOT in `80_agents/hermes-nan/`. It is also the first concrete consumer of the `80_agents/` autonomous-agent commit policy (PR #189) and extends the cross-agent SSOT model (ADR-009).

## What

After this change, observable behavior that did not exist before:

1. **A new `ai/hermes/` overlay exists in dotfiles**, mirroring the per-agent convention (`ai/agy/`, `ai/nan/`, `ai/claude/`):
   - `ai/hermes/AGENTS.md` — a thin pointer (ADR-009 parity) to the root `AGENTS.md` for shared rules **and** to the vault constitution (`80_agents/hermes-nan/AGENTS.md`) for the agent-specific operating law. It is documentation in the dotfiles repo; it is NOT Hermes's runtime brain (Hermes reads the vault, not the dotfiles checkout).
   - `ai/hermes/setup.sh` — the remote provisioner, the entry point a fresh box curls.
   - the dotfiles-side mirror of the cross-agent `pattern-loader` skill (today untracked in the working tree) is brought under version control as part of Hermes's contract.

2. **`setup.sh`, run on a fresh Debian 13 box, provisions the agent end-to-end** and is safe to re-run (idempotent, non-interactive). In one invocation it:
   - verifies prerequisites: `uv` present; `GITHUB_TOKEN_KNOWLEDGE` available from the environment or `~/.hermes/.env` (fail fast, non-zero exit, clear message if either is missing);
   - installs/updates the Hive MCP launcher: `uv tool install --upgrade hive-vault`;
   - clones (first run) or `git pull --ff-only` (subsequent runs) the private vault to a **persistent** path `$HERMES_VAULT_PATH` (default `~/.local/state/hermes/vault`), using a git credential helper that reads the token from `~/.hermes/.env` (token never persisted in `.git/config`);
   - installs the vault-sync mechanism: a userspace `crontab` pull entry and a `post-commit` auto-push hook (installing the `cron` package via `apt-get` only if absent — the box runs as root);
   - ensures `~/.hermes/.env` (the token), strips any token embedded in the vault remote URL, and registers the Hive MCP server via the native `hermes mcp add` CLI (idempotent; the product config at `/hermes-home/config.yaml` is left to the product, never hand-edited);
   - installs mechanical guardrails into Hermes's clone — a `pre-commit` hook (write-zone + secret-scan) and a `pre-push` hook (no force-push);
   - ensures `~/.hermes/.env` (chmod 600) and runs the vault's `validate.sh` as a final smoke check.

3. **The agent's vault SSOT (`80_agents/hermes-nan/`) is internally consistent and replication-ready**: `validate.sh` matches the real numbered filenames, the canonical workspace name is `hermes-nan` throughout, the vault constitution (`AGENTS.md`) exists, and the bootstrap/recovery docs (`00-context.md`, `20-servers.md`, `13-config.md`, `14-env-variables.md`) describe the reconciled `setup.sh` (persistent path, apt-if-absent, marker-patched config). Every curation change updates the structure map the agent reads (`00-context.md`) and leaves a `sessions/` record so the junior agent re-discovers the new state.

4. **`setup.sh` touches none of the local-deploy surface**: not `setup-linux.sh`, not `setup-windows.ps1`, not `mcp-servers.json`. Hermes is remote and self-provisioning by design; it is deliberately not wired into the local bootstrap.

5. **A mechanical safety net** wraps provisioning and the agent's vault writes — Hermes is a low-capability (junior) agent, so instruction-only boundaries are insufficient. It comprises: a preflight token/remote auth-check that aborts before any heavy mutation; an idempotent-by-construction script verified by a full two-run test; a robust auto-pull wrapper that aborts a conflicted rebase instead of wedging the clone; never force-pushing; and — once the commit path is confirmed (git CLI vs Hive MCP) — local `pre-commit`/`pre-push` hooks in Hermes's clone that reject writes outside `80_agents/hermes-nan/`, block token-like content, and forbid force-push.

### Design decisions (resolved with the user, 2026-05-31)

Several original ticket assumptions were revised during design; the divergence is intentional and recorded here for traceability.

| Topic | Decision | Note vs. original ticket |
|---|---|---|
| Vault trust boundary | Full clone (read-all) + write-only `80_agents/hermes-nan/` enforced by instruction (PR #189). | Resolves open Q1. Read access is the point — the cross-project brain is the value. |
| Write boundary nature | Soft, instruction-enforced. The token can push anywhere; only the agent's rules confine writes. Accepted, documented residual. | New, explicit. |
| Secrets | `GITHUB_TOKEN_KNOWLEDGE` via env or `~/.hermes/.env` (chmod 600). **Not** the age-secrets system. | Resolves open Q2. Rationale: the remote box has neither the dotfiles repo nor the age master key, so age cannot serve it; the token is also needed to curl the bootstrap, before any local secret loader could run. |
| Token → git | Credential helper that sources `~/.hermes/.env`; serves both the interactive clone and the cron/non-interactive post-commit push. Token never written to `.git/config`. | New (security). |
| Vault clone path | Persistent `$HERMES_VAULT_PATH` (default `~/.local/state/hermes/vault`). | Revised from `/tmp/hermes-vault`. A durable bridge in ephemeral storage loses the auto-push hook on reboot — the agent's own recovery doc admits this fragility. |
| cron package | `setup.sh` may `apt-get install` it if absent (idempotent; box is root). | Revised from "NO sudo/apt". Relaxing apt is what makes one-shot idempotent recovery possible. |
| `setup.sh` scope | Full recovery protocol (clone/pull + hive-vault + crontab + post-commit hook + config patch + `.env` + final `validate.sh`). | Broader than the original bullet list, matching the agent's documented recovery steps. |
| Hive MCP registration | Native `hermes mcp add` CLI (idempotent via `hermes mcp list`); no hand-edited YAML. | Revised after box probe: product config lives at `/hermes-home/config.yaml` and exposes a CLI — safer than patching YAML. |
| Workspace registration | Ensure `00-context.md` (numbered convention); no `_index.md`. | Revised: the original `_index.md` does not match the folder convention. |
| Idempotence test | bats with an injection seam (`HERMES_SETUP_DRY_RUN` + stubbed network/`uv`/`apt`/`crontab`), runs twice, asserts no-op on the second run. | Resolves open Q3. CI-enforceable in dotfiles. |
| Canonical name | `hermes-nan` (lowercase). | Revised from `Hermes-NaN`. The live workspace already uses lowercase. |

## Out of scope

- **Installing the Hermes product itself** (`curl … install.sh` from Nous Research + `hermes setup --portal`). This spec layers vault integration on top of an already-installed Hermes.
- **Model-provider / portal OAuth credentials** — owned by the Hermes portal, not by `setup.sh`.
- **Wiring Hermes into `setup-linux.sh` / `setup-windows.ps1` / `mcp-servers.json`** — Hermes is remote and self-provisioning by design.
- **Downstream agent capabilities** tracked separately in the agent's own backlog (`80_agents/hermes-nan/30-tasks.md`): Himalaya email (SETUP-003), Headscale/Tailscale auth (SETUP-005), Ansible (SETUP-004).

## Risks / open questions

The design questions are resolved (see table above). Residual risks to keep visible:

- **Hive MCP env-passing flag.** Registration uses the native `hermes mcp add` CLI (product config at `/hermes-home/config.yaml`). One detail open: the flag to pass `HIVE_VAULT_PATH` to the hive-vault server (`hermes mcp add --help`) — without it the server may not locate the clone. Non-blocking: the agent operates over git today regardless of Hive.
- **Idempotence-test seam fidelity.** The bats stubs (network/`uv`/`apt`/`crontab`/`git`) must exercise the real idempotence guards, not short-circuit them — a mock that always returns success would make the test green while hiding a non-idempotent `setup.sh`. The seam must let the second run observe the first run's side effects.
- **Soft → hard write boundary.** The token can push anywhere in the vault repo; instructions alone are insufficient for a junior agent. Planned hardening (AC8): local `pre-commit`/`pre-push` hooks in Hermes's clone that mechanically reject out-of-zone writes, secret-like content, and force-push. **Gated on confirming Hermes commits via the git CLI** (so hooks fire) vs. the Hive MCP (which may bypass them) — if bypassed, enforcement moves to the Hive scope/server side. Residual beyond that: branch protection / pre-receive hook.
- **Cross-repo scope.** This work spans two repos (dotfiles + vault). The dotfiles CI can only gate the dotfiles track; the vault-curation track is verified by running `validate.sh` against the real vault, recorded in `verification.md`.

## Acceptance criteria

- [ ] **AC1** — `ai/hermes/setup.sh` passes `shellcheck` and parses clean under both `bash -n` and `zsh -n` (repo shell-compat standard).
- [ ] **AC2** — `ai/hermes/setup.sh` is idempotent: a bats test runs it twice under the dry-run/stub seam; both runs exit 0 and the second run produces no changes (no duplicate cron entry, no clobbered `config.yaml`, `.env` untouched).
- [ ] **AC3** — `ai/hermes/setup.sh` fails fast (non-zero exit + clear message) when `uv` is absent or `GITHUB_TOKEN_KNOWLEDGE` is unavailable (neither env nor `~/.hermes/.env`).
- [ ] **AC4** — `ai/hermes/AGENTS.md` exists and is a thin pointer to root `AGENTS.md` and the vault constitution, consistent with the `ai/agy/AGY.md` parity pattern (ADR-009).
- [ ] **AC5** — `setup.sh` modifies none of `setup-linux.sh`, `setup-windows.ps1`, `mcp-servers.json` (asserted by a guard in the test).
- [ ] **AC6** (vault track) — `80_agents/hermes-nan/scripts/validate.sh` passes against the real folder (filename checks match the numbered files); the canonical name `hermes-nan` and the vault constitution `AGENTS.md` are present and consistent. Verified vault-side, recorded in `verification.md`.
- [ ] **AC7** — `setup.sh` preflight aborts (non-zero) when the token cannot reach the vault remote; the cron auto-pull wrapper aborts a conflicted rebase rather than wedging the clone. Both covered by `tests/hermes-setup.bats`.
- [ ] **AC8** — mechanical write-zone enforcement in Hermes's clone (confirmed: Hermes commits via git CLI, so local hooks fire): a `pre-commit` hook rejects staged paths outside `80_agents/hermes-nan/` and token-like content; a `pre-push` hook forbids non-fast-forward (force) pushes. Covered by four functional tests in `tests/hermes-setup.bats`.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (HERMES-001 backlog entry); `80_agents/hermes-nan/` (the agent's live SSOT)
- GitHub: issue [#193](https://github.com/mlorentedev/dotfiles/issues/193)
- ADR: `docs/adr/adr-009-multi-agent-runtime.md` (cross-agent SSOT); ADR-012 (deploy = copy/patch, not symlink — the marker-patch idiom)
- Policy: PR #189 (`80_agents/` autonomous-agent commit policy)
- Family: HARNESS-001 / issue [#162](https://github.com/mlorentedev/dotfiles/issues/162) (multi-agent runtime)
- Product: <https://hermes-agent.nousresearch.com/docs>
