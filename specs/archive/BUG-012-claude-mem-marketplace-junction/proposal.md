---
id: "BUG-012-claude-mem-marketplace-junction"
type: spec
status: draft
created: "2026-05-20"
tags: [spec, proposal, claude-mem, plugin-discovery, cross-os-parity]
template_version: "1.0"
---

# BUG-012-claude-mem-marketplace-junction

## Why

Discovered while diagnosing a `UserPromptSubmit operation blocked by hook` failure during the BUG-011 session. The `claude-mem@thedotmack` plugin (installed by `setup-{linux,windows}` ~line 428) ships hooks that look up its scripts at fallback path `~/.claude/plugins/marketplaces/thedotmack/plugin/scripts/` — the literal marketplace `name` declared in `marketplace.json`. Claude Code, however, clones the marketplace directory under the GitHub *repository* name `thedotmack-claude-mem/`. The fallback path therefore never resolves on machines where `CLAUDE_PLUGIN_ROOT` is unset/stale (the user's case), and the hook exits non-zero with `claude-mem: plugin scripts not found` (or its Git-Bash-on-Windows symptom `printf: write error: Permission denied`), blocking every `UserPromptSubmit`.

Same root cause silently neuters `scripts/claude-mem-heal.{sh,ps1}`: both hard-code `marketplaces/thedotmack/plugin` (sh:33, ps1:52), find the dir absent, and no-op — so the upstream `.mcp.json ${_R%/}` bug and missing-zod patches never reach the marketplace source.

The user's manual workaround was to delete the plugin, but `setup-windows.ps1:428` reinstalls it every run. Without a fix, the loop repeats.

## What

Defense in depth — two changes, both in `scripts/claude-mem-heal.{sh,ps1}`:

1. **Create the legacy marketplace junction/symlink** if missing. When `~/.claude/plugins/marketplaces/thedotmack/` does not exist but `~/.claude/plugins/marketplaces/thedotmack-claude-mem/` does, create a junction (Windows) / symlink (Linux) so the plugin's hardcoded fallback paths resolve. Idempotent: skip if either condition fails.
2. **Make the heal script's `MARKETPLACE_DIR` resolution path-aware**. Walk both `thedotmack/plugin` and `thedotmack-claude-mem/plugin` and heal whichever exists. The junction from step 1 effectively unifies these, but the path-aware walk is a backstop in case Claude Code changes the install naming again or the junction creation fails.

Tests: bats assertions in `tests/setup-linux.bats` + `tests/setup-windows.bats` that lock the presence of the junction-creation block and the path-aware healing in both heal scripts.

## Out of scope

- The upstream plugin's `hooks.json` hardcoded fallback — out of repo control; we paper over from our side.
- Removing `claude-mem@thedotmack` from `setup-{linux,windows}` plugin list — the user wants the plugin, just not the breakage.
- The `printf: write error: Permission denied` Git-Bash-on-Windows symptom — diagnosed as secondary (pipe-closed-during-write under Claude Code's hook sandbox). Fixing the discovery resolves the underlying `exit 1`; the printf symptom disappears as a consequence.
- Modifying `setup-{linux,windows}` to skip the plugin install when broken — too brittle; let the heal script run on session start and self-correct.

## Risks / open questions

- **Risk: Claude Code in a future version starts creating `marketplaces/thedotmack/` itself, colliding with the junction.** Mitigation: junction creation is gated on `marketplaces/thedotmack/` NOT existing. If a future Claude Code creates a real dir, we leave it alone.
- **Risk: junction on Windows requires the source directory to exist at junction-create time.** Mitigation: we only create the junction when `marketplaces/thedotmack-claude-mem/` exists. The whole block is skipped otherwise.
- **Risk: junction creation fails (permissions, anti-virus).** Mitigation: heal script always exits 0 and logs the failure; the user falls back to the path-aware heal (step 2). Plugin hook fallback is still broken in this case, but no regression vs current behavior.
- **Risk: bats assertions are pattern-based grep checks.** Same caveat as BUG-011 — they lock presence, not correctness. Acceptable for this defense-in-depth tier.

## Acceptance criteria

- [ ] `scripts/claude-mem-heal.sh`: new block creates `marketplaces/thedotmack` symlink → `marketplaces/thedotmack-claude-mem` when source exists and target missing. Idempotent on re-run.
- [ ] `scripts/claude-mem-heal.ps1`: equivalent block using `New-Item -ItemType Junction`.
- [ ] Both heal scripts walk both legacy and current marketplace paths for healing (path-aware backstop).
- [ ] `tests/setup-linux.bats`: assertion locks the junction-creation block in `claude-mem-heal.sh`.
- [ ] `tests/setup-windows.bats`: assertion locks the junction-creation block in `claude-mem-heal.ps1`.
- [ ] `shellcheck --severity=error scripts/claude-mem-heal.sh` clean.
- [ ] `pwsh -Command "Invoke-ScriptAnalyzer -Path scripts/claude-mem-heal.ps1 -Severity Error"` clean.
- [ ] `bash -n scripts/claude-mem-heal.sh` and PowerShell AST parse clean.
- [ ] verification.md ships with manual repro before/after for the user's current machine.

## References

- Sibling: BUG-011 (PR [#69](https://github.com/mlorentedev/dotfiles/pull/69)) — discovered this bug during BUG-011 hook-failure diagnosis.
- Predecessor: BUG-004 (PR #57) — established the snapshot/restore guard, but didn't address the plugin-discovery path mismatch.
- Upstream context: `thedotmack/claude-mem` plugin ships `hooks.json` with hardcoded fallback `marketplaces/thedotmack/plugin/scripts/...`.
- Pattern: defense-in-depth heal at session start (mirrors the `.mcp.json` + zod patches the heal scripts already apply).
