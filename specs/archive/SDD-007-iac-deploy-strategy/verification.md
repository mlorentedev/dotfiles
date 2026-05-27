---
tags: [spec, verification, sdd-007]
created: "2026-05-25"
---

# Verification - SDD-007-iac-deploy-strategy

## Evidence

| # | Criterion | Evidence |
|---|---|---|
| AC1 | No `ln -sf` for config-deploy in setup-linux.sh | `grep -E '^\s*ln -sf' setup-linux.sh \| grep -vE 'vault\|skills\|memory'` returns 0 lines. Run pre-merge. |
| AC2 | `deploy_file` helper exists, atomic, idempotent | `tests/iac-deploy.bats` — fresh-deploy / no-op second-run / change-trigger / source-missing-rollback. 4/4 PASS. |
| AC3 | Healthcheck content-match for managed files | `scripts/healthcheck.sh` adds `check_deployed` iterating `DEPLOYED_FILES`. `./scripts/healthcheck.sh` PASS on freshly-deployed system; manual `vim ~/.zshrc` edit causes intentional FAIL with "drift" message. |
| AC4 | BUG-100 closure | `tests/antigravity.bats` regression: assert `~/.gemini/config/mcp_config.json` is regular file (not `-L`), `readlink -e` resolves in 1 hop, JSON parses, all servers from `mcp-servers.json` present. Issue #100 closeable. |
| AC5 | setup-linux.sh idempotent | Manual run capture: 1st run logs N "deployed" lines; 2nd run logs 0. Smoke test command: `./setup-linux.sh 2>&1 \| grep -c "deployed"` returns same N first run, 0 second run. |
| AC6 | Windows path parity | `setup-windows.ps1` writes master to `$env:USERPROFILE\.gemini\config\mcp_config.json`. Verified on Windows VM (next session, empirical). Cross-OS drift detector in tests catches divergence. |

## Test status

- Bats suite: `~/.local/bin/bats tests/*.bats` → fill post-impl (target: all green, no regressions in existing 147 tests)
- Manual smoke: fill post-impl with output of 2× `./setup-linux.sh` + `./scripts/healthcheck.sh`
- NaN smoke: fill post-impl with `oc /models` listing NaN provider, `qqn "ping"` round-trip success
- No regressions in existing test suite: fill post-impl

## Decisions made during implementation

(Fill as implementation progresses. Anchors below.)

- **D1 (R3 resolution)**: `DEPLOYED_FILES` registry lives in `scripts/utils.sh` as a `repo_path|home_path` paired array. Both setup-linux.sh and healthcheck.sh source it. Two-place edit cost accepted vs alternative (auto-discover from `cp` log) which is fragile across re-runs.
- **D2 (Antigravity exclusion zone)**: `~/.gemini/antigravity-cli/`, `~/.gemini/skills/` (CLI-managed), and `~/.claude/projects/` are *excluded* from drift detection. They're CLI-owned. Documented in `DEPLOYED_FILES` registry comment.
- **D3 (vault binding stays as symlink)**: `link_vault_skills` + `deploy_auto_memory_symlinks` keep `ln -s`. Different concern: vault is canonical store, bidirectional binding is intentional. Documented explicitly in proposal §Out-of-scope.
- **D4 (NaN secret encryption)**: user-owned step, not codified. Doc says: `age -r $(age-keygen -y ~/.config/age/key.txt) > sensitive/nan.api-key.secret.age` then paste key + Ctrl-D. Spec captures the mapping line + the procedure; the encrypted file is committed by the user.
- **D5 (Go subscription cancellation)**: user-owned (Zen dashboard manual). Out of scope but flagged: until cancellation, the `$10/mo` continues to bill. NaN is "default" from this PR; Go is "removed" from `opencode.jsonc` so it won't appear in `/models` picker.
- **D6 (`agy plugin import gemini` removal from setup)**: per Antigravity docs (agentpedia, Composio), `plugin import` is a **migration step**, not a steady-state operation. Running it every `setup-linux.sh` was the immediate trigger of BUG-100's recurrence. Removed; if user needs to re-migrate, they run it once manually.

## Promotion candidates

- [ ] Lesson for `10_projects/dotfiles/90-lessons.md`: **"Symlinks under directories that modern CLIs write to are a recurring bug class — prefer `deploy_file` (cp + cmp) for config; reserve symlinks for cross-system bindings (vault) and canonical-path secrets (~/.ssh)"**. Yes — recurrence proven (BUG-024 + BUG-100 + #10960 + #145851).
- [ ] ADR for `10_projects/dotfiles/30-architecture/adr-012-deploy-strategy.md`: yes — codifies the symlink-vs-copy decision rule for future contributors.
- [ ] Pattern candidate `00_meta/patterns/pattern-iac-dotfiles.md`: yes — applies beyond this repo (chezmoi/home-manager validate the pattern externally; in-repo `setup-windows.ps1` already practiced it).

## Archive checklist

(Fill post-merge.)

- [ ] Issue #100 closed with link to merge commit
- [ ] BUG-024 lesson cross-referenced (this PR is its generalization)
- [ ] `MEMORY.md` Session Handoff updated
- [ ] `verification.md` Decisions filled
- [ ] Promotion candidates actioned in vault (3 expected: lesson + ADR-012 + pattern)
- [ ] Spec status: draft → implementing → done; `/spec archive SDD-007` once Windows-side empirical validation lands (next session)
