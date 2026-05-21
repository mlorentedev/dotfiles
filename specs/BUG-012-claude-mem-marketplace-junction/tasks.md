---
tags: [spec, tasks, claude-mem, plugin-discovery]
created: "2026-05-20"
---

# Tasks - BUG-012-claude-mem-marketplace-junction

## Setup

- [x] Branch: `fix/BUG-012-claude-mem-marketplace-junction` (off main).
- [x] Spec scaffold (manual — bypassed init-spec.ps1 vault gate for small-scope bugfix).

## Implementation (TDD order)

### Tests first

- [ ] `tests/setup-linux.bats`: assertion that `scripts/claude-mem-heal.sh` contains a guarded `ln -s ... thedotmack-claude-mem ... thedotmack` block.
- [ ] `tests/setup-windows.bats`: assertion that `scripts/claude-mem-heal.ps1` contains a guarded `New-Item -ItemType Junction ...` block referencing both marketplace names.
- [ ] Cross-OS parity assertion: both scripts implement the same junction-creation guard (mirrors existing BUG-004/BUG-011 parity pattern).
- [ ] Run bats — assertions should FAIL (red).

### Implementation

- [ ] `scripts/claude-mem-heal.sh`: add `ensure_marketplace_compat_symlink()` function. Guard: only create if `marketplaces/thedotmack/` missing AND `marketplaces/thedotmack-claude-mem/` exists. Logs one line on creation; silent if already present or sources missing.
- [ ] `scripts/claude-mem-heal.sh`: extend `MARKETPLACE_DIR` walk to BOTH legacy (`thedotmack/plugin`) and current (`thedotmack-claude-mem/plugin`) paths.
- [ ] `scripts/claude-mem-heal.ps1`: equivalent `Ensure-MarketplaceCompatJunction` function using `New-Item -ItemType Junction`.
- [ ] `scripts/claude-mem-heal.ps1`: extend `$MarketplaceDir` walk similarly.
- [ ] Run bats — assertions now PASS (green).

### Lint + cross-check

- [ ] `shellcheck --severity=error scripts/claude-mem-heal.sh` clean.
- [ ] `pwsh -Command "Invoke-ScriptAnalyzer -Path scripts/claude-mem-heal.ps1 -Severity Error"` clean.
- [ ] `bash -n scripts/claude-mem-heal.sh` parse clean.
- [ ] PowerShell AST parse of `scripts/claude-mem-heal.ps1` clean.
- [ ] Manual repro on user's Windows machine: pre-fix `Test-Path marketplaces/thedotmack` = false; post-fix = true; junction target = `thedotmack-claude-mem`.

## Closing

- [ ] verification.md filled (before/after manual evidence, bats output, junction `Get-Item .LinkType` confirmation).
- [ ] PR opened referencing `specs/BUG-012-claude-mem-marketplace-junction/`.
- [ ] PR body documents that the printf symptom is a Windows secondary; the underlying discovery failure is cross-OS.
- [ ] Post-merge: archive `specs/BUG-012-...` to `specs/archive/`.
- [ ] Post-merge: vault `11-tasks.md` entry (created retroactively for traceability — small-scope bugfix exception per AGENTS.md).
- [ ] Post-merge: vault lesson "plugin discovery: marketplace dir name follows GitHub repo, not declared `name` field — always heal both paths".
