---
tags: [spec, tasks]
created: "2026-05-31"
---

# Tasks - HERMES-001-add-hermes-agent

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> Two tracks. **Track A (dotfiles)** ships in this PR and is gated by dotfiles CI. **Track B (vault curation)** is staged in the vault and verified vault-side (`validate.sh`); it is not gated by dotfiles CI.

## Setup

- [x] Branch created from main: `feat/add-hermes-agent`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No blocking open questions left in `proposal.md` "Risks / open questions" (the Hermes config-schema risk blocks only the config-patch task — sequence it after verification)

## Track A — dotfiles (this PR, TDD order)

- [x] Bring the untracked `pattern-loader` skill under version control at its canonical dotfiles location (decide `ai/skills/` vs `harness/skills/`); fix its hardcoded `/tmp/hermes-vault` to `$HERMES_VAULT_PATH`
- [x] Write failing bats: `setup.sh` fails fast (non-zero + message) when `uv` missing OR `GITHUB_TOKEN_KNOWLEDGE` absent (AC3)
- [x] Write failing bats: `setup.sh` is idempotent under the dry-run/stub seam — run twice, both exit 0, second run is a no-op (AC2)
- [x] Write failing bats/guard: `setup.sh` references/modifies none of `setup-linux.sh`, `setup-windows.ps1`, `mcp-servers.json` (AC5)
- [x] Implement `ai/hermes/setup.sh` skeleton: arg/flag parse, `HERMES_SETUP_DRY_RUN` seam, prereq checks (`uv`, token via env/`~/.hermes/.env`) — make AC3 green
- [x] Implement clone/pull to `$HERMES_VAULT_PATH` via env-sourced credential helper (no token in `.git/config`) — idempotent (clone-if-absent-else-`pull --ff-only`)
- [x] Implement `uv tool install --upgrade hive-vault` (idempotent by `--upgrade`)
- [x] Implement vault-sync install: userspace `crontab` pull entry (insert-once) + `post-commit` auto-push hook; `apt-get install cron` only if absent
- [x] Implement `~/.hermes/.env` ensure (chmod 600; write only if var present and not already recorded) and `config.yaml` marker-patch stub
- [x] Refactor `setup.sh` for clarity; confirm `shellcheck` + `bash -n` + `zsh -n` clean (AC1) — make AC2/AC5 green
- [x] Write `ai/hermes/AGENTS.md` thin pointer (root `AGENTS.md` + vault constitution), parity with `ai/agy/AGY.md` (AC4)

## Track A — safety net (junior agent: mechanical guardrails)

- [x] Preflight token/remote auth-check: abort before any heavy mutation (AC7) + test
- [x] Robust auto-pull wrapper (`vault-pull.sh`): abort conflicted rebase, never wedge (AC7) + test
- [x] Full idempotence test (DRY_RUN=0, stubbed git/crontab): two runs no-op, no dupes (AC2)
- [x] Block-2 confirmed: Hermes commits via git CLI (HOOKS FIRE) — git-hook enforcement is valid
- [x] `pre-commit` hook: reject staged paths outside `80_agents/hermes-nan/` (AC8) + functional test
- [x] `pre-commit` secret-scan: reject token-like content (AC8) + functional test
- [x] `pre-push` hook: forbid non-fast-forward / force-push (AC8) + test

## Track A — box-reality hardening (from the 2026-05-31 Hermes probe)

- [x] Strip the token embedded in the vault remote URL (`x-access-token:***@`) — move it to the credential helper, out of `.git/config`
- [x] Migrate the stale `/tmp/hermes-vault` inline cron line to the robust `vault-pull.sh` wrapper (single entry)
- [x] Register Hive MCP via native `hermes mcp add` (replaces the YAML-patch plan; idempotent via `hermes mcp list`)
- [x] Confirm `hermes mcp add` env-passing flag so `HIVE_VAULT_PATH` reaches the hive-vault server (ask: `hermes mcp add --help`)

## Track B — vault curation (staged in vault, verified by validate.sh)

- [x] Fix `80_agents/hermes-nan/scripts/validate.sh`: check the real numbered filenames (`10-memory.md`, `11-skills.md`, `12-cronjobs.md`, `13-config.md`, `20-servers.md`) and the persistent vault path
- [x] Create `80_agents/hermes-nan/AGENTS.md` (vault constitution): thin pointer to root `AGENTS.md` + Hermes operating law (write-zone, sync discipline, bootstrap protocol)
- [x] Reconcile naming to `hermes-nan` and Hive scope to `agents:hermes-nan` across the folder
- [x] Align bootstrap/recovery docs (`00-context.md`, `20-servers.md`, `13-config.md`, `14-env-variables.md`) to the reconciled `setup.sh` (persistent `$HERMES_VAULT_PATH`, apt-if-absent, marker-patched config, no `_index.md`)
- [x] Update the structure map in `00-context.md` and leave a `sessions/2026-05-31-*.md` record of the reorg so the agent re-discovers state
- [x] Run `validate.sh` against the real folder → exit 0 (AC6)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test or vault-side check
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Lint passes: `shellcheck ai/hermes/setup.sh`; `bats` suite green
- [x] No unrelated changes in the diff (no scope creep); `setup-linux.sh` / `mcp-servers.json` untouched
- [x] `verification.md` filled in with evidence (commit hashes, test output, vault `validate.sh` result)
- [x] PR opened referencing this spec folder

## Machine-readable features

See sibling `features.json`. Each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence`.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state.
