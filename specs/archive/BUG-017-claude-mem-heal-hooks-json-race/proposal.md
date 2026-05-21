---
id: "BUG-017-claude-mem-heal-hooks-json-race"
type: spec
status: implementing
created: "2026-05-21"
tags: [spec, proposal, claude-mem, heal, cross-os-parity, pipe-race]
template_version: "1.0"
---

# BUG-017-claude-mem-heal-hooks-json-race

## Why

BUG-016 (PR #83 merged 2026-05-21) closed the EPIPE pipe-race for claude-mem's `.mcp.json`. The exact same race pattern (`{ printf; ls; printf; } | while ... break`) exists in `plugin/hooks/hooks.json` across all 6 hooks (Setup, SessionStart x2, UserPromptSubmit, PostToolUse, PreToolUse, Stop). BUG-016 explicitly deferred hooks.json with "future BUG-017 could mirror this if upstream stays unfixed". Empirically the user hit `UserPromptSubmit operation blocked by hook: printf: write error: Permission denied` again minutes after BUG-016 merged — proving the deferral was wrong: the user-visible symptom is identical, only the affected surface differs. Open this same session.

## What

Extend `claude-mem-heal.{sh,ps1}` with a new `heal_hooks_json` / `Repair-HooksJson` function that walks both `cache/<version>/hooks/hooks.json` and `marketplace/.../plugin/hooks/hooks.json` (paths Claude Code's plugin loader honours). For each, apply a minimal literal substitution: `break; }; done` -> `}; done | head -n1`. This keeps the loop running to completion, then `head -n1` consumes the first output. Loops no longer break early; producers no longer EPIPE.

Idempotent: subsequent runs skip cleanly when the broken pattern is absent (already patched).

## Out of scope

- **Full hooks.json rewrite to a canonical template.** Six distinct command tails (start, context, session-init, observation, file-context, summarize) would need to be reproduced exactly. The minimal substitution preserves each hook's tail bit-for-bit.
- **Upstream PR to thedotmack/claude-mem.** Already filed as #2607. This local heal is defense-in-depth; when upstream lands a real fix, the heal becomes a no-op (no broken pattern to detect).
- **Replacing Repair-McpJson with a similar minimal patcher.** BUG-016 already shipped the full-rewrite approach for .mcp.json; not refactoring here.

## Risks / open questions

- **Risk: idempotent re-runs still trigger heal on `/plugin update` reverts.** Mitigation: heal runs at every SessionStart via claude-session-start.{sh,ps1}; if upstream re-writes hooks.json with the broken pattern, next session re-patches it. Expected and desired behaviour.
- **Risk: substitution matches an unrelated `break; }; done` in some future hooks.json variant.** Mitigation: the literal `break; }; done` sequence is unique to the cascade-pipe pattern; doesn't appear in normal JSON content. Low collision risk.
- **Risk: PSScriptAnalyzer em-dash regression (BUG-014's CI fail).** Mitigation: ASCII-only check pre-commit; same lint rule applies.

## Acceptance criteria

- [ ] `scripts/claude-mem-heal.sh::heal_hooks_json` defined; walks `<dir>/hooks/hooks.json` and `<dir>/plugin/hooks/hooks.json`; substitutes literal `break; }; done` -> `}; done | head -n1`; idempotent.
- [ ] `scripts/claude-mem-heal.ps1::Repair-HooksJson` equivalent on Windows.
- [ ] Both heal scripts log a single message per patched file, including the count of hooks transformed (`7 hook(s)` for the canonical v13.x file).
- [ ] `tests/setup-linux.bats`: 3 new parity asserts (function defined cross-OS, literal substitution present, hooks.json walk paths present, BUG-017 reference).
- [ ] `bash -n` clean; PowerShell AST + PSScriptAnalyzer clean; ASCII-only on the `.ps1`.
- [ ] Empirical on user's Windows: 2 distinct hooks.json files (cache + marketplace-via-junction) patched on first heal run; second run silent (idempotent).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § BUG-017 entry.
- Predecessor: BUG-016 (PR #83) — same pattern fix applied to `.mcp.json`.
- Upstream: [thedotmack/claude-mem#2607](https://github.com/thedotmack/claude-mem/issues/2607) — root cause documentation; Option A (`head -n1`) is what this PR applies locally.
- Pattern: BUG-016's lesson generalised — heal scripts must patch ALL surfaces affected by an upstream bug class, not just the first surface discovered.
