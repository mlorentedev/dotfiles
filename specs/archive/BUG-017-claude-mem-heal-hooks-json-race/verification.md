---
tags: [spec, verification, claude-mem, heal]
created: "2026-05-21"
---

# Verification - BUG-017-claude-mem-heal-hooks-json-race

## Evidence

- `scripts/claude-mem-heal.sh::heal_hooks_json` defined (lines ~127-145), walked from `heal_dir`.
- `scripts/claude-mem-heal.ps1::Repair-HooksJson` defined (lines ~173-195), walked from `Repair-PluginDir`.
- Literal substitution `break; }; done` -> `}; done | head -n1` present in both heal scripts.
- `tests/setup-linux.bats`: 3 new asserts in the BUG-017 block.

## Empirical (2026-05-21, user's Windows daily-driver, post-BUG-016 merge)

```
PS> pwsh -NoProfile -File scripts/claude-mem-heal.ps1 -VerboseOutput
[claude-mem-heal] .mcp.json already healthy: ...cache/.../13.3.0/.mcp.json
[claude-mem-heal] patched hooks.json (BUG-017, 7 hook(s) -> head -n1 race-free form): ...cache/.../13.3.0/hooks/hooks.json
[claude-mem-heal] zod present in ...
[claude-mem-heal] legacy marketplace path already present: ...
[claude-mem-heal] .mcp.json already healthy: ...marketplaces/thedotmack/plugin/.mcp.json
[claude-mem-heal] patched hooks.json (BUG-017, 7 hook(s) -> head -n1 race-free form): ...marketplaces/thedotmack/plugin/hooks/hooks.json
[claude-mem-heal] .mcp.json already healthy: ...marketplaces/thedotmack-claude-mem/plugin/.mcp.json
[claude-mem-heal] hooks.json already healthy: ...marketplaces/thedotmack-claude-mem/plugin/hooks/hooks.json (same content as thedotmack/plugin via junction)
```

- 14 hooks (7 cache + 7 marketplace) patched.
- Idempotent: second run silent.
- Post-patch `grep -c 'break; }; done' <file>` -> 0; `grep -c 'head -n1' <file>` -> 7.

## Decisions

- **Minimal substitution** over full rewrite: 6 hooks have different command tails (start, context, session-init, etc.) -- preserving them bit-for-bit is cleaner.
- **Walk both `hooks/` AND `plugin/hooks/`**: cache layout omits the `plugin/` subdir; marketplace layout includes it. Try both.
- **Same `break; }; done` literal in both .sh and .ps1**: cross-OS parity, identical patch outcome.

## Lesson candidate

"When a bug class spans multiple surfaces of an upstream system, the heal must patch ALL surfaces in the same PR. BUG-016 deferred hooks.json; BUG-017 was needed minutes later because the same user hit the same race on a different surface."

## Archive checklist
- [ ] Set status: archived post-merge.
- [ ] Move to specs/archive/.
- [ ] Tick vault entry with PR link.
