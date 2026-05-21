---
id: "BUG-016-claude-mem-heal-v13-refresh"
type: spec
status: implementing
created: "2026-05-21"
tags: [spec, proposal, claude-mem, heal, cross-os-parity]
template_version: "1.0"
---

# BUG-016-claude-mem-heal-v13-refresh

## Why

`scripts/claude-mem-heal.{sh,ps1}::Repair-McpJson` was authored against v12.7.4's broken `${_R%/}` pattern (PR #57, 2026-05-19). v13.0.0+ ships a different broken pattern — `sh -c` invoking a cascading-printf pipe that triggers the EPIPE race documented in [thedotmack/claude-mem#2607](https://github.com/thedotmack/claude-mem/issues/2607) and reported in this dotfiles session as `/mcp Failed to reconnect ... -32000`. Empirically the heal silently no-oped against v13.3.0 today: user hit the MCP failure AFTER BUG-014 (PR #75) restored the install AND BUG-015 (PR #81) shipped the detection layer. The heal needs to detect the v13.x signature AND replace the broken `.mcp.json` with a race-free form.

## What

After this PR, on every Claude Code session start (via `claude-session-start.{sh,ps1}` → `claude-mem-heal.{sh,ps1}`):

1. The heal detects BOTH the legacy v12.7.4 `${_R%/}` literal AND the v13.x signature (`"sh"` + `while IFS=` content together).
2. If either signature is present, the heal overwrites `.mcp.json` with a canonical race-free template: the same v13.x cascade-style resolver but piping the consumer's matches through `done | head -n1` instead of using `done` with an inner `break`. This consumes the entire producer pipe (no leftover writes → no EPIPE).
3. Idempotent: subsequent runs see no v12 or v13 signature in the patched file → silent skip.

Empirical validation on user's daily-driver Windows during implementation: 3 affected `.mcp.json` files (cache 13.3.0, marketplace junction, marketplace canonical) all patched on first heal run; second run silent (no signatures left to detect).

## Out of scope

- **`hooks.json` patches.** The same pipe-race pattern is in the 6 hooks (Setup, SessionStart, UserPromptSubmit, PostToolUse, PreToolUse, Stop) but each has a different command tail (`bun-runner.js worker-service.cjs hook ...`) and 6× the substitution surface. Real fix is upstream (#2607 Option A); BUG-015 already ships the detection layer. A future BUG-017 could mirror the `head -n1` patch into the heal for hooks.json — opens if upstream stays unfixed >2 weeks.
- **Upstream version pinning.** We do not pin claude-mem to a specific version; the heal must work against any v12.x or v13.x install.
- **Reverting to v10.6.3 simple-form template.** That form relies exclusively on `${CLAUDE_PLUGIN_ROOT}` being set by Claude Code per-plugin context. Empirically Claude Code does NOT always set it (especially for the MCP server launch path), so the cascade form is more robust.

## Risks / open questions

- **Risk: the v13.x signature regex catches third-party `.mcp.json` files that happen to use a similar pattern.** Mitigation: detection runs only against `.mcp.json` files in claude-mem's specific install paths (cache + marketplace dirs), not vault-wide.
- **Risk: a future claude-mem v14.x ships yet another pattern.** Mitigation: documented behaviour — heal is bug-version-aware; new versions get new detection rules (BUG-NNN per new pattern, same heal file).
- **Risk: the race-free form still fails when CLAUDE_PLUGIN_ROOT is unset AND cache is empty AND marketplace dirs missing.** Mitigation: that's the genuine "claude-mem not installed" case; the trailing `[ -n "$_P" ] || { ... exit 1; }` correctly surfaces the failure as a one-line stderr (no race involved).
- **Open question: does the Claude Code MCP loader correctly expand `${CLAUDE_CONFIG_DIR:-$HOME/.claude}` and `${PLUGIN_ROOT:-}`?** Empirical answer: yes — the original v13.x form uses these same expansions, only the pipe pattern was broken. Verified by the heal-then-`/mcp` cycle on user's machine working post-patch.

## Acceptance criteria

- [ ] `scripts/claude-mem-heal.sh::heal_mcp_json` detects both v12.7.4 (`${_R%/}` literal) and v13.x signatures and replaces with the head-n1 cascade form.
- [ ] `scripts/claude-mem-heal.ps1::Repair-McpJson` does the same on Windows.
- [ ] The replacement template contains `done | head -n1` (NOT `done` with `break`) — verifiable by grep against the heal source.
- [ ] Both heal scripts reference `BUG-016` and `claude-mem#2607` in their comments for traceability.
- [ ] `tests/setup-linux.bats`: 3 new asserts — detection signature parity, head-n1 form parity, BUG-016 + #2607 reference parity.
- [ ] `Invoke-ScriptAnalyzer -Settings .PSScriptAnalyzerSettings.psd1 -Severity Error,Warning scripts/claude-mem-heal.ps1` clean.
- [ ] `bash -n scripts/claude-mem-heal.sh` clean.
- [ ] Empirical: running `claude-mem-heal.{sh,ps1}` on a machine with v13.3.0 patches all 3 `.mcp.json` files (cache + 2 marketplace paths); second run is silent (idempotent).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § BUG-016 entry.
- Predecessor: BUG-012 (PR #70, 2026-05-20) — established the heal junction-creation pattern.
- Predecessor: PR #57 — original `Repair-McpJson` authored against v12.7.4 `${_R%/}`.
- Upstream: [thedotmack/claude-mem#2607](https://github.com/thedotmack/claude-mem/issues/2607) — root cause documentation + 3 fix options. This PR applies Option A locally to the MCP server case; hooks.json case is upstream territory.
- Sibling: BUG-015 (PR #81) — detection-only layer that catches when path resolution itself fails. Complementary to BUG-016's heal-time fix.
