---
tags: [spec, verification, claude-mem, heal, cross-os-parity]
created: "2026-05-21"
---

# Verification - BUG-016-claude-mem-heal-v13-refresh

## Evidence (per acceptance criterion)

- **claude-mem-heal.sh detects + replaces v12+v13** → `scripts/claude-mem-heal.sh:66-104` (`heal_mcp_json` function). Detection via `grep -cF '${_R%/}'` OR `grep -cE '"sh".*"-c"|while IFS= read'`. Replacement template includes literal `done | head -n1`.
- **claude-mem-heal.ps1 same on Windows** → `scripts/claude-mem-heal.ps1:96-148` (`Repair-McpJson` function). Detection via `$content -match '\$\{_R%/\}'` OR (`'"sh"\s*,\s*\r?\n?\s*"args"'` AND `'while IFS='`).
- **`done | head -n1` in template** → grep both `claude-mem-heal.{sh,ps1}` for `head -n1` → present.
- **BUG-016 + #2607 references** → grep both files for `BUG-016` and `claude-mem#2607` → present.
- **3 new bats asserts** → `tests/setup-linux.bats:177-198` (BUG-016 block).

## Test status

### Pre-fix state (Windows daily-driver, 2026-05-21)

```
PS> cat ~/.claude/plugins/cache/thedotmack/claude-mem/13.3.0/.mcp.json | head -6
{
  "mcpServers": {
    "mcp-search": {
      "type": "stdio",
      "command": "sh",
      "args": [

PS> Get-Content ~/.claude/plugins/cache/thedotmack/claude-mem/13.3.0/.mcp.json -Raw | Select-String -Pattern 'while IFS=' -Quiet
True   # v13.x signature present

PS> pwsh -NoProfile -File ~/scripts/claude-mem-heal.ps1 -VerboseOutput | Select-String 'patched'
(nothing -- silent no-op, the bug)
```

### Post-fix empirical run

```
PS> pwsh -NoProfile -File scripts/claude-mem-heal.ps1 -VerboseOutput
[claude-mem-heal] patched .mcp.json (v13.x cascade -> head -n1 race-free form): C:\Users\Manu\.claude\plugins\cache\thedotmack\claude-mem\13.3.0\.mcp.json
[claude-mem-heal] zod present in C:\Users\Manu\.claude\plugins\cache\thedotmack\claude-mem\13.3.0
[claude-mem-heal] legacy marketplace path already present: C:\Users\Manu\.claude\plugins\marketplaces\thedotmack
[claude-mem-heal] patched .mcp.json (v13.x cascade -> head -n1 race-free form): C:\Users\Manu\.claude\plugins\marketplaces\thedotmack\plugin\.mcp.json
[claude-mem-heal] zod present in C:\Users\Manu\.claude\plugins\marketplaces\thedotmack\plugin
[claude-mem-heal] patched .mcp.json (v13.x cascade -> head -n1 race-free form): C:\Users\Manu\.claude\plugins\marketplaces\thedotmack-claude-mem\plugin\.mcp.json
[claude-mem-heal] zod present in C:\Users\Manu\.claude\plugins\marketplaces\thedotmack-claude-mem\plugin
$LASTEXITCODE = 0
```

All 3 `.mcp.json` files patched on first run. Second run silent (idempotent — no v12/v13 signature left).

### Lint results

- `bash -n scripts/claude-mem-heal.sh` → OK
- PowerShell AST `[Parser]::ParseFile` on `claude-mem-heal.ps1` → clean
- `Invoke-ScriptAnalyzer -Settings .PSScriptAnalyzerSettings.psd1 -Severity Error,Warning` → clean
- ASCII-only check on `claude-mem-heal.ps1` → zero non-ASCII chars

### Bats (post-CI)

To be confirmed after CI green.

## Decisions made during implementation

- **Cascade-with-head-n1 over v10.6.3 simple form**: the simple form (`${CLAUDE_PLUGIN_ROOT}/scripts/mcp-server.cjs`) only works when Claude Code sets `CLAUDE_PLUGIN_ROOT` per-plugin context. Empirically that's not always the case (the v13.x cascade ITSELF falls back to cache/marketplace dirs because CLAUDE_PLUGIN_ROOT is often unset for the MCP server launch). Keeping the cascade structure makes the heal robust to both states.
- **`done | head -n1` over `done` + `break`**: the upstream pattern uses `break` to exit early, leaving unconsumed producer writes that EPIPE. Replacing with `head -n1` lets the consumer drain the whole upstream pipe, then `head` takes the first match line. No leftover writes, no EPIPE.
- **NOT removing the `thedotmack/plugin` fallback path**: the BUG-012 legacy junction is still in place on user machines; removing it from the template would mean missing the BUG-012 fallback case. Both paths stay.
- **Single template for both v12 and v13 detection paths**: simpler than maintaining a v12-specific and v13-specific template. Whichever signature triggered the patch, the result is the same canonical form.

## Promotion candidates

- [x] **Lesson for `90-lessons.md`** — yes: "heal scripts must be versioned against the upstream bug class they paper over; when upstream's bug pattern changes, the heal's detection regex MUST be refreshed in the same PR that discovers the new pattern. Else the heal silently no-ops while users continue hitting the bug." Pairs with the existing BUG-012 lesson on heal walking real disk paths.
- [ ] ADR-worthy? **no** — tactical pattern refresh; ADR-007 (heal-at-session-start) covers the strategy.
- [ ] New pattern candidate? **possibly** — "incident → guard" pattern (existing) could be extended to "incident → guard → re-validate guard when upstream signature changes". Worth a discussion in `00_meta/patterns/` if BUG-017+ happens.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived` (post-merge).
- [ ] Folder moved to `specs/archive/BUG-016-claude-mem-heal-v13-refresh/`.
- [ ] Vault `11-tasks.md` BUG-016 entry ticked ✓ with PR link.
- [ ] Vault `90-lessons.md` lesson appended.
