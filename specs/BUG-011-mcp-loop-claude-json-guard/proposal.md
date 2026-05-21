---
id: "BUG-011-mcp-loop-claude-json-guard"
type: spec
status: draft
created: "2026-05-20"
tags: [spec, proposal, claude-json, truncate-guard, mcp, cross-os-parity]
template_version: "1.0"
---

# BUG-011-mcp-loop-claude-json-guard

## Why

The user ran `setup-windows.ps1` and `~/.claude/.claude.json` was truncated again, forcing re-authentication in every project. BUG-004 (PR #57) added `Backup-AndRestoreClaudeJson` / `backup_and_restore_claude_json` snapshot+restore around `claude plugin install` — but the same upstream truncation bug (`anthropics/claude-code#59870`) fires on **every** Claude CLI invocation that goes through the deserialize-modify-serialize cycle. The MCP registration loop runs `claude mcp get` + `claude mcp add` for ~9 servers per setup (≈18 unwrapped invocations), and `claude plugin list` is called once before the plugin install loop. None of these are wrapped. The recurrence is the natural consequence: BUG-004 fixed one call site, the others are still hot.

## What

Wrap **every** Claude CLI invocation in both `setup-linux.sh` and `setup-windows.ps1` with the existing snapshot/restore helper, per-call (not per-loop, to preserve legitimate state additions on success and only restore on >50% size drop). Specifically:

1. `setup-linux.sh`: relocate `snapshot_claude_json` / `restore_claude_json_if_truncated` ABOVE the MCP registration block (currently defined after it). Wrap each MCP loop iteration around `claude mcp get` + `mcp add`. Wrap the `claude plugin list` pre-loop call.
2. `setup-windows.ps1`: wrap each MCP `foreach` iteration around `claude mcp get` + `mcp add` with `Backup-AndRestoreClaudeJson`. Wrap the `claude plugin list` pre-loop call.
3. Update the BUG-004 comment block in both scripts to reflect the broader scope ("every Claude CLI call site is guarded", not just "plugin install").
4. Add bats parity assertions in `tests/setup-linux.bats` + `tests/setup-windows.bats` so future call sites added without the guard fail CI.

After this PR, no `claude <subcommand>` call in either setup script executes without a snapshot in place.

## Out of scope

- The upstream fix in `anthropics/claude-code#59870` itself — out of repo control (already filed via SDD-022 cross-issue commentary).
- Wrapping `claude` calls in **runtime** scripts (e.g. `claude-session-start.{sh,ps1}`) — those are read-only canaries (size check), not setup-time mutators. They detect the bug; they do not mutate `.claude.json`. Out of scope here.
- Refactoring `Backup-AndRestoreClaudeJson` / `snapshot_claude_json` themselves — the helpers are correct and unchanged. We only re-wrap more call sites.
- Adding new MCP servers or new plugins. The list is frozen for this PR; the only diff is structural (guard wrapping).

## Risks / open questions

- **Risk: per-call snapshot+restore adds ~9 × file copy operations to setup runtime.** Mitigation: the file is small (≤75 KB); even 18 snapshots cost ~1.5 MB of temp I/O and <1 s wall time. Acceptable.
- **Risk: a legitimate `mcp add` increases the file size, snapshot/restore could mask later truncation in the same run.** Mitigation: per-call wrap (chosen via user Q1) snapshots BEFORE each call and restores AFTER that single call, so each iteration is independently protected. A successful `mcp add` that adds 200 bytes is not >50% smaller than its own snapshot → no spurious restore.
- **Risk: the `mcp get` pre-check (read-only by API) might NOT trigger the upstream bug, making its wrapping redundant.** Mitigation: BUG-004's comment block explicitly says "every `claude plugin install` writes to `.claude.json`" — empirically, any CLI invocation goes through the same serializer. Cost of an extra snapshot per iteration is negligible; cost of skipping it and discovering a third recurrence path is much higher.
- **Risk: the bats assertions are pattern-based grep checks** (text-level), not behavioral tests. They lock the *presence* of the wrap, not its correctness. Mitigation: pattern matches the BUG-004 family — same level of enforcement, same caveats; combined with the upstream bug already empirically observed, this is sufficient defense-in-depth.

## Acceptance criteria

- [ ] `setup-linux.sh`: `snapshot_claude_json` is defined BEFORE the MCP registration block (lexical order, no forward reference).
- [ ] `setup-linux.sh`: every `claude mcp get` / `claude mcp add` / `claude plugin list` / `claude plugin install` call has a `snapshot_claude_json` within 5 lines above AND a `restore_claude_json_if_truncated` within 10 lines below.
- [ ] `setup-windows.ps1`: every `claude mcp get` / `claude mcp add` / `claude plugin list` / `claude plugin install` call is inside a `Backup-AndRestoreClaudeJson -Action { ... }` scriptblock.
- [ ] `tests/setup-linux.bats` and `tests/setup-windows.bats`: new parity assertions fail if the MCP loop or `plugin list` is added without the guard.
- [ ] `bats tests/setup-linux.bats tests/setup-windows.bats` green (no regressions in existing 670+ assertions).
- [ ] `shellcheck --severity=error` clean on `setup-linux.sh`.
- [ ] `pwsh -Command "Invoke-ScriptAnalyzer -Path setup-windows.ps1 -Severity Error"` clean (matches CI gate).
- [ ] verification.md ships with commit hashes + test output excerpts.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` BUG-011 entry (this PR's "vault gate").
- Predecessor: BUG-004 (PR [#57](https://github.com/mlorentedev/dotfiles/pull/57)) — established the snapshot/restore pattern for `claude plugin install`.
- Sibling: SDD-021 (PR #56) — session-start canary that detects truncation; this PR prevents it at the source.
- Upstream: `anthropics/claude-code#59870` (filed via SDD-022 cross-issue commentary).
- Pattern: `00_meta/patterns/fix-small-debt.md` (audit all call sites of a vulnerable API when patching one).
- Vault lesson (post-merge): `90-lessons.md` "Incident → guard pattern" — extend with "when guarding one CLI call site, audit ALL call sites of the same vulnerable CLI in the same PR".
