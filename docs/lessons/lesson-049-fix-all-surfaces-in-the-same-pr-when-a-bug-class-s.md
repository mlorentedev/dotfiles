---
id: lesson-049-fix-all-surfaces-in-the-same-pr-when-a-bug-class-s
type: lesson
status: active
created: "2026-05-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 049: Fix ALL surfaces in the same PR when a bug class spans multiple call sites

**Context:** Today's claude-mem upstream bug cascade. BUG-016 fixed `.mcp.json` cascade-pipe EPIPE race but deferred `hooks.json` as "future BUG-017". Minutes after BUG-016 merged, user hit hooks.json fail → BUG-017 had to ship. Then BUG-017 fixed the pipe but UserPromptSubmit still blocked because hook command lacked `{"continue":true}` directive → BUG-018 narrow (only session-init). Minutes later, Stop hook failed 9x in a row → BUG-018 extended via regex to all 5 hooks. Three deferrals, ~30 min user pain each.
**Problem:** When patching a bug class affecting multiple call sites of the same upstream system, "fix one surface and ship" creates a guaranteed cascade. Each subsequent surface fail is a separate ticket, separate context-switch, separate ~30min of user-visible breakage. The cost of audit-all-surfaces-once is low (10-20 min grep + apply); the cost of skipping is unbounded.
**Solution:** When discovering a bug pattern (e.g. `break; }; done` cascade race; missing `{"continue":true}` directive), the SAME PR that fixes the first surface MUST: (a) `grep -E '<broken-pattern>' <relevant-files>` to enumerate every callsite; (b) apply the same fix or regex-substitute across all of them; (c) bats parity asserts that lock the substitution count. Companion to BUG-011's "audit all call sites of vulnerable upstream API" lesson — applies the same principle one layer down (pattern instead of API).
**Tags:** `#bug-class` `#audit-discipline` `#claude-mem` `#regex-substitution` `#cascade-cost`
