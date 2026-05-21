---
tags: [spec, verification, claude-mem, heal]
created: "2026-05-21"
---

# Verification - BUG-018-userpromptsubmit-continue-directive

## Evidence

- claude-mem-heal.sh::heal_hooks_json -- sed with regex capture for any `hook claude-code <event>"`.
- claude-mem-heal.ps1::Repair-HooksJson -- PowerShell `-replace` with equivalent regex; reports counts.
- tests/setup-linux.bats BUG-018 parity block.

## Empirical (2026-05-21)

1. BUG-016 merged: .mcp.json fixed.
2. BUG-017 merged: hooks.json cascade race fixed; error changed from `printf: write error` to `No stderr output`.
3. Narrow BUG-018 manual patch (session-init only): UserPromptSubmit works (ping/pong) but Stop fails 9 times.
4. Regex BUG-018 patch (this commit): all 5 hooks patched; loop resolved.

Post-patch grep on user's hooks.json:
- `continue":true` count: 6 (5 patched + 1 pre-existing in SessionStart-start)
- Unpatched `hook claude-code <X>"` terminators: 0

## Lint

- bash -n OK
- PowerShell AST parse clean
- PSScriptAnalyzer clean
- ASCII-only zero non-ASCII

## Decisions

- **Regex capture over per-event substitutions:** maintainable if upstream adds a 6th event.
- **Setup hook left untouched:** different terminator, not on user hot path.
- **`2>/dev/null` to silence stderr:** harmless since bun-runner writes to stdout.

## Lesson candidate

"When a bug class affects N callsites of an upstream system, the heal must patch ALL N in the same PR. Each deferral cost ~30 min of user pain today: BUG-016 (.mcp.json) -> deferred hooks.json (BUG-017) -> deferred continue-directive (BUG-018 narrow) -> all 5 hooks (BUG-018 extended). Be proactive, not reactive."

## Archive checklist
- [ ] status: archived post-merge
- [ ] move to specs/archive/
- [ ] tick vault entry with PR link
- [ ] append lesson to 90-lessons.md
