---
tags: [spec, verification, claude-mem, plugin-discovery]
created: "2026-05-20"
---

# Verification - BUG-012-claude-mem-marketplace-junction

## Evidence (per acceptance criterion)

- [x] **`claude-mem-heal.sh` symlink block**: `ensure_marketplace_compat_symlink()` defined at `scripts/claude-mem-heal.sh:39-56`, invoked at l.135 before `heal_dir` walks. Guards: source dir exists, target absent. `ln -s` with both `-e` and `-L` checks on the legacy path.
- [x] **`claude-mem-heal.ps1` junction block**: `Repair-MarketplaceCompatJunction` at `scripts/claude-mem-heal.ps1:69-88`, invoked at l.180. Uses `New-Item -ItemType Junction` (no admin required). Guards: `Test-Path -PathType Container` on source, `Test-Path` on target.
- [x] **Path-aware walk**: both heal scripts now iterate the legacy AND actual marketplace paths (`MARKETPLACE_DIR_ACTUAL` / `$MarketplaceDirActual`). Idempotent — internal heal functions grep before write.
- [x] **Bats assertions**: 4 new asserts in `tests/setup-linux.bats` (BUG-012 block) + 2 in `tests/setup-windows.bats`. Manual grep equivalents PASS post-implementation.

## Test status

- **Pre-fix state on user's Windows machine** (captured during diagnosis):
  ```
  marketplaces/claude-plugins-official/   ← exists (matches name)
  marketplaces/thedotmack-claude-mem/     ← exists (Claude Code's repo-based naming)
  marketplaces/thedotmack/                ← MISSING (what plugin's hooks.json fallback expects)
  cache/thedotmack/claude-mem/            ← MISSING (cache never populated)
  ```
  Hook failure consequence:
  ```
  UserPromptSubmit operation blocked by hook:
    /usr/bin/bash: line 1: printf: write error: Permission denied
  ```
  Translation: hook discovery script fell through all fallback paths → `_P` empty → `exit 1` → operation blocked.

- **Post-fix empirical result on user's Windows machine** (run 1, dry repair):
  ```
  BEFORE: thedotmack exists = False
  [claude-mem-heal] created legacy marketplace junction:
    C:\Users\Manu\.claude\plugins\marketplaces\thedotmack -> thedotmack-claude-mem
  AFTER: thedotmack exists = True
  LinkType: Junction
  Target: C:\Users\Manu\.claude\plugins\marketplaces\thedotmack-claude-mem
  ```
- **Idempotency** (run 2, no -VerboseOutput): no output, exit 0. With -VerboseOutput: `legacy marketplace path already present`. Confirms the silent-on-healthy contract is preserved.
- **PowerShell AST parse** of `claude-mem-heal.ps1`: clean. **PSScriptAnalyzer -Severity Error**: clean.
- **`bash -n` of `claude-mem-heal.sh`**: clean.
- **Linux empirical**: not run (no Linux test machine in this session). The symlink logic mirrors the PowerShell Junction logic with identical guards; bash syntax checked.

## Decisions made during implementation

- **Junction vs symbolic link on Windows**: junction (no admin required, works on NTFS, directory-only — sufficient here). Matches the precedent in `setup-windows.ps1` auto-memory deploy.
- **Skip the upstream plugin's `hooks.json` rewrite**: it would be reverted on every `/plugin update`. Junction is one-shot and outlasts plugin updates.
- **Heal cache root only when present**: `cache/thedotmack/claude-mem/` populated by Claude Code on plugin install/load; if missing we simply don't iterate it (existing behavior preserved).
- **Cross-OS parity**: applied identical logic to both heal scripts even though only Windows symptom was empirically observed. Marketplace dir naming is Claude Code-driven, not OS-driven; Linux exposure is theoretical but real.

## Promotion candidates

To be assessed post-merge.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived` (post-merge).
- [ ] Folder moved to `specs/archive/BUG-012-claude-mem-marketplace-junction/` (post-merge).
- [ ] Vault `11-tasks.md` entry created with PR link (retroactive, post-merge).
- [ ] Vault `90-lessons.md` appended: "plugin discovery: marketplace dir name follows GitHub repo, not declared `name` field — heal both paths".
