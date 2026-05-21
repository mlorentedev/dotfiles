---
id: "BUG-014-claude-mem-marketplace-register"
type: spec
status: implementing
created: "2026-05-21"
tags: [spec, proposal, claude-mem, plugin-discovery, cross-os-parity]
template_version: "1.0"
---

# BUG-014-claude-mem-marketplace-register

## Why

`setup-{linux.sh,windows.ps1}` list `claude-mem@thedotmack` in the plugin install loop but **neither script ever registers the `thedotmack` marketplace first**. On a fresh machine `claude plugin marketplace list` shows only `claude-plugins-official` (the bundled default), so `claude plugin install claude-mem@thedotmack` cannot resolve `@thedotmack` and fails silently inside the wide `try{} catch{}` (`setup-windows.ps1:456-462`, `setup-linux.sh` plugin loop). Result: `installed_plugins.json` never contains `claude-mem@thedotmack`, the session-start hook does not fire, `/mem-search` is unavailable. The global `~/.claude/CLAUDE.md` declares claude-mem "Active by default in every session" — the install gap silently breaks that contract.

Empirically reproduced on Windows daily-driver 2026-05-21: 15-plugin install record contained zero `@thedotmack` entries. `claude plugin marketplace add thedotmack/claude-mem` followed by `claude plugin install claude-mem@thedotmack` succeeded (exit 0 both, idempotent on re-run, no `.claude.json` truncation under BUG-004 snapshot guard) and restored claude-mem in the next session. The fix command is upstream-supported and stable; setup scripts just never call it.

BUG-012 (PR #70) created a legacy junction filesystem-side but did not address marketplace registration. `claude-mem-heal.{sh,ps1}` only patches existing installs; it cannot bootstrap a marketplace. Healthcheck `[4/12] PASS: claude-mem marketplace legacy junction present (BUG-012)` is a **false positive** — it checks a proxy artifact (filesystem junction) instead of the canonical state (`installed_plugins.json` membership).

## What

1. **Both setup scripts** insert one new step BEFORE the existing plugin install loop: invoke `claude plugin marketplace add thedotmack/claude-mem`, wrapped with the existing `Backup-AndRestoreClaudeJson` / `backup_and_restore_claude_json` BUG-004/BUG-011 snapshot guard. The CLI is idempotent (exit 0 + `'thedotmack' already on disk` message on re-run), so no pre-check needed.
2. **Both healthcheck scripts** sec 4 add a **new** assertion: `claude-mem@thedotmack` is present in `~/.claude/plugins/installed_plugins.json`. The existing junction check (BUG-012) is preserved as a secondary fallback but explicitly labeled as such — install-state is now the primary FAIL signal.
3. **Bats parity** in `tests/setup-{linux,windows}.bats`: lock the marketplace-add line with snapshot guard wrapping. In `tests/healthcheck{,-ps1}.bats`: lock the new install-state assertion.

After this PR, a fresh `setup-{linux,windows}` run on a machine with no prior claude-mem state registers the marketplace, installs the plugin, and healthcheck reports PASS on canonical state (not a proxy).

## Out of scope

- Re-installing claude-mem when registered-but-uninstalled — that's claude-mem-heal's territory in a future evolution (separate ticket). This PR only handles the *first-time bootstrap* gap.
- Removing the BUG-012 legacy-junction check from healthcheck — it stays as secondary diagnostic (different failure class: install OK but plugin discovery still broken because junction missing).
- Generalising to "any `@xxx` marketplace" — only `thedotmack` is in the current plugin array. YAGNI; revisit when a second non-official marketplace appears.
- Modifying `claude-mem-heal.{sh,ps1}` — heal already handles the cases it should (cache patches, zod backfill, junction); the registration gap is exclusively setup-script territory.
- Touching `ai/claude/settings.json` `enabledPlugins` map — orthogonal to install state; SDD-002 merge policy preserves it.

## Risks / open questions

- **Risk: `claude plugin marketplace add` requires network at setup time.** Mitigation: wrapped in `try{} catch{}` so a transient network failure doesn't break setup — same pattern as the existing plugin install loop. Subsequent re-run resolves once network returns.
- **Risk: future Claude Code version changes the marketplace-add command syntax.** Mitigation: bats parity test grep-locks the literal command. If syntax breaks, CI fails loud + lesson captured.
- **Risk: bats assertions are pattern-based grep checks.** Same caveat as BUG-011/BUG-012 — they lock presence, not correctness. Acceptable for defense-in-depth tier.
- **Risk: healthcheck's new install-state assertion file-locks an upstream `installed_plugins.json` schema.** Mitigation: check uses `Select-String -SimpleMatch 'claude-mem@thedotmack'` / `grep -F`, not JSON schema parsing — survives Claude Code internal restructuring as long as the plugin name appears verbatim somewhere in the file.

## Acceptance criteria

- [ ] `setup-linux.sh` invokes `claude plugin marketplace add thedotmack/claude-mem` BEFORE the plugin install loop, wrapped with `backup_and_restore_claude_json`. Idempotent: a second setup run shows `Marketplace 'thedotmack' already on disk` and exits 0.
- [ ] `setup-windows.ps1` equivalent: same call wrapped with `Backup-AndRestoreClaudeJson`.
- [ ] `scripts/healthcheck.sh` sec 4 contains a new check: greps `installed_plugins.json` for `claude-mem@thedotmack`; FAIL on absence, PASS on presence. Existing junction check preserved as secondary.
- [ ] `scripts/healthcheck.ps1` sec 4: equivalent check via `Select-String -SimpleMatch`.
- [ ] `tests/setup-linux.bats`: assertion locks the `claude plugin marketplace add thedotmack/claude-mem` line wrapped with `backup_and_restore_claude_json`.
- [ ] `tests/setup-windows.bats`: equivalent assertion for the PowerShell side.
- [ ] `tests/healthcheck.bats` + `tests/healthcheck-ps1.bats`: assertions lock the new install-state check.
- [ ] `shellcheck --severity=error scripts/healthcheck.sh setup-linux.sh` clean.
- [ ] `Invoke-ScriptAnalyzer -Severity Error scripts/healthcheck.ps1 setup-windows.ps1` clean.
- [ ] verification.md ships with manual repro: pre-fix `marketplace list` shows only official; post-fix shows both + `installed_plugins.json` contains claude-mem@thedotmack.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § BUG-014 entry (2026-05-21).
- Predecessor: BUG-012 (PR #70) — fixed legacy junction filesystem-side; did not address registration. The PR-#70 `healthcheck` PASS message is the false-positive this PR closes.
- Defense-in-depth precedent: BUG-004 (PR #57) + BUG-011 (PR #69) — established `Backup-AndRestoreClaudeJson` / `backup_and_restore_claude_json` guards for every `claude` CLI call site. This PR adds one more wrapped call site.
- Upstream: `thedotmack/claude-mem` marketplace at `https://github.com/thedotmack/claude-mem`.
- Pattern: same incident → guard rule as BUG-011: every new vulnerable CLI invocation must be wrapped at all setup-script sites; never partial.
