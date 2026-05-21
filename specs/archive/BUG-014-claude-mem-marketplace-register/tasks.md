---
tags: [spec, tasks, claude-mem, plugin-discovery]
created: "2026-05-21"
---

# Tasks - BUG-014-claude-mem-marketplace-register

## Setup

- [x] Branch: `feat/BUG-014-claude-mem-marketplace-register` (off main).
- [x] Spec scaffolded via `init-spec.ps1` (vault gate satisfied — entry in `10_projects/dotfiles/11-tasks.md`).
- [x] Empirical pre-flight on user's Windows machine: `claude plugin marketplace add thedotmack/claude-mem` + `claude plugin install claude-mem@thedotmack` → exit 0, idempotent, no `.claude.json` truncation. Command syntax confirmed against `claude plugin marketplace add --help`.

## Implementation (TDD order)

### Tests first

- [ ] `tests/setup-linux.bats`: assertion locks the `claude plugin marketplace add thedotmack/claude-mem` line inside a `backup_and_restore_claude_json` wrap, positioned BEFORE the plugin install loop.
- [ ] `tests/setup-windows.bats`: equivalent assertion for `Backup-AndRestoreClaudeJson` wrapping the marketplace-add call.
- [ ] `tests/healthcheck.bats`: assertion locks a new sec 4 check that greps `installed_plugins.json` for `claude-mem@thedotmack`.
- [ ] `tests/healthcheck-ps1.bats`: equivalent assertion via `Select-String -SimpleMatch`.
- [ ] Run all four bats files — new assertions should FAIL (red).

### Implementation

- [ ] `setup-linux.sh`: insert the marketplace-add block immediately before the existing plugin install loop. Same `backup_and_restore_claude_json` snapshot pattern as MCP loop iterations (BUG-011). Output: one INFO line on first add ("Registered marketplace: thedotmack/claude-mem"); silent on subsequent runs.
- [ ] `setup-windows.ps1`: equivalent block with `Backup-AndRestoreClaudeJson` wrapping. Single `& claude plugin marketplace add thedotmack/claude-mem 2>&1` inside the scriptblock.
- [ ] `scripts/healthcheck.sh` sec 4: add `installed_plugins.json` membership check. Use `grep -F` against the literal `claude-mem@thedotmack`. PASS on hit, FAIL on miss. Keep the existing junction check immediately after, but reword its label to mark it secondary ("BUG-012 secondary diagnostic").
- [ ] `scripts/healthcheck.ps1` sec 4: equivalent via `Select-String -SimpleMatch 'claude-mem@thedotmack' -Quiet`.
- [ ] Run bats — assertions now PASS (green).

### Lint + cross-check

- [ ] `shellcheck --severity=error setup-linux.sh scripts/healthcheck.sh` clean.
- [ ] `Invoke-ScriptAnalyzer -Path setup-windows.ps1,scripts/healthcheck.ps1 -Severity Error` clean.
- [ ] `bash -n setup-linux.sh` + `bash -n scripts/healthcheck.sh` parse clean.
- [ ] PowerShell AST parse of both `.ps1` files clean.
- [ ] Manual repro on user's Windows machine: pre-fix `installed_plugins.json` had 10 plugins (no thedotmack); post-fix has 15 incl. `claude-mem@thedotmack`. Captured in `verification.md`.

## Closing

- [ ] `verification.md` filled with manual repro evidence (marketplace list before/after, installed_plugins.json before/after, healthcheck sec 4 output transition).
- [ ] PR opened referencing `specs/BUG-014-claude-mem-marketplace-register/`. PR body explicitly calls out cross-OS parity and the BUG-012 healthcheck false-positive that this PR closes.
- [ ] Post-merge: archive `specs/BUG-014-...` to `specs/archive/`.
- [ ] Post-merge: append lesson to `10_projects/dotfiles/90-lessons.md` — "healthcheck assertion that checks proxy artifact (junction) instead of canonical state (installed_plugins.json) is itself a bug — every health check must validate the end-state, not a side-effect that can exist without it".
- [ ] Post-merge: update vault `11-tasks.md` BUG-014 entry → ✓ with PR link.
