---
tags: [spec, tasks, claude-json, truncate-guard]
created: "2026-05-20"
---

# Tasks - BUG-011-mcp-loop-claude-json-guard

## Setup

- [x] Branch: `fix/BUG-011-mcp-loop-claude-json-guard` (off main).
- [x] Vault entry in `11-tasks.md`.
- [x] Spec scaffolded via `scripts/init-spec.ps1`.

## Implementation (TDD order)

### Tests first

- [ ] `tests/setup-linux.bats`: add assertion that the MCP loop wraps `claude mcp add` / `mcp get` with `snapshot_claude_json` + `restore_claude_json_if_truncated`.
- [ ] `tests/setup-linux.bats`: add assertion that `claude plugin list` is wrapped with the snapshot/restore pair.
- [ ] `tests/setup-windows.bats`: add assertion that the MCP loop foreach body is inside `Backup-AndRestoreClaudeJson`.
- [ ] `tests/setup-windows.bats`: add assertion that `claude plugin list` invocation is inside `Backup-AndRestoreClaudeJson`.
- [ ] Cross-OS parity assertion: both scripts wrap the MCP loop (mirrors the existing BUG-004 parity assert at `tests/setup-linux.bats:216`).
- [ ] Run bats — assertions should FAIL (red).

### Implementation

- [ ] `setup-linux.sh`: move `snapshot_claude_json` and `restore_claude_json_if_truncated` function definitions ABOVE the MCP registration block (currently at lines 606-630, needs to be before line 540).
- [ ] `setup-linux.sh`: wrap each MCP loop iteration body (`claude mcp get` + `claude mcp add`) with `_snap=$(snapshot_claude_json)` / `restore_claude_json_if_truncated "$_snap"`.
- [ ] `setup-linux.sh`: wrap `installed_plugins=$(claude plugin list ...)` with the same snapshot/restore pair.
- [ ] `setup-linux.sh`: update the BUG-004 comment block to reference BUG-011 (extended scope).
- [ ] `setup-windows.ps1`: wrap the MCP foreach iteration body with `Backup-AndRestoreClaudeJson -Action { ... }` around both `claude mcp get` and `claude mcp add`.
- [ ] `setup-windows.ps1`: wrap `$installedPlugins = ... claude plugin list ...` inside `Backup-AndRestoreClaudeJson`.
- [ ] `setup-windows.ps1`: update the BUG-004 comment block to reference BUG-011 (extended scope).
- [ ] Run bats — assertions now PASS (green).

### Lint + cross-check

- [ ] `shellcheck --severity=error scripts/*.sh setup-linux.sh` clean.
- [ ] `pwsh -Command "Invoke-ScriptAnalyzer -Path setup-windows.ps1 -Severity Error"` clean.
- [ ] Re-run full bats suite: `bats tests/` — no regressions (target green).

## Closing

- [ ] verification.md filled (commit hashes, bats output excerpts, before/after grep diff showing wrapped call sites).
- [ ] PR opened referencing `specs/BUG-011-mcp-loop-claude-json-guard/`.
- [ ] PR body notes: empirical confirmation pending until next clean-machine setup run; user can verify by running `setup-windows.ps1` on a machine with `.claude.json ≥ 10 KB` and confirming the file is unchanged (or restored from snapshot if upstream fires).
- [ ] Post-merge: tick BUG-011 in vault `11-tasks.md`, append PR link.
- [ ] Post-merge: `git mv specs/BUG-011-mcp-loop-claude-json-guard/ specs/archive/`.
- [ ] Post-merge: append lesson to `90-lessons.md` — rule: "when guarding one CLI call site, audit ALL call sites of the same vulnerable CLI in the same PR".
