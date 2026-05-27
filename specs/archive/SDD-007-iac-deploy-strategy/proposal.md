---
id: "SDD-007-ai-tooling-consolidation"
type: spec
status: archived
created: "2026-05-25"
archived: "2026-05-26"
tags: [spec, proposal, sdd-007, cross-os, iac, deploy-strategy, ai-tooling, bug-100-resolution, audit-002-followup, tool-consolidation]
template_version: "1.0"
---

# SDD-007: AI tooling consolidation + IaC deploy strategy

> **Scope merge note (2026-05-25)**: this spec originally targeted only the symlink→copy migration. User decision mid-scope: also (1) **drop legacy Gemini CLI** (`@google/gemini-cli` npm binary) and **drop Aider** entirely; (2) **keep `agy` (Antigravity)** as the rightful replacement for legacy gemini-cli, but treat it as a **fresh install from scratch** — no `agy plugin import gemini` carryover, no dual-compat Gemini-CLI settings.json. OpenCode + Claude Code + agy are the AI tools going forward. All architectural changes ship in one PR per user request ("big-bang en un solo PR"). BUG-100 still in scope because agy stays; the IaC pattern's main test case is precisely "stop fighting agy's filesystem expectations".

> **Naming**: file lives at `<repo>/specs/SDD-007-iac-deploy-strategy/proposal.md`.

## Why

**Two threads converge here.**

**(A) Legacy AI tooling is paying double-cost.** The repo currently maintains:
- `agy` (Antigravity CLI) — Google's official rebrand and replacement for the legacy `@google/gemini-cli`. MCP-capable, skills-capable. **Stays.** v1.0.2 has known filesystem fragility (BUG-100 / forum #145851 EEXIST / gemini-cli #10960 symlink clobber) but those are caused by *our deploy strategy fighting agy's expected layout*, not by agy itself being defective. The fix is to stop putting symlinks where agy writes-in-place, and to write the master MCP config at `~/.gemini/config/mcp_config.json` (where agy actually reads from per agentpedia / Composio / Dazbo docs), not at `~/.gemini/mcp_config.json` (where the current BUG-100 patch puts it).
- **Legacy `gemini-cli`** (`@google/gemini-cli` npm package) — the *old* Google CLI that `agy` replaces. setup-linux.sh currently writes a Gemini-CLI-compatible `~/.gemini/settings.json` (line 295-298) and runs `agy plugin import gemini` (line 416) on every setup, both of which assume a legacy installation exists. On a fresh machine those are noise; on the user's machine they're stale. **Removed.**
- `aider` — Python-based AI coding agent, already marked "sunset PR2" in MEMORY.md, partially purged (`ai/aider/` dir already gone) but with `ai`/`aic`/`aia` aliases still wired in `.zsh/aliases.zsh`/`.bashrc`/`powershell/profile.ps1` plus test scaffolding still asserting their presence. **Removed.**
- `opencode` (primary daily) — installed by setup-linux.sh:528-545 via curl install; under-utilized today because user is unsatisfied with the OpenCode Go subscription tier. **NaN community becomes the new default provider**, OpenRouter stays as frontier fallback, Ollama (homelab/VPN endpoint, user-managed in separate session) slot left commented in opencode.jsonc.
- `claude code` — primary architecture/debug tool. **Untouched.**

OpenCode covers everything aider delivered; agy covers the Gemini-family slot legacy gemini-cli used to fill. The user's decision: **collapse to {agy, opencode, claude code}** and stop maintaining legacy-gemini-cli + aider parallel paths.

**(B) BUG-100 (Antigravity `mcp_config.json` circular symlinks, fix 8a5072d *still leaking*) is not a one-off** — it's the recurring tax of mixing two deploy paradigms in one repo. `setup-linux.sh` uses `ln -sf` for ~14 config files (`.zshrc`, `.bashrc`, `tmux.conf`, `.gitconfig`, ssh/config, .zsh/*.zsh, etc.) while `setup-windows.ps1` already copies every equivalent file because Windows symlinks need dev-mode + admin. The mental model is therefore **asymmetric across OS** and **fragile against any modern CLI that rewrites its own config** — already documented for `agy` (issue #10960 clobbers `~/.gemini/settings.json` symlink, forum #145851 EEXIST on `mcp_config.json`), and the same class hits VSCode, Cursor, gh, and any tool that emits its own settings on first launch. Modern dotfile managers (chezmoi, home-manager, dotbot --copy, yadm) are **all copy-based with state reconciliation** for this reason. The current hybrid is the worst of both worlds: live editing convenience on Linux only, plus a cascade of symlink-clobber bug classes that re-open every time a tool ships a new write-in-place behavior.

## What

Introduce `deploy_file SRC DEST` in `scripts/utils.sh` (atomic cp via tempfile+mv, idempotent: skip if `cmp -s` matches, log only on change). Replace all ~14 config-deploy `ln -sf` calls in `setup-linux.sh` with `deploy_file`. The Antigravity section uses the same helper and writes the master MCP config to `~/.gemini/config/mcp_config.json` (the path `agy` actually reads from per agentpedia/Dazbo/Composio docs) instead of the current flat-file at `~/.gemini/mcp_config.json` with a sibling symlink. `setup-windows.ps1` already uses copies; this PR only aligns its target paths for `~/.gemini/config/`. Workflow consequence: **edits happen in the repo, never in `~`**. The two-tier deploy verification rule (currently a "guardrail when remembered") becomes a universal precondition trivially enforced by drift assertions in `scripts/healthcheck.sh`.

## Out of scope

- **Vault→home symlinks** under `link_vault_skills` / `deploy_auto_memory_symlinks` (setup-linux.sh lines ~902-1010): these point `~/Projects/knowledge/...` (Obsidian vault) → `~/.claude/projects/<hash>/memory/`. They are intentional **cross-system bindings**, not config deploy. Vault is the canonical store; copying would create drift between vault and Claude memory. Stays as symlinks.
- **Secret file deployments via `secrets_add_file`** (`~/.ssh/id_ed25519`, `~/.config/age/key.txt`): these need canonical absolute paths for OpenSSH/age to find them; the secrets loader already handles them separately from config deploy.
- **Convenience binary symlinks** (e.g., `python → python3` in versioned Python install). Tool-internal, not dotfile-managed.
- **AI-010 Ollama install** — separate spec, blocked on homelab/VPN endpoint setup (user owns externally).
- **Cancelling the OpenCode Go subscription** — manual action in Zen dashboard, not codifiable here.

## Risks / open questions

- **R1 (BLOCKER — must be assertion-covered before merge):** Edit-in-`~` regression. After this PR, `vim ~/.zshrc; source ~/.zshrc` makes a change live for the current session but the change is **lost on next `setup-linux.sh` run**. Mitigation: `scripts/healthcheck.sh` adds a `check_deployed` assertion (per managed file: `cmp -s repo/path ~/path`); diff means user edited `~` directly. This converts silent drift into a loud red healthcheck. **Without this, the migration is net-negative** because losing an edit silently is worse than a symlink clobber that surfaces as an error.
- **R2:** First-run setup-linux.sh wall-time increase. Bulk `cp` vs `ln -sf` adds ~200-400ms across 14 files. Acceptable. No mitigation needed.
- **R3 (BLOCKER — must resolve before tasks.md freeze):** Healthcheck noise. Today some files in `~/.claude/` and `~/.gemini/` are intentionally clobbered by CLIs (claude-session-start state, agy plugin caches). If `check_deployed` flags every CLI-managed file as drift, the signal/noise collapses. **Solution**: maintain an explicit `DEPLOYED_FILES` array in `scripts/utils.sh` or `setup-linux.sh`. Healthcheck iterates that array only — files not in it are CLI-managed and excluded from drift detection. Trade-off: adding a new deployed file is now a two-place edit (`deploy_file` call + `DEPLOYED_FILES` entry). Decision: accepted; better than the alternative.
- **R4:** `.zshrc` workflow muscle memory. Years of "edit ~/.zshrc directly" reflex won't die in one session. Mitigation: shell alias `dotfiles-edit` (or just `dot`) opens `$DOTFILES_DIR` in `$EDITOR`; documented in `.claude/CLAUDE.md` workflow table. Drift detector catches forgotten cases.
- **R5 (open):** `BUG-024` was about drift-detector false-positives on rc files that setup writes. With this PR, **setup is the only writer** of rc files (since they're no longer symlinks pointing to the repo — they're copies of repo content). BUG-024's `ensure_line_in_file`-into-symlinked-rc trap goes away. Confirm no other code path in scripts/ writes to `~/.zshrc` / `~/.bashrc` post-deploy. (Spot-check: `grep -rn 'HOME/.zshrc\|HOME/.bashrc' scripts/` before merge.)

## Acceptance criteria

- [x] **AC1**: No `ln -sf` invocations for config-deploy paths in `setup-linux.sh` after this PR. Vault→home symlinks (memory/skills) remain. Verified by: `grep -E '^\s*ln -sf' setup-linux.sh | grep -vE 'vault|skills|memory' | wc -l` returns 0.
- [x] **AC2**: `deploy_file SRC DEST` exists in `scripts/utils.sh`, is atomic (tempfile + mv), idempotent (skip if cmp matches), logs only on actual change. Covered by `tests/iac-deploy.bats` (≥4 test cases: fresh-deploy, idempotent-second-run, change-detection, atomic-rollback-on-error).
- [x] **AC3**: `scripts/healthcheck.sh` asserts content equivalence between each entry of a `DEPLOYED_FILES` registry and its target in `~`. Drift = healthcheck fail. Excludes CLI-managed files (anything under `~/.gemini/antigravity-cli/` / `~/.claude/projects/`).
- [x] **AC4 (BUG-100 closure)**: `~/.gemini/config/mcp_config.json` is a **regular file** (not a symlink), readable as valid JSON, contains all servers from `mcp-servers.json`. No path under `~/.gemini/config/` is a symlink. `readlink -e ~/.gemini/config/mcp_config.json` resolves in ≤1 hop (i.e., is the file itself). `tests/antigravity.bats` adds a regression test for this. Issue #100 closeable.
- [x] **AC5**: `setup-linux.sh` is idempotent on the local dev machine: running it twice produces zero `cp` log lines on the second run (everything already up-to-date). Verified by manual smoke test logged in `verification.md`.
- [x] **AC6 (cross-OS parity)**: `setup-windows.ps1` writes its master MCP config to `$env:USERPROFILE\.gemini\config\mcp_config.json` (mirror of Linux path), not the previous Windows-specific location. Existing Windows healthcheck section is updated.

## Completeness review

Standard items:
- **Rate limit / cost guard**: N/A (local-only deploy operations).
- **Idempotency**: explicitly required by AC5 + tested by AC2.
- **Regression test**: BUG-100 covered by AC4; broader drift detection by AC3.
- **Rollback plan**: All changes ship in one PR. `git revert` on merge commit restores `ln -sf` strategy AND keeps `~/.gemini/config/` populated as files (no harm — `agy` will repopulate or read its own copies). Symmetric undo.

Adding (load-bearing for this PR):
- **Workflow migration note**: this PR ships a CLAUDE.md update documenting the edit-in-repo rule. Without that doc change, the muscle-memory reversion to `vim ~/.zshrc` will produce silent drift that the healthcheck catches but the user might dismiss as "test noise".
- **NaN integration co-shipped**: the user is replacing the OpenCode Go subscription with NaN community as default provider in the same PR window. Co-ships because: (a) `opencode.jsonc` deploy already uses `cp` (no symlink involved), so it's a clean "first new deploy via the new helper" example; (b) keeps the disruption to a single rebuild/healthcheck cycle for the user. Out-of-band: secret encryption (`age -r ... < nan-api-key > sensitive/nan.api-key.secret.age`) is a user-owned step.

## References

- Vault: `10_projects/dotfiles/30-architecture/audit-002-cross-os-duplication.md` (root-cause framing for cross-OS divergence)
- Lesson: `10_projects/dotfiles/90-lessons.md` lesson #9 (BUG-024: setup-time mutations to repo-symlinked files create permanent drift false-positives) — this PR generalizes that lesson into the deploy architecture.
- Lesson: `10_projects/dotfiles/90-lessons.md` lesson #7 (verify-before-act on agent audits) — explicitly applied here: the web research surfaced multiple symlink-clobber bug classes; the audit-once response is architectural, not file-by-file.
- Issue: GitHub `#100` (Antigravity CLI Circular Symlink Recursion on mcp_config.json) — closed by AC4.
- Pattern: `00_meta/patterns/pattern-iac-dotfiles.md` — **new pattern candidate** (see verification.md §Promotion candidates), since the symlink→copy migration is reusable across other dotfile-managed projects.
- External: chezmoi (https://chezmoi.io), home-manager (NixOS), agentpedia "Antigravity 1.21.6 fixes" (https://agentpedia.codes/blog/antigravity-1-21-6-update-issues-fix) confirming symlink advisory.
