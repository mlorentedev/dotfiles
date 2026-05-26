---
tags: [spec, tasks, sdd-007]
created: "2026-05-25"
---

# Tasks - SDD-007-iac-deploy-strategy

> TDD order. One task = one focused commit (user commits). Tick as you go. Big-bang single-PR per user decision (2026-05-25).

## Setup

- [x] Branch: `feature/antigravity-integration` (existing — scope expanded from BUG-100 fix)
- [x] `proposal.md` complete with R3 healthcheck-noise resolved (`DEPLOYED_FILES` registry)
- [x] R5 confirmed: no other code path writes to `~/.zshrc` / `~/.bashrc` post-deploy (`grep -rn 'HOME/.zshrc\|HOME/.bashrc' scripts/` returns only setup-linux.sh + the new healthcheck reader)

## Implementation

> TDD order. Each task is a clean commit.

### 1. deploy_file helper (foundation)

- [ ] Write `tests/iac-deploy.bats` with 4 failing cases: fresh-deploy creates file, idempotent second-run skips (zero log lines), source-change triggers re-deploy with log, atomic-rollback when source is unreadable
- [ ] Implement `deploy_file SRC DEST` in `scripts/utils.sh` (tempfile + `cmp -s` skip + `mv` atomic)
- [ ] Run `~/.local/bin/bats tests/iac-deploy.bats` → 4/4 PASS

### 2. setup-linux.sh migration (the core)

- [ ] Replace ~14 dotfile-deploy `ln -sf` calls with `deploy_file` (lines 57, 58, 63, 70, 79, 80, 81, 82, 149 — config files only; lines under `link_vault_skills` and `deploy_auto_memory_symlinks` STAY as symlinks per Out-of-scope)
- [ ] Add `DEPLOYED_FILES` array (associative or paired array — bash+zsh compatible) listing every `repo→home` pair touched by `deploy_file`
- [ ] BUG-100 fix in same pass: master config to `~/.gemini/config/mcp_config.json` (was: `~/.gemini/mcp_config.json`), drop sibling symlink at `$AGY_APP_DATA/mcp_config.json` (was a symlink; now just don't write there — `agy` reads from `~/.gemini/config/` per docs)
- [ ] Remove `agy plugin import gemini` from setup (line 416) — migration is one-time, not per-setup
- [ ] Run `shellcheck setup-linux.sh` + `bash -n setup-linux.sh` + `zsh -n setup-linux.sh` → all clean

### 3. healthcheck.sh updates

- [ ] Replace `check_symlink` calls for managed dotfiles (`.zshrc`, `.bashrc`, `.zsh/*.zsh`, ssh/config, etc.) with new `check_deployed REPO_REL HOME_PATH` function (cmp content)
- [ ] Antigravity section: assert `~/.gemini/config/mcp_config.json` is a regular file (not symlink), valid JSON, and `readlink -e` resolves in ≤1 hop
- [ ] Drop the "agy mcp_config.json should be a symlink to master" assertion (no longer applicable)
- [ ] Add `check_no_symlinks_under ~/.gemini/config/` loop
- [ ] Run `./scripts/healthcheck.sh` → 0 failures on freshly re-deployed system

### 4. setup-windows.ps1 parity

- [ ] Repoint Windows master MCP config to `$env:USERPROFILE\.gemini\config\mcp_config.json` (was elsewhere — verify current path first)
- [ ] Verify no non-ASCII chars introduced (pattern-powershell-ascii-only — hit twice already)
- [ ] Update Windows healthcheck section to mirror the no-symlinks-under-config assertion (Windows-side check_deployed equivalent uses `Compare-Object`)

### 5. NaN integration (co-shipped)

- [ ] `sensitive/env-mapping.conf`: add line `NAN_API_KEY=nan.api-key` (user encrypts the actual key separately with `age`)
- [ ] `ai/nan/README.md`: provider docs (base URL, model catalog, rate limits, link to https://nan.builders/docs)
- [ ] `ai/opencode/opencode.jsonc`: replace `opencode-go` provider block with `nan` provider (3 chat models from NaN catalog), keep `openrouter` as fallback, change `model` default to `nan/qwen3.6`
- [ ] `.zsh/aliases.zsh` + `.bashrc` + `powershell/profile.ps1`: add `qqn` quick-question alias backed by `nan/qwen3.6` + `export NAN_BASE_URL="https://api.nan.builders/v1"`
- [ ] Comment in opencode.jsonc: Ollama provider slot (commented-out, with `http://<homelab-ip>:11434/v1` placeholder) ready for next session
- [ ] Manual smoke (user): `oc /models` → NaN models listed; `qqn "hola"` → response

### 6. .gitignore + .geminiignore

- [ ] `.gitignore`: add `.antigravitycli/`, `.antigravitycli.bak/` (currently leaking into git status)
- [ ] `.geminiignore`: replace current absurd `*` + `!README.md` with sensible content (`sensitive/`, `.secrets/`, `node_modules/`, `*.secret.age`, `.git/`)
- [ ] `deploy_file` for `.geminiignore` → `~/.gemini/.geminiignore`

### 7. CLAUDE.md workflow rule

- [ ] `.claude/CLAUDE.md` (project): add row to workflow table — "edit configs in repo, run setup-linux.sh; never edit `~/.zshrc` directly (drift)"
- [ ] `~/.claude/CLAUDE.md` (global, deployed from `ai/claude/CLAUDE.md`): same rule under "Project Memory Hierarchy" or new "Deploy Discipline" section

### 8. Tests

- [ ] `tests/iac-deploy.bats`: deploy_file unit tests (covered in step 1)
- [ ] `tests/antigravity.bats`: BUG-100 regression — `~/.gemini/config/mcp_config.json` is not a symlink + no symlinks under config/
- [ ] `tests/setup-linux.bats` (if exists): update any tests that assumed `~/.zshrc` was a symlink
- [ ] Full suite: `~/.local/bin/bats tests/*.bats` → all green

## Closing

- [ ] Every AC1-AC6 covered by at least one test or verification.md row
- [ ] `shellcheck scripts/*.sh setup-linux.sh` clean
- [ ] PSScriptAnalyzer clean on `setup-windows.ps1` (no non-ASCII)
- [ ] `./scripts/healthcheck.sh` → all green, including new drift checks
- [ ] Second `./setup-linux.sh` run → zero "deployed" log lines (AC5 idempotency)
- [ ] No `ln -sf` remains for dotfile config (AC1)
- [ ] `verification.md` filled in with evidence per AC
- [ ] PR opened from `feature/antigravity-integration`, references this spec folder, closes issue #100

## Post-merge follow-ups (discovered exercising the merged PR #102 on Windows)

Three bugs surfaced when re-running setup-windows.ps1 on a real Windows machine after PR #102 merged — all from the same SDD-007 migration that wasn't completed end-to-end. Addressed in PR #105 (this PR):

- [x] **MCP loop StrictMode-safe**. `$srv.prerequisite_binary` threw under `Set-StrictMode -Version Latest` for every MCP entry that omits the field (only `hive` declares it). Linux side already handles this via `jq '(.prerequisite_binary // "")'`. Fix: probe `PSObject.Properties['name']` first so missing fields read as `$null` instead of erroring. — `setup-windows.ps1:380-394`.
- [x] **GEMINI.md orphan cleanup**. Pre-SDD-007 installs deployed `~/.gemini/GEMINI.md`; the rename to `AGY.md` (sec 6) never removed the old file. Added explicit `rm -f` in both `setup-windows.ps1` and `setup-linux.sh` before the AGY.md deploy (idempotent — no-op if absent). — `setup-windows.ps1:857-863`, `setup-linux.sh:407-413`.
- [x] **Lingering GEMINI.md references** in tooling that asserted the old name post-rename:
  - `scripts/healthcheck.ps1:226` — Test-DeployedFile checked `GEMINI.md` → always FAIL. Updated to `AGY.md`.
  - `scripts/init-project.ps1:225-231, 621` — injected `GEMINI.md` into new projects + summary text. Updated to `AGY.md` (Linux side at `init-project.sh:186-191` was already correct — was missing parity).

Bats regression coverage:

- [x] `tests/setup-windows.bats`: two new tests — StrictMode-safe MCP probe; GEMINI.md cleanup line present.
- [x] `tests/init-project-ps1.bats`: assert AGY.md injection + regression guard against the old GEMINI.md path returning (`! grep 'Injected GEMINI\.md'`).

Empirical verification on a real Windows machine: two consecutive `setup-windows.ps1` runs produced identical state — second run's only difference was the `[INFO] Removed legacy GEMINI.md` line on the first run. AC5 idempotency holds on Windows.

## Machine-readable features

Optional sibling `features.json` per [[pattern-feature-list-as-primitive]] — defer unless harness consumes it.
