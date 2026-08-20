---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - MEMORY-001-cross-agent-session-bridge

> TDD order. Implements ADR-014. This PR ships the **ADR + spec (design)**; the implementation tasks below execute in a follow-up.

## Setup

- [x] Vault gate entry (MEMORY-001-mirror) in `11-tasks.md`
- [x] **ADR-014** written + ratified (`docs/adr/adr-014-cross-agent-session-memory-bridge.md`)
- [x] `proposal.md` complete; acceptance criteria testable

## Implementation (TDD)

- [x] Failing bats `tests/session-handoff.bats` (fixture `SessionEnd` payload) → RED (AC1)
- [x] `scripts/session-handoff.sh`: parse stdin JSON (`jq`), locate the project MEMORY.md, **archive its `## Session Handoff` block** to `00_meta/sessions/<date>-<project>-claude.md`; resilient no-op on trivial/missing/malformed input → GREEN (AC1). *(Design: the agent authors via /handoff; the hook persists — see ADR-014.)*
- [x] bats: no handoff block / no MEMORY.md / empty stdin → no record, clean exit (AC2)
- [x] Wire the Claude `SessionEnd` hook in `ai/claude/settings.json` (`__SESSION_END_COMMAND__`) + the setup merge policy (substitution + `.hooks.SessionEnd` merge, cross-OS `setup-{linux,windows}`) + `claude-settings-template.bats` asserts (AC3)
- [x] `scripts/session-handoff.ps1` Windows port (ASCII-only functional mirror) + deployed by `setup-windows.ps1` (AC4 — Windows runtime validation remains empirical)

## Closing

- [ ] Every AC covered by a test
- [ ] `features.json` present
- [ ] `verification.md` filled
- [ ] PR opened referencing this spec folder

## Follow-up tickets (downstream, per ADR-014)

- [ ] Per-agent session-end hooks: OpenCode (`opencode.jsonc`), agy (`~/.gemini/settings.json`), Copilot — each needs its session-end surface validated (some Windows-empirical).
- [ ] `00_meta/sessions/` archival / rotation hygiene.
