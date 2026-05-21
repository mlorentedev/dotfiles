---
tags: [spec, tasks, claude-mem, heal, cross-os-parity]
created: "2026-05-21"
---

# Tasks - BUG-016-claude-mem-heal-v13-refresh

## Setup

- [x] Branch `fix/BUG-016-claude-mem-heal-v13-refresh` (off main).
- [x] Spec scaffolded via `init-spec.ps1` (vault entry exists).
- [x] Empirical confirmation user's v13.3.0 install on Windows daily-driver has all 3 `.mcp.json` files in the broken v13.x cascade form; current heal silently no-ops.

## Implementation (TDD order)

### Tests first

- [ ] `tests/setup-linux.bats`: assert both heal scripts grep for `while IFS=` (v13.x signature).
- [ ] `tests/setup-linux.bats`: assert both heal scripts contain `head -n1` in their replacement template.
- [ ] `tests/setup-linux.bats`: cross-OS parity assert for BUG-016 + claude-mem#2607 references.
- [ ] Run bats — should FAIL (red).

### Implementation

- [x] `scripts/claude-mem-heal.sh::heal_mcp_json`: extend detection to OR-of v12.7.4 + v13.x signature; replace template with race-free cascade-with-head-n1 form; log message distinguishes which version was patched.
- [x] `scripts/claude-mem-heal.ps1::Repair-McpJson`: equivalent on Windows.
- [x] Comments in both files reference BUG-016 + thedotmack/claude-mem#2607 for traceability.
- [ ] Run bats — assertions GREEN.

### Lint + cross-check

- [x] `bash -n scripts/claude-mem-heal.sh` → OK.
- [x] PowerShell AST parse on `scripts/claude-mem-heal.ps1` clean.
- [x] `Invoke-ScriptAnalyzer -Settings .PSScriptAnalyzerSettings.psd1 -Severity Error,Warning` clean.
- [x] ASCII-only check on `claude-mem-heal.ps1` (zero non-ASCII).
- [x] Empirical run on user's Windows: heal patched 3 `.mcp.json` files (cache 13.3.0 + 2 marketplace paths) on first run with exit 0; re-run silent (idempotent).

## Closing

- [x] `verification.md` filled with empirical evidence (heal log + post-patch file fragments).
- [ ] PR opened referencing `specs/BUG-016-claude-mem-heal-v13-refresh/`.
- [ ] Post-merge: archive spec to `specs/archive/`.
- [ ] Post-merge: tick vault `11-tasks.md` BUG-016 entry → ✓ with PR link.
- [ ] Post-merge: append lesson candidate to `90-lessons.md` — "heal scripts must be versioned against the upstream bug class they paper over; when upstream's bug pattern changes, the heal's detection regex MUST be refreshed in the same PR that discovers the new pattern".
